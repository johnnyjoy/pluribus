package promotion

import (
	"time"

	"control-plane/internal/signal"
)

// ExperienceRecord is one JSONL line (Phase 4 promoted experience).
type ExperienceRecord struct {
	Type           string    `json:"type"`
	Timestamp      time.Time `json:"timestamp"`
	Tags           []string  `json:"tags"`
	Content        string    `json:"content"`
	SourceVariants []string  `json:"source_variants"`
	Score          float64   `json:"score"`
}

// PromotionInput controls where and how to append a promotion.
type PromotionInput struct {
	Intent    signal.IntentText
	StorePath string
}

const experienceType = "experience"

// DefaultExperiencesPath is the default JSONL path (override with flag or EXPERIENCES_PATH).
const DefaultExperiencesPath = "data/memory/experiences.jsonl"
