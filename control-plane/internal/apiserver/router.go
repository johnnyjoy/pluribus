// Package apiserver wires the control-plane HTTP API (same routes as cmd/controlplane).
package apiserver

import (
	"net/http"
	"strings"
	"time"

	"control-plane/internal/app"
	"control-plane/internal/contradiction"
	"control-plane/internal/curation"
	"control-plane/internal/distillation"
	"control-plane/internal/drift"
	"control-plane/internal/enforcement"
	"control-plane/internal/evidence"
	"control-plane/internal/httpx"
	"control-plane/internal/ingest"
	"control-plane/internal/mcp"
	"control-plane/internal/memory"
	"control-plane/internal/recall"
	"control-plane/internal/runmulti"
	"control-plane/internal/similarity"
	"control-plane/internal/synthesis"
	"control-plane/internal/tooling"

	"github.com/go-chi/chi/v5"
)

// NewRouter returns the control-plane HTTP router (readiness, /v1/*, optional MCP) wrapped with Pluribus auth when PLURIBUS_API_KEY is set.
func NewRouter(cfg *app.Config, container *app.Container) (http.Handler, error) {
	router := httpx.NewRouter()
	httpx.RegisterReadiness(router, container.DB)

	memoryRepo := &memory.Repo{DB: container.DB}
	cacheTTL := cfg.Recall.CacheTTLSeconds
	if cacheTTL <= 0 {
		cacheTTL = 300
	}
	memorySvc := &memory.Service{
		Repo:     memoryRepo,
		Cache:    container.Cache,
		CacheTTL: time.Duration(cacheTTL) * time.Second,
	}
	if cfg.Memory.Lifecycle != nil {
		memorySvc.Lifecycle = &memory.LifecycleConfig{
			AuthorityPositiveDelta:       cfg.Memory.Lifecycle.AuthorityPositiveDelta,
			AuthorityNegativeDelta:       cfg.Memory.Lifecycle.AuthorityNegativeDelta,
			ExpirationAuthorityThreshold: cfg.Memory.Lifecycle.ExpirationAuthorityThreshold,
		}
		if memorySvc.Lifecycle.AuthorityPositiveDelta <= 0 {
			memorySvc.Lifecycle.AuthorityPositiveDelta = 0.1
		}
		if memorySvc.Lifecycle.AuthorityNegativeDelta <= 0 {
			memorySvc.Lifecycle.AuthorityNegativeDelta = 0.2
		}
		if memorySvc.Lifecycle.ExpirationAuthorityThreshold <= 0 {
			memorySvc.Lifecycle.ExpirationAuthorityThreshold = 2
		}
	}
	if cfg.Memory.Dedup != nil {
		memorySvc.Dedup = &memory.DedupConfig{Enabled: cfg.Memory.Dedup.Enabled}
	}
	if cfg.Memory.PatternGeneralization != nil {
		pg := cfg.Memory.PatternGeneralization
		memorySvc.PatternGeneralization = &memory.PatternGeneralizationConfig{
			Enabled:                    pg.Enabled,
			MergeJaccardMin:            pg.MergeJaccardMin,
			MinTagOverlapFraction:      pg.MinTagOverlapFraction,
			MaxCandidatesScan:          pg.MaxCandidatesScan,
			MaxSupportingStatementKeys: pg.MaxSupportingStatementKeys,
			NegationGuard:              pg.NegationGuard,
		}
	}
	if cfg.Memory.PatternElevation != nil {
		pe := cfg.Memory.PatternElevation
		memorySvc.PatternElevation = &memory.PatternElevationConfig{
			Enabled:                 pe.Enabled,
			MinReuseScore:           pe.MinReuseScore,
			MinDistinctContexts:     pe.MinDistinctContexts,
			MinDistinctAgents:       pe.MinDistinctAgents,
			MinAuthority:            pe.MinAuthority,
			MaxSupportingPatterns:   pe.MaxSupportingPatterns,
			AuthorityElevationDelta: pe.AuthorityElevationDelta,
			MergeJaccardMin:         pe.MergeJaccardMin,
			MinTagOverlapPair:       pe.MinTagOverlapPair,
			MaxScanPatterns:         pe.MaxScanPatterns,
			LogElevation:            pe.LogElevation,
		}
	}
	if cfg.Memory.RecallReinforcement != nil {
		memorySvc.Reinforcement = cfg.Memory.RecallReinforcement
	}
	if cfg.Recall.SemanticRetrieval != nil {
		memorySvc.Semantic = cfg.Recall.SemanticRetrieval
		memorySvc.Embedder = memory.NewEmbedderFromConfig(cfg.Recall.SemanticRetrieval)
	}
	memoryRelRepo := &memory.RelationshipRepo{DB: container.DB}
	memorySvc.Relationships = memoryRelRepo
	memoryHandlers := &memory.Handlers{Service: memorySvc, Relationships: memoryRelRepo}

	contradictionRepo := &contradiction.Repo{DB: container.DB}
	contradictionSvc := &contradiction.Service{Repo: contradictionRepo, MemoryRepo: memoryRepo}
	contradictionHandlers := &contradiction.Handlers{Service: contradictionSvc}

	curationRepo := &curation.Repo{DB: container.DB}
	curationConfig := &curation.SalienceConfig{
		CandidateThreshold: cfg.Curation.CandidateThreshold,
		ReviewThreshold:    cfg.Curation.ReviewThreshold,
		PromoteThreshold:   cfg.Curation.PromoteThreshold,
	}

	recallRepo := &recall.Repo{DB: container.DB}
	recallCompiler := &recall.Compiler{
		Memory:          memorySvc,
		Contradiction:   contradictionSvc,
		Relationships:   memoryRelRepo,
	}
	if cfg.Memory.Dedup != nil {
		recallCompiler.NearDupJaccardThreshold = cfg.Memory.Dedup.NearDupJaccardThreshold
	}
	rk := cfg.Recall.Ranking
	if rk == nil {
		rk = &app.RecallRankingConfig{}
	}
	symbolOverlap := rk.WeightSymbolOverlap
	if cfg.LSP != nil {
		if !cfg.LSP.Enabled {
			symbolOverlap = 0
		} else if cfg.LSP.RecallSymbolBoost > 0 {
			symbolOverlap = cfg.LSP.RecallSymbolBoost
		}
	}
	w := recall.RankingWeightsFromConfig(
		rk.WeightAuthority,
		rk.WeightRecency,
		rk.WeightTagMatch,
		rk.WeightFailureOverlap,
		symbolOverlap,
		rk.WeightPatternPriority,
		rk.WeightLexicalSimilarity,
		rk.WeightPatternGeneralization,
		rk.WeightFailureSeverity,
		rk.WeightCrossContextSalience,
		rk.WeightCrossContextSalienceK,
		rk.WeightCrossAgentSalience,
		rk.WeightCrossAgentSalienceK,
		0, // semantic resolved below (explicit YAML 0 is handled by ResolveSemanticSimilarityWeight)
		rk.WeightElevationSuppression,
	)
	w.SemanticSimilarity = recall.ResolveSemanticSimilarityWeight(rk.WeightSemanticSimilarity)
	recallCompiler.Ranking = &w
	if cfg.Recall.SemanticRetrieval != nil && cfg.Recall.SemanticRetrieval.RetrievalEnabled() {
		recallCompiler.Semantic = &recall.SemanticRecallConfig{
			Enabled:             true,
			MaxCandidates:       cfg.Recall.SemanticRetrieval.MaxSemanticCandidates,
			MinCosineSimilarity: cfg.Recall.SemanticRetrieval.MinCosineSimilarity,
			LogSemanticMatches:  cfg.Recall.SemanticRetrieval.LogSemanticMatches,
		}
	}
	if cfg.Recall.ExperiencesEnabled {
		recallCompiler.Experiences = recall.NewDBExperienceLister(memorySvc)
		recallCompiler.ExperiencesLimit = cfg.Recall.ExperiencesLimit
	}
	if cfg.Recall.RIU != nil && cfg.Recall.RIU.Enabled {
		if recallCompiler.Ranking == nil {
			w := recall.DefaultRankingWeights()
			recallCompiler.Ranking = &w
		}
		recallCompiler.RIU = &recall.RIUConfig{
			Enabled: true,
			Policy:  recall.ParseContradictionPolicy(cfg.Recall.RIU.ContradictionPolicy),
			Weights: recall.RIUWeightsFromConfig(
				cfg.Recall.RIU.WeightApplicability,
				cfg.Recall.RIU.WeightTransferable,
				cfg.Recall.RIU.WeightLineageProxy,
				cfg.Recall.RIU.WeightContradictionPenalty,
			),
			BoundedPairMax: cfg.Recall.RIU.BoundedPairMax,
		}
	}
	recallCompiler.LogRankTopN = cfg.Recall.LogRankTopN
	recallSvc := &recall.Service{
		Compiler:           recallCompiler,
		Repo:               recallRepo,
		Cache:              container.Cache,
		CacheTTL:           time.Duration(cacheTTL) * time.Second,
		DefaultMaxTotal:    cfg.Recall.DefaultMaxTotal,
		DefaultMaxTokens:   cfg.Recall.DefaultMaxTokens,
		BehaviorValidation: cfg.Recall.BehaviorValidation,
		Promotion: &recall.PromotionPolicy{
			RequireEvidence:      cfg.Promotion.RequireEvidence,
			MinEvidenceLinks:     cfg.Promotion.MinEvidenceLinks,
			MinEvidenceScore:     cfg.Promotion.MinEvidenceScore,
			RequireReview:        cfg.Promotion.RequireReview,
			MinPromoteConfidence: cfg.Promotion.MinPromoteConfidence,
			MinPolicyComposite:   cfg.Promotion.MinPolicyComposite,
			WeightConfidence:     cfg.Promotion.WeightConfidence,
			WeightSignal:         cfg.Promotion.WeightSignal,
			WeightEvidence:       cfg.Promotion.WeightEvidence,
			SignalNormDivisor:    cfg.Promotion.SignalNormDivisor,
		},
	}
	if cfg.LSP != nil && cfg.LSP.Enabled {
		gopls := &tooling.GoplsClient{}
		recallCompiler.LSPRecallEnabled = true
		recallCompiler.LSP = gopls
		recallCompiler.ReferenceExpansionLimit = cfg.LSP.ReferenceExpansionLimit
		recallSvc.LSPEnabled = true
		recallSvc.LSP = gopls
		recallSvc.LSPAutoSymbolMax = cfg.LSP.AutoSymbolMax
		if recallSvc.LSPAutoSymbolMax <= 0 {
			recallSvc.LSPAutoSymbolMax = 64
		}
	}
	baseURL := cfg.Server.Bind
	if strings.HasPrefix(baseURL, ":") {
		baseURL = "http://127.0.0.1" + baseURL
	} else if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	recallSvc.RunMultiBaseURL = baseURL
	if cfg.Synthesis.Enabled {
		backend, err := synthesis.NewBackend(&cfg.Synthesis)
		if err != nil {
			return nil, err
		}
		llm := runmulti.NewSynthesisLLM(backend)
		recallSvc.RunMultiRunner = runmulti.NewRunner(baseURL, llm)
	} else {
		recallSvc.RunMultiRunner = nil
	}
	recallSvc.MemoryPromoter = memorySvc
	recallSvc.UsageReinforcer = memorySvc
	if cfg.SlowPathEnabled() {
		recallSvc.SlowPath = &recall.SlowPathPreflightConfig{
			Enabled:               true,
			HighRiskThreshold:     cfg.SlowPath.HighRiskThreshold,
			ExpandConstraintsBy:   cfg.SlowPath.ExpandConstraintsBy,
			ExpandFailuresBy:      cfg.SlowPath.ExpandFailuresBy,
			ExpandPatternsBy:      cfg.SlowPath.ExpandPatternsBy,
			ExtraVariantsWhenSlow: cfg.SlowPath.ExtraVariantsWhenSlow,
		}
	}
	if cfg.Recall.TriggeredRecall != nil {
		recallSvc.TriggerRecall = recall.NormalizeTriggerRecall(cfg.Recall.TriggeredRecall)
	}
	recallHandlers := &recall.Handlers{Service: recallSvc}

	driftRepo := &drift.Repo{DB: container.DB}
	driftSvc := &drift.Service{Repo: driftRepo, Memory: memorySvc}
	driftSvc.PatternHighBlocks = cfg.Drift.PatternHighBlocks
	if !driftSvc.PatternHighBlocks {
		driftSvc.PatternHighBlocks = true
	}
	if cfg.Drift.FailureFuzzyThreshold > 0 {
		driftSvc.FailureFuzzyThreshold = cfg.Drift.FailureFuzzyThreshold
	}
	if cfg.SlowPathEnabled() {
		driftSvc.RequireSecondDriftCheck = cfg.SlowPath.RequireSecondDriftCheck
	}
	threshold := cfg.Drift.LSPHighRiskReferenceThreshold
	if cfg.LSP != nil {
		if cfg.LSP.HighRiskReferenceThreshold > 0 {
			threshold = cfg.LSP.HighRiskReferenceThreshold
		}
		if !cfg.LSP.Enabled {
			threshold = 0
		}
	}
	if threshold > 0 {
		driftSvc.LSPHighRiskReferenceThreshold = threshold
		driftSvc.LSP = &tooling.GoplsClient{}
	}
	driftHandlers := &drift.Handlers{Service: driftSvc}

	evidenceStorage := &evidence.Storage{RootPath: container.EvidenceRoot}
	evidenceRepo := &evidence.Repo{DB: container.DB}
	evidenceSvc := &evidence.Service{
		Repo:            evidenceRepo,
		Storage:         evidenceStorage,
		Memory:          memoryRepo,
		AuthorityFactor: cfg.Evidence.AuthorityFactor,
	}
	if evidenceSvc.AuthorityFactor <= 0 {
		evidenceSvc.AuthorityFactor = 0.1
	}
	memorySvc.Evidence = evidenceSvc
	recallSvc.Evidence = evidenceSvc
	recallSvc.EvidenceInBundle = cfg.Recall.EvidenceInBundle
	recallSvc.EvidenceLister = evidenceSvc
	evidenceHandlers := &evidence.Handlers{Service: evidenceSvc}

	enforcementSvc := &enforcement.Service{
		Repo:              memoryRepo,
		Evidence:          evidenceSvc,
		Config:            &cfg.Enforcement,
		SuccessReinforcer: memorySvc,
	}
	enforcementHandlers := &enforcement.Handlers{Service: enforcementSvc}

	simRepo := &similarity.Repo{DB: container.DB}

	curationSvc := &curation.Service{
		Repo:           curationRepo,
		Config:         curationConfig,
		Memory:         memorySvc,
		MemoryLookup:   memorySvc,
		Relationships:  memoryRelRepo,
		MemoryDup:      memoryRepo,
		FailureCounter: memoryRepo,
		Episodes:       simRepo,
		DigestLimits: &curation.DigestLimits{
			MaxProposals:        cfg.Curation.DigestMaxProposals,
			WorkSummaryMaxBytes: cfg.Curation.DigestWorkSummaryMaxBytes,
			StatementMaxBytes:   cfg.Curation.DigestStatementMaxBytes,
			ReasonMaxBytes:      cfg.Curation.DigestReasonMaxBytes,
		},
		Evidence: evidenceSvc,
		Promotion: &curation.PromotionDigestConfig{
			RequireEvidence:        cfg.Promotion.RequireEvidence,
			MinEvidenceLinks:       cfg.Promotion.MinEvidenceLinks,
			RequireReview:          cfg.Promotion.RequireReview,
			AutoPromote:            cfg.Promotion.AutoPromote,
			AutoMinSupportCount:    cfg.Promotion.AutoMinSupportCount,
			AutoMinSalience:        cfg.Promotion.AutoMinSalience,
			AutoAllowedKinds:       cfg.Promotion.AutoAllowedKinds,
			CanonicalConsolidation: memory.NormalizeCanonicalConsolidation(cfg.Promotion.CanonicalConsolidation),
		},
	}
	curationHandlers := &curation.Handlers{Service: curationSvc}

	ingestRepo := &ingest.Repo{DB: container.DB}
	ingestSvc := ingest.NewService(ingestRepo)
	ingestSvc.AutoPromote = cfg.Ingest.AutoPromote
	ingestSvc.Promoter = memorySvc
	ingestHandlers := &ingest.Handlers{Service: ingestSvc}
	dedupEnabled := true
	dedupWin := 120 * time.Second
	if cfg.MCP != nil && cfg.MCP.MemoryFormation != nil {
		mf := cfg.MCP.MemoryFormation
		if mf.DedupEnabled != nil {
			dedupEnabled = *mf.DedupEnabled
		}
		if mf.DedupWindowSeconds > 0 {
			dedupWin = time.Duration(mf.DedupWindowSeconds) * time.Second
		}
	}
	simCfg := &similarity.Config{
		Enabled:         cfg.Similarity.Enabled,
		MaxSummaryBytes: cfg.Similarity.MaxSummaryBytes,
		MaxEpisodesScan: cfg.Similarity.MaxEpisodesScan,
		MaxResults:      cfg.Similarity.MaxResults,
		MinResemblance:  cfg.Similarity.MinResemblance,
		McpDedupEnabled: dedupEnabled,
		McpDedupWindow:  dedupWin,
	}
	simSvc := &similarity.Service{Repo: simRepo, Config: simCfg}
	simHandlers := &similarity.Handlers{Service: simSvc}

	distSvc := &distillation.Service{
		Curation: curationRepo,
		Episodes: simRepo,
		Config: &distillation.Config{
			Enabled:                  cfg.Distillation.Enabled,
			AutoFromAdvisoryEpisodes: cfg.Distillation.AutoFromAdvisoryEpisodes,
			MinStatementChars:        cfg.Distillation.MinStatementChars,
		},
	}
	distHandlers := &distillation.Handlers{Service: distSvc}
	if cfg.Distillation.Enabled && cfg.Distillation.AutoFromAdvisoryEpisodes {
		simHandlers.AutoDistill = distSvc
	}

	router.Route("/v1", func(r chi.Router) {
		r.Route("/memories", func(r chi.Router) {
			r.Post("/", memoryHandlers.CreateMemories)
			r.Post("/search", memoryHandlers.SearchMemories)
		})
		r.Route("/memory", func(r chi.Router) {
			r.Post("/relationships", memoryHandlers.CreateRelationship)
			r.Get("/{id}/relationships", memoryHandlers.ListRelationships)
			r.Post("/", memoryHandlers.Create)
			r.Post("/pattern-elevation/run", memoryHandlers.RunPatternElevation)
			r.Post("/promote", memoryHandlers.Promote)
			r.Post("/search", memoryHandlers.Search)
			r.Put("/{id}/attributes", memoryHandlers.SetAttributes)
			r.Post("/{id}/authority/event", memoryHandlers.ApplyAuthorityEvent)
			r.Post("/expire", memoryHandlers.ExpireMemories)
		})
		r.Route("/contradictions", func(r chi.Router) {
			r.Post("/", contradictionHandlers.Create)
			r.Post("/detect", contradictionHandlers.DetectAndRecord)
			r.Get("/", contradictionHandlers.List)
			r.Get("/{id}", contradictionHandlers.GetByID)
			r.Patch("/{id}/resolution", contradictionHandlers.UpdateResolution)
		})
		r.Route("/curation", func(r chi.Router) {
			r.Post("/digest", curationHandlers.Digest)
			r.Post("/evaluate", curationHandlers.Evaluate)
			r.Post("/auto-promote", curationHandlers.AutoPromote)
			r.Get("/pending", curationHandlers.ListPending)
			r.Get("/promotion-suggestions", curationHandlers.PromotionSuggestions)
			r.Get("/strengthened", curationHandlers.Strengthened)
			r.Get("/candidates/{id}/review", curationHandlers.Review)
			r.Post("/candidates/{id}/materialize", curationHandlers.Materialize)
			r.Post("/candidates/{id}/promote", curationHandlers.MarkPromoted)
			r.Post("/candidates/{id}/reject", curationHandlers.MarkRejected)
		})
		r.Route("/recall", func(r chi.Router) {
			r.Get("/", recallHandlers.GetBundle)
			r.Post("/preflight", recallHandlers.Preflight)
			r.Post("/compile", recallHandlers.Compile)
			r.Post("/compile-multi", recallHandlers.CompileMulti)
			r.Post("/run-multi", recallHandlers.RunMulti)
		})
		r.Route("/drift", func(r chi.Router) {
			r.Post("/check", driftHandlers.Check)
		})
		r.Route("/enforcement", func(r chi.Router) {
			r.Post("/evaluate", enforcementHandlers.Evaluate)
		})
		r.Route("/evidence", func(r chi.Router) {
			r.Get("/", evidenceHandlers.List)
			r.Post("/", evidenceHandlers.Create)
			r.Get("/{id}", evidenceHandlers.GetByID)
			r.Post("/{id}/link", evidenceHandlers.Link)
		})
		r.Route("/ingest", func(r chi.Router) {
			r.Post("/cognition", ingestHandlers.Cognition)
			r.Post("/{id}/commit", ingestHandlers.Commit)
		})
		r.Route("/advisory-episodes", func(r chi.Router) {
			r.Post("/", simHandlers.Create)
			r.Post("/similar", simHandlers.Similar)
		})
		r.Post("/episodes/distill", distHandlers.Distill)
	})

	return httpx.WrapWithPluribusAuth(mcp.WrapHandler(router, cfg), container.APIKey), nil
}
