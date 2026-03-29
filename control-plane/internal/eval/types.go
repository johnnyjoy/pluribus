package eval

type Scenario struct {
	ID                string            `json:"id"`
	Goal              string            `json:"goal"`
	Context           string            `json:"context"`
	Trap              string            `json:"trap"`
	SuccessCondition  string            `json:"success_condition"`
	ScenarioType      string            `json:"scenario_type,omitempty"`
	Steps             []WorkflowStep    `json:"steps,omitempty"`
	ResumePoints      []ResumePoint     `json:"resume_points,omitempty"`
	DriftExpectations DriftExpectations `json:"drift_expectations,omitempty"`

	ExpectedExtraction struct {
		State      []string `json:"state"`
		Decision   []string `json:"decision"`
		Failure    []string `json:"failure"`
		Pattern    []string `json:"pattern"`
		Constraint []string `json:"constraint"`
	} `json:"expected_extraction"`

	RecallExpectations struct {
		Query       string   `json:"query"`
		MustInclude []string `json:"must_include"`
		// MustBeFirst: entries "bucket::substring" — first row in that recall bucket must contain substring (case-insensitive). Buckets: continuity, constraints, experience.
		MustBeFirst []string `json:"must_be_first,omitempty"`
	} `json:"recall_expectations"`

	BehaviorExpectations struct {
		NewTask   string   `json:"new_task"`
		MustAvoid []string `json:"must_avoid"`
		MustApply []string `json:"must_apply"`
	} `json:"behavior_expectations"`
}

type WorkflowStep struct {
	Task        string   `json:"task"`
	Agent       string   `json:"agent,omitempty"`
	Action      string   `json:"action"`
	Query       string   `json:"query,omitempty"`
	Trap        string   `json:"trap,omitempty"`
	NewTask     string   `json:"new_task,omitempty"`
	MustAvoid   []string `json:"must_avoid,omitempty"`
	MustApply   []string `json:"must_apply,omitempty"`
	MustInclude []string `json:"must_include,omitempty"`
}

type ResumePoint struct {
	Step int    `json:"step"`
	Gap  string `json:"gap,omitempty"`
}

type DriftExpectations struct {
	MustDetect    []string `json:"must_detect,omitempty"`
	MustNotDetect []string `json:"must_not_detect,omitempty"`
}

type CheckResult struct {
	Pass    bool
	Details []string
}

// ScenarioReport is the legacy single-path report shape (explicit recall only).
type ScenarioReport struct {
	ScenarioID string
	Extraction CheckResult
	Recall     CheckResult
	Behavior   BehaviorResult
}

type BehaviorResult struct {
	Pass                bool
	Output              string
	AvoidedFailure      bool
	AppliedPattern      bool
	AlignedWithDecision bool
	Details             []string
}

// ArmReport is recall + behavior validation for one recall mode (explicit or triggered).
type ArmReport struct {
	Recall   CheckResult
	Behavior BehaviorResult
}

// TriggerObserved captures triggered-recall metadata for measurement (timing proxy + over-trigger analysis).
type TriggerObserved struct {
	TriggersFired    int
	Kinds            []string
	SkippedReason    string
	ExplicitQuery    string
	EffectiveQuery   string
	Capped           bool
	RedundantTrigger bool
}

// DeltaResult compares explicit vs triggered outcomes (behavior-first per sprint charter).
type DeltaResult struct {
	Improvement string // "yes" | "no" | "same"
	Notes       string
}

// DualScenarioReport holds extraction once plus both arms and comparative delta.
type DualScenarioReport struct {
	ScenarioID string
	Extraction CheckResult
	Explicit   ArmReport
	Triggered  ArmReport
	Trigger    TriggerObserved
	Delta      DeltaResult
	Stress     *StressReport
}

type StepTrace struct {
	Index               int
	Task                string
	Action              string
	RecallUsed          bool
	RestoredState       int
	RestoredConstraints int
	DriftEvents         []DriftEvent
}

type DriftEvent struct {
	Type  string
	Cause string
}

type StressReport struct {
	ScenarioType         string
	ContinuityMaintained bool
	FailureAvoided       bool
	PatternReused        bool
	DriftDetected        bool
	Issues               []string
	StepTraces           []StepTrace
}
