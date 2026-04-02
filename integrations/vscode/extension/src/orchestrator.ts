import * as vscode from "vscode";
import {
  getBaseUrl,
  jsonHeaders,
  postAdvisoryEpisode,
  postRecallCompile,
  getHealth,
  plainHeaders,
} from "./api";
import type { Metrics } from "./metrics";

export interface OrchestratorDeps {
  output: vscode.OutputChannel;
  metrics: Metrics;
  setLastAutoRecall: (text: string) => void;
  setLastAutoRecord: (text: string) => void;
  setHealthOk: (ok: boolean, detail: string) => void;
  refreshTree: () => void;
  updateStatusBar: () => void;
  /** Last time a recall succeeded (ms), for soft nudges */
  markRecallOk: () => void;
}

function defaultTags(cfg: vscode.WorkspaceConfiguration): string[] {
  const raw = cfg.get<string>("defaultTags");
  if (raw && raw.trim().length > 0) {
    return raw
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);
  }
  return ["vscode"];
}

function orchestratorEnabled(cfg: vscode.WorkspaceConfiguration): boolean {
  return cfg.get<boolean>("orchestrator.enabled") !== false;
}

export function registerOrchestrator(
  context: vscode.ExtensionContext,
  deps: OrchestratorDeps
): { checkHealth: () => Promise<boolean> } {
  let nudgedDisconnected = false;
  let consecutiveTaskFailures = 0;
  const saveTimers = new Map<string, ReturnType<typeof setTimeout>>();

  const cfg = () => vscode.workspace.getConfiguration("pluribus");

  async function runHealth(reason: string): Promise<boolean> {
    const c = cfg();
    const base = getBaseUrl(c);
    const h = plainHeaders(c);
    deps.output.appendLine(`[health] ${reason} → GET ${base}/healthz`);
    const res = await getHealth(base, h);
    const ok = res.ok && res.status >= 200 && res.status < 300;
    const detail = ok ? `HTTP ${res.status}` : `HTTP ${res.status} ${res.body.slice(0, 200)}`;
    deps.setHealthOk(ok, detail);
    if (!ok) {
      deps.metrics.failedHealth += 1;
    }
    deps.updateStatusBar();
    deps.refreshTree();
    return ok;
  }

  /** Fire-and-forget recall with logging. */
  async function autoRecall(
    source: "task_start" | "debug_start" | "save",
    retrievalQuery: string
  ): Promise<void> {
    const c = cfg();
    if (!orchestratorEnabled(c)) {
      return;
    }
    if (source === "task_start" && c.get<boolean>("orchestrator.recallOnTaskStart") === false) {
      return;
    }
    if (source === "debug_start" && c.get<boolean>("orchestrator.recallOnDebugStart") === false) {
      return;
    }
    if (source === "save" && c.get<boolean>("orchestrator.recallOnSave") !== true) {
      return;
    }

    const base = getBaseUrl(c);
    const headers = await jsonHeaders(c);
    const maxTotal = c.get<number>("orchestrator.maxRecallTotal") ?? 24;
    deps.output.appendLine(`[auto-recall ${source}] ${retrievalQuery.slice(0, 200)}`);
    const result = await postRecallCompile(base, headers, retrievalQuery, defaultTags(c), maxTotal);
    deps.metrics.bumpAutoRecall(source);
    if (result.ok) {
      deps.setHealthOk(true, `recall ${source} HTTP ${result.status}`);
      if (source === "task_start") {
        deps.markRecallOk();
        consecutiveTaskFailures = 0;
      }
      deps.setLastAutoRecall(result.body);
      deps.output.appendLine(`[auto-recall ${source}] HTTP ${result.status} ok`);
    } else {
      deps.metrics.failedRecall += 1;
      deps.output.appendLine(`[auto-recall ${source}] HTTP ${result.status} ${result.body.slice(0, 500)}`);
      if (result.status === 0 && c.get<boolean>("nudges.whenDisconnected") !== false && !nudgedDisconnected) {
        nudgedDisconnected = true;
        deps.metrics.nudgesShown += 1;
        void vscode.window.showWarningMessage(
          "Pluribus: recall failed (network?) — check base URL and control plane.",
          "Show output"
        ).then((sel) => {
          if (sel === "Show output") {
            deps.output.show(true);
          }
        });
      }
      deps.setHealthOk(false, `recall failed HTTP ${result.status}`);
    }
    deps.updateStatusBar();
    deps.refreshTree();
  }

  async function autoRecord(
    kind: "task_failure" | "debug_end",
    summary: string
  ): Promise<void> {
    const c = cfg();
    if (!orchestratorEnabled(c)) {
      return;
    }
    if (kind === "task_failure" && c.get<boolean>("orchestrator.recordOnTaskProcessFailure") === false) {
      return;
    }
    if (kind === "debug_end" && c.get<boolean>("orchestrator.recordOnDebugEnd") !== true) {
      return;
    }

    const base = getBaseUrl(c);
    const headers = await jsonHeaders(c);
    const tags = [...defaultTags(c), "orchestrator", kind];
    deps.output.appendLine(`[auto-record ${kind}] ${summary.slice(0, 300)}`);
    const result = await postAdvisoryEpisode(
      base,
      headers,
      summary,
      "vscode-orchestrator",
      tags
    );
    deps.metrics.bumpAutoRecord(kind);
    if (result.ok) {
      deps.setHealthOk(true, `record ${kind} HTTP ${result.status}`);
      deps.setLastAutoRecord(result.body);
      deps.output.appendLine(`[auto-record ${kind}] HTTP ${result.status} ok`);
    } else {
      deps.metrics.failedRecord += 1;
      deps.output.appendLine(`[auto-record ${kind}] HTTP ${result.status} ${result.body.slice(0, 500)}`);
      deps.setHealthOk(false, `record failed HTTP ${result.status}`);
    }
    deps.updateStatusBar();
    deps.refreshTree();
  }

  // —— startup + interval health ——
  void runHealth("activate");
  const intervalMin = cfg().get<number>("healthCheckIntervalMinutes") ?? 5;
  if (intervalMin > 0) {
    const id = setInterval(() => {
      void runHealth("interval");
    }, intervalMin * 60 * 1000);
    context.subscriptions.push({ dispose: () => clearInterval(id) });
  }

  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration("pluribus")) {
        nudgedDisconnected = false;
        void runHealth("config change");
      }
    })
  );

  // —— tasks ——
  context.subscriptions.push(
    vscode.tasks.onDidStartTask((e) => {
      const task = e.execution.task;
      const name = task.name;
      const ws = vscode.workspace.name ?? "workspace";
      const q = `VS Code task starting: "${name}" (workspace: ${ws})`;
      void autoRecall("task_start", q);
    })
  );

  context.subscriptions.push(
    vscode.tasks.onDidEndTaskProcess((e) => {
      const exitCode = e.exitCode;
      const task = e.execution.task;
      const name = task.name;
      const ws = vscode.workspace.name ?? "workspace";
      if (exitCode === undefined) {
        return;
      }
      if (exitCode === 0) {
        consecutiveTaskFailures = 0;
        return;
      }
      consecutiveTaskFailures += 1;
      const summary = `VS Code task failed: "${name}" exitCode=${exitCode} workspace=${ws}`;
      void autoRecord("task_failure", summary);

      const threshold = cfg().get<number>("nudges.afterConsecutiveTaskFailures") ?? 0;
      if (threshold > 0 && consecutiveTaskFailures === threshold) {
        deps.metrics.nudgesShown += 1;
        void vscode.window.showInformationMessage(
          `Pluribus: ${consecutiveTaskFailures} task failures in a row. Consider running "Pluribus: Recall Context" with your situation.`,
          "OK"
        );
      }
    })
  );

  // —— debug ——
  context.subscriptions.push(
    vscode.debug.onDidStartDebugSession((session) => {
      const ws = vscode.workspace.name ?? "workspace";
      const q = `VS Code debug starting: "${session.name}" type=${session.type} (workspace: ${ws})`;
      void autoRecall("debug_start", q);
    })
  );

  context.subscriptions.push(
    vscode.debug.onDidTerminateDebugSession((session) => {
      if (cfg().get<boolean>("orchestrator.recordOnDebugEnd") !== true) {
        return;
      }
      const ws = vscode.workspace.name ?? "workspace";
      const summary = `VS Code debug session ended: "${session.name}" type=${session.type} workspace=${ws}`;
      void autoRecord("debug_end", summary);
    })
  );

  // —— save (optional, debounced) ——
  context.subscriptions.push(
    vscode.workspace.onDidSaveTextDocument((doc) => {
      const c = cfg();
      if (c.get<boolean>("orchestrator.recallOnSave") !== true) {
        return;
      }
      if (doc.uri.scheme !== "file") {
        return;
      }
      const debounce = c.get<number>("orchestrator.saveDebounceMs") ?? 2500;
      const key = doc.uri.toString();
      const prev = saveTimers.get(key);
      if (prev) {
        clearTimeout(prev);
      }
      const t = setTimeout(() => {
        saveTimers.delete(key);
        const ws = vscode.workspace.name ?? "workspace";
        const rel = vscode.workspace.asRelativePath(doc.uri);
        void autoRecall("save", `VS Code file saved: ${rel} (workspace: ${ws})`);
      }, debounce);
      saveTimers.set(key, t);
    })
  );

  return { checkHealth: () => runHealth("manual") };
}
