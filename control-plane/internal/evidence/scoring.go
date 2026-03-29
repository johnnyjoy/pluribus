package evidence

// Evidence kind base scores (Task 79: 1.0, 0.8, 0.5, 0.2).
const (
	KindTest        = "test"
	KindBenchmark   = "benchmark"
	KindLog         = "log"
	KindObservation = "observation"
)

var baseScores = map[string]float64{
	KindTest:        1.0,
	KindBenchmark:   0.8,
	KindLog:         0.5,
	KindObservation: 0.2,
}

// BaseScore returns the base score for an evidence kind (test=1.0, benchmark=0.8, log=0.5, observation=0.2).
// Unknown kinds return 0.2 (observation).
func BaseScore(kind string) float64 {
	if s, ok := baseScores[kind]; ok {
		return s
	}
	return 0.2
}
