package agentusefulness

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"control-plane/internal/utility"
)

// RunSuite executes Phase 11B task fixtures only (legacy gate).
func RunSuite(lc *LoadedCorpus) (*BenchmarkReport, error) {
	return RunSuiteForTasks(lc, lc.Phase11BTasks())
}

// RunSuiteForTasks executes the given task fixtures.
func RunSuiteForTasks(lc *LoadedCorpus, tasks []TaskFixture) (*BenchmarkReport, error) {
	if lc == nil {
		return nil, fmt.Errorf("nil corpus")
	}
	th := DefaultGateThresholds()
	var taskResults []TaskResult

	for _, task := range tasks {
		tr, err := RunTask(context.Background(), lc, task)
		if err != nil {
			return nil, fmt.Errorf("task %s: %w", task.ID, err)
		}
		taskResults = append(taskResults, tr)
	}

	summary := ComputeSuiteMetrics(taskResults)
	passed, fails := EvaluateGate(summary, th)
	return &BenchmarkReport{
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Summary:      summary,
		Tasks:        taskResults,
		Thresholds:   th,
		GatePassed:   passed,
		GateFailures: fails,
	}, nil
}

// RunCognitiveSuite executes Phase 11C research-backed cognitive tasks.
func RunCognitiveSuite(lc *LoadedCorpus) (*CognitiveBenchmarkReport, error) {
	if lc == nil {
		return nil, fmt.Errorf("nil corpus")
	}
	tasks := lc.CognitiveTasks()
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no cognitive tasks loaded")
	}
	th := DefaultCognitiveGateThresholds()
	var taskResults []TaskResult
	for _, task := range tasks {
		tr, err := RunTask(context.Background(), lc, task)
		if err != nil {
			return nil, fmt.Errorf("task %s: %w", task.ID, err)
		}
		taskResults = append(taskResults, tr)
	}
	summary := ComputeCognitiveMetrics(tasks, taskResults, lc)
	passed, fails := EvaluateCognitiveGate(summary, th)
	return &CognitiveBenchmarkReport{
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		ResearchSources: DocumentedResearchSources(),
		Summary:         summary,
		Tasks:           taskResults,
		Thresholds:      th,
		GatePassed:      passed,
		GateFailures:    fails,
	}, nil
}

// WriteCognitiveReport writes Phase 11C benchmark JSON artifact.
func WriteCognitiveReport(report *CognitiveBenchmarkReport, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeJSON(path, report)
}

// RunTask executes no-memory, REST memory, and MCP memory variants.
func RunTask(ctx context.Context, lc *LoadedCorpus, task TaskFixture) (TaskResult, error) {
	tr := TaskResult{
		TaskID:             task.ID,
		RequiresMemoryHelp: task.RequiresMemoryHelp,
		ParityRequired:     task.CheckMCPRESTParity,
	}

	// No-memory baseline
	noFacts, noTrace := SimulateAgent(task, lc, RunModeNoMemory, nil)
	noScore, noFailures := ScoreRun(task, RunModeNoMemory, noFacts, nil, noTrace)
	tr.NoMemory = RunResult{
		TaskID:    task.ID,
		RunID:     task.ID + ":no_memory",
		Interface: InterfaceREST,
		Mode:      RunModeNoMemory,
		AnswerFacts: noFacts,
		UseTrace:  noTrace,
		Score:     noScore,
		Failures:  noFailures,
	}
	tr.NoMemory.OutcomeFeedback = BuildOutcomeFeedback(task, tr.NoMemory, lc)

	objects := lc.ObjectsForTask(task)
	compiler := NewCompiler(objects, filterUtility(lc, task), lc.LabelToID)
	req := BuildCompileRequest(task)

	// REST memory run
	bundleREST, err := RecallREST(ctx, compiler, req)
	if err != nil {
		return tr, fmt.Errorf("rest recall: %w", err)
	}
	itemsREST := FlattenBundle(bundleREST)
	restFacts, restTrace := SimulateAgent(task, lc, RunModeMemory, itemsREST)
	manifestREST := BuildManifest(task.ID, task.ID+":memory:rest", InterfaceREST, RunModeMemory, req, bundleREST, lc)
	restScore, restFailures := ScoreRun(task, RunModeMemory, restFacts, manifestREST, restTrace)
	tr.MemoryREST = RunResult{
		TaskID:    task.ID,
		RunID:     task.ID + ":memory:rest",
		Interface: InterfaceREST,
		Mode:      RunModeMemory,
		AnswerFacts: restFacts,
		Manifest:  manifestREST,
		UseTrace:  restTrace,
		Score:     restScore,
		Failures:  restFailures,
	}
	tr.MemoryREST.OutcomeFeedback = BuildOutcomeFeedback(task, tr.MemoryREST, lc)

	// MCP memory run
	mcpReq, bundleMCP, err := RecallMCP(ctx, compiler, task)
	if err != nil {
		return tr, fmt.Errorf("mcp recall: %w", err)
	}
	itemsMCP := FlattenBundle(bundleMCP)
	mcpFacts, mcpTrace := SimulateAgent(task, lc, RunModeMemory, itemsMCP)
	manifestMCP := BuildManifest(task.ID, task.ID+":memory:mcp", InterfaceMCP, RunModeMemory, *mcpReq, bundleMCP, lc)
	mcpScore, mcpFailures := ScoreRun(task, RunModeMemory, mcpFacts, manifestMCP, mcpTrace)
	tr.MemoryMCP = RunResult{
		TaskID:    task.ID,
		RunID:     task.ID + ":memory:mcp",
		Interface: InterfaceMCP,
		Mode:      RunModeMemory,
		AnswerFacts: mcpFacts,
		Manifest:  manifestMCP,
		UseTrace:  mcpTrace,
		Score:     mcpScore,
		Failures:  mcpFailures,
	}
	tr.MemoryMCP.OutcomeFeedback = BuildOutcomeFeedback(task, tr.MemoryMCP, lc)

	if task.CheckMCPRESTParity {
		ok, note := SameRecalledIDs(manifestREST, manifestMCP)
		tr.MCPRESTParity = ok && restScore.AnswerPass == mcpScore.AnswerPass
		tr.MCPRESTParityNote = note
		if restScore.AnswerPass != mcpScore.AnswerPass {
			tr.MCPRESTParity = false
			tr.MCPRESTParityNote = "answer/score mismatch between REST and MCP"
		}
	} else {
		tr.MCPRESTParity = true
	}

	tr.MemoryHelped = task.RequiresMemoryHelp &&
		!tr.NoMemory.Score.AnswerPass &&
		tr.MemoryREST.Score.AnswerPass &&
		tr.MemoryREST.Score.MemoryHelped &&
		len(tr.MemoryREST.UseTrace.UsedLabels) > 0

	return tr, nil
}

func filterUtility(lc *LoadedCorpus, task TaskFixture) map[string]utility.Score {
	out := map[string]utility.Score{}
	labels := append(append([]string{}, task.MemoryLabels...), task.DecoyLabels...)
	for _, lbl := range labels {
		if us, ok := lc.Utility[lbl]; ok {
			out[lbl] = us
		}
	}
	return out
}

// WriteReport writes benchmark JSON artifact.
func WriteReport(report *BenchmarkReport, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeJSON(path, report)
}
