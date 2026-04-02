/** Local counters for orchestrator verification — no telemetry network. */

export type EventSource =
  | "task_start"
  | "task_process_end"
  | "debug_start"
  | "debug_end"
  | "save"
  | "manual"
  | "health"
  | "config";

export class Metrics {
  autoRecallBySource: Record<string, number> = {
    task_start: 0,
    debug_start: 0,
    save: 0,
  };
  autoRecordBySource: Record<string, number> = {
    task_failure: 0,
    debug_end: 0,
  };
  manualRecall = 0;
  manualRecord = 0;
  failedRecall = 0;
  failedRecord = 0;
  failedHealth = 0;
  /** Nudge shown (e.g. network failure). */
  nudgesShown = 0;

  bumpAutoRecall(source: EventSource): void {
    const k = String(source);
    this.autoRecallBySource[k] = (this.autoRecallBySource[k] ?? 0) + 1;
  }

  bumpAutoRecord(kind: "task_failure" | "debug_end"): void {
    this.autoRecordBySource[kind] = (this.autoRecordBySource[kind] ?? 0) + 1;
  }

  summaryLine(): string {
    const ar = Object.entries(this.autoRecallBySource)
      .filter(([, v]) => v > 0)
      .map(([k, v]) => `${k}:${v}`)
      .join(" ");
    const rr = Object.entries(this.autoRecordBySource)
      .filter(([, v]) => v > 0)
      .map(([k, v]) => `${k}:${v}`)
      .join(" ");
    return [
      `autoRecall[${ar || "none"}]`,
      `autoRecord[${rr || "none"}]`,
      `manual R/R ${this.manualRecall}/${this.manualRecord}`,
      `fail r/R/h ${this.failedRecall}/${this.failedRecord}/${this.failedHealth}`,
      `nudge ${this.nudgesShown}`,
    ].join(" | ");
  }

  log(output: import("vscode").OutputChannel): void {
    const ts = new Date().toISOString();
    output.appendLine(`[metrics ${ts}] ${this.summaryLine()}`);
  }
}
