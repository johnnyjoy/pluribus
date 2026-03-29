package memory

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"sort"
	"strings"

	"control-plane/internal/memorynorm"
	"control-plane/internal/similarity"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// PatternElevationConfig gates cluster elevation into a single dominant pattern row.
type PatternElevationConfig struct {
	Enabled                 bool    `yaml:"enabled"`
	MinReuseScore           float64 `yaml:"min_reuse_score"`
	MinDistinctContexts     int     `yaml:"min_distinct_contexts"`
	MinDistinctAgents       int     `yaml:"min_distinct_agents"`
	MinAuthority            int     `yaml:"min_authority"`
	MaxSupportingPatterns   int     `yaml:"max_supporting_patterns"`
	AuthorityElevationDelta int     `yaml:"authority_elevation_delta"`
	MergeJaccardMin         float64 `yaml:"merge_jaccard_min"`
	MinTagOverlapPair       float64 `yaml:"min_tag_overlap_pair"`
	MaxScanPatterns         int     `yaml:"max_scan_patterns"`
	LogElevation            bool    `yaml:"log_elevation"`
}

// NormalizePatternElevation fills zero values with conservative defaults.
func NormalizePatternElevation(c *PatternElevationConfig) *PatternElevationConfig {
	if c == nil {
		return nil
	}
	out := *c
	if out.MinAuthority <= 0 {
		out.MinAuthority = 4
	}
	if out.MinDistinctContexts <= 0 {
		out.MinDistinctContexts = 2
	}
	if out.MinDistinctAgents < 0 {
		out.MinDistinctAgents = 0
	}
	if out.MaxSupportingPatterns <= 0 {
		out.MaxSupportingPatterns = 8
	}
	if out.AuthorityElevationDelta <= 0 {
		out.AuthorityElevationDelta = 1
	}
	if out.AuthorityElevationDelta > 3 {
		out.AuthorityElevationDelta = 3
	}
	if out.MergeJaccardMin <= 0 || out.MergeJaccardMin > 1 {
		out.MergeJaccardMin = 0.82
	}
	if out.MinTagOverlapPair <= 0 || out.MinTagOverlapPair > 1 {
		out.MinTagOverlapPair = 0.4
	}
	if out.MaxScanPatterns <= 0 {
		out.MaxScanPatterns = 200
	}
	return &out
}

// ReuseScore combines authority and salience into one eligibility score.
func ReuseScore(authority, distinctContexts, distinctAgents int) float64 {
	return float64(authority) + 1.5*math.Log1p(float64(distinctContexts)) + 1.0*math.Log1p(float64(distinctAgents))
}

func (s *Service) eligibleForElevation(o *MemoryObject, cfg *PatternElevationConfig) bool {
	if o == nil || o.Kind != api.MemoryKindPattern || o.Status != api.StatusActive {
		return false
	}
	var p PatternPayload
	if len(o.Payload) == 0 || json.Unmarshal(o.Payload, &p) != nil {
		return false
	}
	if strings.TrimSpace(p.SupersededBy) != "" {
		return false
	}
	if p.Generalization != nil && p.Generalization.Reason == PatternElevationReason {
		return false
	}
	ctx, ag := SalienceDistinctCounts(o.Payload)
	if o.Authority < cfg.MinAuthority {
		return false
	}
	if ctx < cfg.MinDistinctContexts {
		return false
	}
	if ag < cfg.MinDistinctAgents {
		return false
	}
	if cfg.MinReuseScore > 0 && ReuseScore(o.Authority, ctx, ag) < cfg.MinReuseScore {
		return false
	}
	return true
}

func pairCompatible(a, b *MemoryObject, mergeJ, tagMin float64, negGuard bool) bool {
	ca := a.StatementCanonical
	if ca == "" {
		ca = memorynorm.StatementCanonical(a.Statement)
	}
	cb := b.StatementCanonical
	if cb == "" {
		cb = memorynorm.StatementCanonical(b.Statement)
	}
	j := similarity.CanonicalTokenJaccard(ca, cb)
	if j < mergeJ {
		return false
	}
	if negGuard && negationConflict(ca, cb) {
		return false
	}
	if tagOverlapFraction(a.Tags, b.Tags) < tagMin {
		return false
	}
	return true
}

type uf struct {
	p []int
}

func newUF(n int) *uf {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	return &uf{p: p}
}

func (u *uf) find(i int) int {
	if u.p[i] != i {
		u.p[i] = u.find(u.p[i])
	}
	return u.p[i]
}

func (u *uf) union(a, b int) {
	ra, rb := u.find(a), u.find(b)
	if ra != rb {
		u.p[rb] = ra
	}
}

// clusterElevationGroups returns disjoint index groups of size >= 2.
func clusterElevationGroups(patterns []MemoryObject, cfg *PatternElevationConfig) [][]int {
	n := len(patterns)
	if n < 2 {
		return nil
	}
	pg := NormalizePatternGeneralization(&PatternGeneralizationConfig{MergeJaccardMin: cfg.MergeJaccardMin, NegationGuard: true})
	mergeJ := cfg.MergeJaccardMin
	negGuard := true
	if pg != nil {
		mergeJ = pg.MergeJaccardMin
		negGuard = pg.NegationGuard
	}
	u := newUF(n)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if pairCompatible(&patterns[i], &patterns[j], mergeJ, cfg.MinTagOverlapPair, negGuard) {
				u.union(i, j)
			}
		}
	}
	roots := make(map[int][]int)
	for i := 0; i < n; i++ {
		r := u.find(i)
		roots[r] = append(roots[r], i)
	}
	var out [][]int
	for _, idxs := range roots {
		if len(idxs) >= 2 {
			sort.Ints(idxs)
			out = append(out, idxs)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i][0] < out[j][0]
	})
	return out
}

func sortedClusterIDs(patterns []MemoryObject, idxs []int) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(idxs))
	for _, i := range idxs {
		ids = append(ids, patterns[i].ID)
	}
	sort.Slice(ids, func(a, b int) bool { return ids[a].String() < ids[b].String() })
	return ids
}

func supportingKeysForCluster(patterns []MemoryObject, idxs []int) []string {
	var keys []string
	seen := make(map[string]struct{})
	for _, i := range idxs {
		sk := patterns[i].StatementKey
		if sk == "" {
			sk = memorynorm.StatementKey(patterns[i].Statement)
		}
		if sk == "" {
			continue
		}
		if _, ok := seen[sk]; ok {
			continue
		}
		seen[sk] = struct{}{}
		keys = append(keys, sk)
	}
	sort.Strings(keys)
	return keys
}

func avgJaccardInCluster(patterns []MemoryObject, idxs []int, mergeJ float64) float64 {
	var sum float64
	var n int
	for a := 0; a < len(idxs); a++ {
		for b := a + 1; b < len(idxs); b++ {
			i, j := idxs[a], idxs[b]
			ca := patterns[i].StatementCanonical
			if ca == "" {
				ca = memorynorm.StatementCanonical(patterns[i].Statement)
			}
			cb := patterns[j].StatementCanonical
			if cb == "" {
				cb = memorynorm.StatementCanonical(patterns[j].Statement)
			}
			sum += similarity.CanonicalTokenJaccard(ca, cb)
			n++
		}
	}
	if n == 0 {
		return mergeJ
	}
	return sum / float64(n)
}

func maxSalienceFromCluster(patterns []MemoryObject, idxs []int) (maxCtx, maxAg int) {
	for _, i := range idxs {
		c, a := SalienceDistinctCounts(patterns[i].Payload)
		if c > maxCtx {
			maxCtx = c
		}
		if a > maxAg {
			maxAg = a
		}
	}
	return maxCtx, maxAg
}

func intersectTags(patterns []MemoryObject, idxs []int) []string {
	if len(idxs) == 0 {
		return nil
	}
	set := make(map[string]int)
	for _, t := range patterns[idxs[0]].Tags {
		t = strings.TrimSpace(strings.ToLower(t))
		if t != "" {
			set[t] = 1
		}
	}
	for _, i := range idxs[1:] {
		next := make(map[string]int)
		for _, t := range patterns[i].Tags {
			t = strings.TrimSpace(strings.ToLower(t))
			if t == "" {
				continue
			}
			if set[t] > 0 {
				next[t] = set[t] + 1
			}
		}
		set = next
	}
	var out []string
	for t := range set {
		if set[t] == len(idxs) {
			out = append(out, t)
		}
	}
	sort.Strings(out)
	return out
}

func synthesizeDirective(directives []string) string {
	var dirs []string
	for _, d := range directives {
		d = strings.TrimSpace(d)
		if d != "" {
			dirs = append(dirs, d)
		}
	}
	if len(dirs) == 0 {
		return "Consolidated pattern strategy"
	}
	sort.Strings(dirs)
	if len(dirs) == 1 {
		return dirs[0]
	}
	common := longestCommonPrefixWords(dirs)
	if len(strings.TrimSpace(common)) >= 16 {
		return strings.TrimSpace(common)
	}
	return "Combine: " + strings.Join(dirs, "; ")
}

func longestCommonPrefixWords(sortedDirs []string) string {
	if len(sortedDirs) == 0 {
		return ""
	}
	first := strings.Fields(sortedDirs[0])
	if len(first) == 0 {
		return ""
	}
	var maxLen int
	for _, d := range sortedDirs[1:] {
		words := strings.Fields(d)
		n := 0
		for n < len(first) && n < len(words) && strings.EqualFold(first[n], words[n]) {
			n++
		}
		if n == 0 {
			return ""
		}
		if maxLen == 0 || n < maxLen {
			maxLen = n
		}
	}
	if maxLen == 0 {
		maxLen = len(first)
		for _, d := range sortedDirs[1:] {
			words := strings.Fields(d)
			n := 0
			for n < len(first) && n < len(words) && strings.EqualFold(first[n], words[n]) {
				n++
			}
			if n < maxLen {
				maxLen = n
			}
		}
	}
	return strings.Join(first[:maxLen], " ")
}

func bestPatternTemplate(patterns []MemoryObject, idxs []int) PatternPayload {
	best := -1
	bestAuth := -1
	for _, i := range idxs {
		if patterns[i].Authority > bestAuth {
			bestAuth = patterns[i].Authority
			best = i
		}
	}
	if best < 0 {
		return PatternPayload{}
	}
	var p PatternPayload
	_ = json.Unmarshal(patterns[best].Payload, &p)
	return p
}

func (s *Service) elevationAlreadyExists(ctx context.Context, sourceIDs []uuid.UUID) (bool, error) {
	list, err := s.Repo.Search(ctx, SearchRequest{
		Status: "active",
		Max:    500,
		Kinds:  []api.MemoryKind{api.MemoryKindPattern},
	})
	if err != nil {
		return false, err
	}
	sort.Slice(sourceIDs, func(i, j int) bool { return sourceIDs[i].String() < sourceIDs[j].String() })
	for i := range list {
		var p PatternPayload
		if json.Unmarshal(list[i].Payload, &p) != nil {
			continue
		}
		if p.Generalization == nil || p.Generalization.Reason != PatternElevationReason {
			continue
		}
		got := parseSupportingUUIDs(p.SupportingMemoryIDs)
		sort.Slice(got, func(a, b int) bool { return got[a].String() < got[b].String() })
		if uuidSlicesEqual(got, sourceIDs) {
			return true, nil
		}
	}
	return false, nil
}

func uuidSlicesEqual(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func parseSupportingUUIDs(ss []string) []uuid.UUID {
	var out []uuid.UUID
	for _, s := range ss {
		id, err := uuid.Parse(strings.TrimSpace(s))
		if err != nil {
			continue
		}
		out = append(out, id)
	}
	return out
}

func patchSupersededBy(existing []byte, elevatedID uuid.UUID) ([]byte, error) {
	raw := make(map[string]json.RawMessage)
	if len(existing) > 0 {
		_ = json.Unmarshal(existing, &raw)
	}
	if raw == nil {
		raw = make(map[string]json.RawMessage)
	}
	b, err := json.Marshal(elevatedID.String())
	if err != nil {
		return nil, err
	}
	raw["superseded_by"] = b
	return json.Marshal(raw)
}

// TryElevatePatterns scans eligible patterns, builds similarity clusters, and creates one elevated pattern per cluster.
func (s *Service) TryElevatePatterns(ctx context.Context, tags []string) ([]MemoryObject, error) {
	cfg := NormalizePatternElevation(s.PatternElevation)
	if s == nil || s.Repo == nil || cfg == nil || !cfg.Enabled {
		return nil, nil
	}
	list, err := s.Repo.Search(ctx, SearchRequest{
		Tags:   tags,
		Status: "active",
		Max:    cfg.MaxScanPatterns,
		Kinds:  []api.MemoryKind{api.MemoryKindPattern},
	})
	if err != nil {
		return nil, err
	}
	var eligible []MemoryObject
	for i := range list {
		if s.eligibleForElevation(&list[i], cfg) {
			eligible = append(eligible, list[i])
		}
	}
	groups := clusterElevationGroups(eligible, cfg)
	var created []MemoryObject
	for _, idxs := range groups {
		members := make([]MemoryObject, len(idxs))
		for j, ix := range idxs {
			members[j] = eligible[ix]
		}
		if len(members) > cfg.MaxSupportingPatterns {
			sort.SliceStable(members, func(i, j int) bool {
				if members[i].Authority != members[j].Authority {
					return members[i].Authority > members[j].Authority
				}
				return members[i].ID.String() < members[j].ID.String()
			})
			members = members[:cfg.MaxSupportingPatterns]
		}
		idxs := make([]int, len(members))
		for i := range idxs {
			idxs[i] = i
		}
		srcIDs := sortedClusterIDs(members, idxs)
		ok, err := s.elevationAlreadyExists(ctx, srcIDs)
		if err != nil {
			return created, err
		}
		if ok {
			continue
		}
		obj, err := s.elevateCluster(ctx, members, cfg)
		if err != nil {
			return created, err
		}
		if obj != nil {
			created = append(created, *obj)
		}
	}
	return created, nil
}

func (s *Service) elevateCluster(ctx context.Context, members []MemoryObject, cfg *PatternElevationConfig) (*MemoryObject, error) {
	if len(members) < 2 {
		return nil, nil
	}
	idxs := make([]int, len(members))
	for i := range idxs {
		idxs[i] = i
	}
	srcIDs := sortedClusterIDs(members, idxs)
	keys := supportingKeysForCluster(members, idxs)
	jAvg := avgJaccardInCluster(members, idxs, cfg.MergeJaccardMin)
	maxCtx, maxAg := maxSalienceFromCluster(members, idxs)
	tags := intersectTags(members, idxs)
	if len(tags) == 0 {
		tags = members[0].Tags
	}
	tpl := bestPatternTemplate(members, idxs)
	var directives []string
	for i := range members {
		var p PatternPayload
		if json.Unmarshal(members[i].Payload, &p) == nil && strings.TrimSpace(p.Directive) != "" {
			directives = append(directives, p.Directive)
		}
	}
	dir := synthesizeDirective(directives)
	if dir == "" {
		dir = synthesizeDirective([]string{members[0].Statement})
	}
	maxAuth := 0
	for i := range members {
		if members[i].Authority > maxAuth {
			maxAuth = members[i].Authority
		}
	}
	newAuth := maxAuth + cfg.AuthorityElevationDelta
	if newAuth > AuthorityScale {
		newAuth = AuthorityScale
	}
	supIDs := make([]string, len(srcIDs))
	for i := range srcIDs {
		supIDs[i] = srcIDs[i].String()
	}
	gen := PatternGeneralizationMeta{
		Reason:                  PatternElevationReason,
		Jaccard:                   jAvg,
		TagOverlapFraction:        tagOverlapFraction(members[0].Tags, members[1].Tags),
		SupportingStatementKeys: keys,
	}
	tpl.Generalization = &gen
	tpl.SupportingMemoryIDs = supIDs
	tpl.Directive = dir
	tpl.Experience = "Elevated from repeated successful patterns: " + strings.Join(supIDs, ", ")
	tpl.Decision = "Prefer this consolidated strategy over partial variants when both match."
	tpl.Outcome = "Higher-confidence recall via dominance, not deletion of evidence."
	pl, err := json.Marshal(&tpl)
	if err != nil {
		return nil, err
	}
	salRaw, err := json.Marshal(map[string]any{
		"distinct_contexts": maxCtx,
		"distinct_agents":   maxAg,
	})
	if err != nil {
		return nil, err
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(pl, &root); err != nil {
		return nil, err
	}
	root["salience"] = salRaw
	merged, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	raw := json.RawMessage(merged)
	cr := CreateRequest{
		Kind:                 api.MemoryKindPattern,
		Authority:            newAuth,
		Applicability:        tpl.PolarToApplicability(),
		Statement:            dir,
		Tags:                 tags,
		Payload:              &raw,
		SkipPatternNearMerge: true,
	}
	if cr.Applicability == "" {
		cr.Applicability = api.ApplicabilityGoverning
	}
	obj, err := s.Create(ctx, cr)
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		patched, perr := patchSupersededBy(m.Payload, obj.ID)
		if perr != nil {
			return obj, perr
		}
		if err := s.Repo.UpdatePayload(ctx, m.ID, patched); err != nil {
			return obj, err
		}
	}
	if s.Cache != nil {
		_ = s.Cache.DeleteByPrefix(ctx, "memory:tags:")
		s.invalidateRecallBundleCache(ctx)
	}
	if cfg.LogElevation {
		slog.Info("[PATTERN ELEVATION]", "elevated_id", obj.ID.String(), "source_patterns", supIDs, "statement", dir)
	}
	return obj, nil
}

// PolarToApplicability maps pattern polarity to applicability for create (internal).
func (p PatternPayload) PolarToApplicability() api.Applicability {
	if strings.EqualFold(p.Polarity, string(PatternPolarityNegative)) {
		return api.ApplicabilityGoverning
	}
	return api.ApplicabilityAdvisory
}
