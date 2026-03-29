package promotion

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// Namespace for deterministic synthetic memory IDs from promoted experiences (Phase 4).
var experienceIDNamespace = uuid.MustParse("018e23d5-7c4e-7a3f-9b2c-000000000004")

// FileExperienceLister loads recent experience JSONL lines for recall (implements recall.ExperienceLister).
type FileExperienceLister struct {
	Path           string
	BaseAuthority  int
	AuthorityBoost int
}

// NewFileExperienceLister returns a lister reading path with boosted authority for synthetic decisions.
func NewFileExperienceLister(path string, baseAuthority, authorityBoost int) *FileExperienceLister {
	if baseAuthority <= 0 {
		baseAuthority = 7
	}
	if authorityBoost < 0 {
		authorityBoost = 0
	}
	return &FileExperienceLister{Path: path, BaseAuthority: baseAuthority, AuthorityBoost: authorityBoost}
}

// ListForCompile returns up to limit experience records as synthetic MemoryObject (kind=decision).
func (f *FileExperienceLister) ListForCompile(ctx context.Context, limit int) ([]memory.MemoryObject, error) {
	_ = ctx
	if limit <= 0 {
		limit = 50
	}
	if f.Path == "" {
		return nil, nil
	}
	recs, err := loadExperienceRecords(f.Path, limit)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]memory.MemoryObject, 0, len(recs))
	for _, rec := range recs {
		out = append(out, experienceToMemoryObject(rec, f.BaseAuthority, f.AuthorityBoost, now))
	}
	return out, nil
}

func loadExperienceRecords(path string, limit int) ([]ExperienceRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var matched []ExperienceRecord
	sc := bufio.NewScanner(f)
	// Large lines
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var rec ExperienceRecord
		if json.Unmarshal([]byte(line), &rec) != nil {
			continue
		}
		if rec.Type != "" && rec.Type != experienceType {
			continue
		}
		matched = append(matched, rec)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(matched) > limit {
		matched = matched[len(matched)-limit:]
	}
	return matched, nil
}

const maxStatementRunes = 4000

func experienceToMemoryObject(rec ExperienceRecord, base, boost int, now time.Time) memory.MemoryObject {
	auth := base + boost
	if auth > 10 {
		auth = 10
	}
	stmt := rec.Content
	if utf8.RuneCountInString(stmt) > maxStatementRunes {
		r := []rune(stmt)
		stmt = string(r[:maxStatementRunes]) + "…"
	}
	ts := rec.Timestamp
	if ts.IsZero() {
		ts = now
	}
	id := uuid.NewSHA1(experienceIDNamespace, []byte(rec.Content+"|"+ts.UTC().Format(time.RFC3339Nano)))

	tags := []string{"experience", "merge"}
	for _, t := range rec.Tags {
		t = strings.TrimSpace(strings.ToLower(t))
		if t != "" {
			tags = append(tags, t)
		}
	}
	tags = dedupeTags(tags)

	return memory.MemoryObject{
		ID:            id,
		Kind:          api.MemoryKindDecision,
		Authority:     auth,
		Applicability: api.ApplicabilityAdvisory,
		Statement:     stmt,
		Status:        api.StatusActive,
		Tags:          tags,
		CreatedAt:     ts,
		UpdatedAt:     ts,
	}
}

func dedupeTags(in []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, t := range in {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}
