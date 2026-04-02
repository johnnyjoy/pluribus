import * as vscode from "vscode";
import {
  getBaseUrl,
  jsonHeaders,
  postRecallCompile,
  postAdvisoryEpisode,
  plainHeaders,
} from "./api";
import { Metrics } from "./metrics";
import { registerOrchestrator } from "./orchestrator";

const output = vscode.window.createOutputChannel("Pluribus");

function cfg(): vscode.WorkspaceConfiguration {
  return vscode.workspace.getConfiguration("pluribus");
}

class PluribusTree implements vscode.TreeDataProvider<vscode.TreeItem> {
  private _onDidChange = new vscode.EventEmitter<void>();
  readonly onDidChangeTreeData = this._onDidChange.event;

  lastRecall = "";
  lastRecord = "";
  lastPending = "";
  lastAutoRecall = "";
  lastAutoRecord = "";
  healthLine = "Health: (starting)";
  metricsLine = "";
  connectionOk: boolean | undefined = undefined;

  private metrics: Metrics;

  constructor(metrics: Metrics) {
    this.metrics = metrics;
  }

  refresh(): void {
    this.metricsLine = this.metrics.summaryLine();
    this._onDidChange.fire();
  }

  setRecall(s: string): void {
    this.lastRecall = s;
    this.refresh();
  }

  setRecord(s: string): void {
    this.lastRecord = s;
    this.refresh();
  }

  setPending(s: string): void {
    this.lastPending = s;
    this.refresh();
  }

  setAutoRecall(s: string): void {
    this.lastAutoRecall = s;
    this.refresh();
  }

  setAutoRecord(s: string): void {
    this.lastAutoRecord = s;
    this.refresh();
  }

  setHealth(ok: boolean | undefined, detail: string): void {
    this.connectionOk = ok;
    if (ok === undefined) {
      this.healthLine = "Health: unknown";
    } else if (ok) {
      this.healthLine = `Health: OK (${detail})`;
    } else {
      this.healthLine = `Health: down (${detail})`;
    }
    this.refresh();
  }

  getTreeItem(element: vscode.TreeItem): vscode.TreeItem {
    return element;
  }

  getChildren(): vscode.TreeItem[] {
    const trunc = (s: string, n: number) =>
      s.length <= n ? s : s.slice(0, n) + "…";
    const mk = (label: string, body: string, tip: string) => {
      const it = new vscode.TreeItem(label, vscode.TreeItemCollapsibleState.None);
      it.description = body ? trunc(body.replace(/\s+/g, " "), 72) : "(empty)";
      it.tooltip = tip || body;
      return it;
    };
    return [
      mk("Connection", this.healthLine, this.healthLine),
      mk("Session metrics", this.metricsLine, this.metricsLine),
      mk("Last auto recall", this.lastAutoRecall, this.lastAutoRecall),
      mk("Last auto record", this.lastAutoRecord, this.lastAutoRecord),
      mk("Last manual recall", this.lastRecall, this.lastRecall),
      mk("Last manual record", this.lastRecord, this.lastRecord),
      mk("Pending candidates", this.lastPending, this.lastPending),
    ];
  }
}

let tree: PluribusTree;
let metrics: Metrics;
let statusBar: vscode.StatusBarItem;

export function activate(context: vscode.ExtensionContext): void {
  metrics = new Metrics();
  tree = new PluribusTree(metrics);

  statusBar = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 100);
  statusBar.command = "pluribus.showOutput";
  context.subscriptions.push(statusBar);

  const updateStatusBar = (): void => {
    if (tree.connectionOk === false) {
      statusBar.text = "$(error) Pluribus";
      statusBar.tooltip = "Disconnected — click for output";
      statusBar.backgroundColor = new vscode.ThemeColor("statusBarItem.errorBackground");
    } else if (tree.connectionOk === true) {
      statusBar.text = "$(pass) Pluribus";
      statusBar.tooltip = `OK — ${metrics.summaryLine()}`;
      statusBar.backgroundColor = undefined;
    } else {
      statusBar.text = "$(question) Pluribus";
      statusBar.tooltip = "Checking…";
      statusBar.backgroundColor = undefined;
    }
    statusBar.show();
  };

  const markRecallOk = (): void => {
    /* reserved for future nudges based on recall recency */
  };

  const orch = registerOrchestrator(context, {
    output,
    metrics,
    setLastAutoRecall: (t) => {
      tree.setAutoRecall(t);
    },
    setLastAutoRecord: (t) => {
      tree.setAutoRecord(t);
    },
    setHealthOk: (ok, detail) => tree.setHealth(ok, detail),
    refreshTree: () => tree.refresh(),
    updateStatusBar,
    markRecallOk,
  });

  const logIntervalMin = cfg().get<number>("metrics.logIntervalMinutes") ?? 0;
  if (logIntervalMin > 0) {
    const logId = setInterval(() => {
      metrics.log(output);
    }, logIntervalMin * 60 * 1000);
    context.subscriptions.push({ dispose: () => clearInterval(logId) });
  }

  context.subscriptions.push(
    vscode.window.registerTreeDataProvider("pluribusSidebar", tree),
    vscode.commands.registerCommand("pluribus.recallContext", recallContext),
    vscode.commands.registerCommand("pluribus.recordExperience", recordExperience),
    vscode.commands.registerCommand("pluribus.viewLearnings", viewLearnings),
    vscode.commands.registerCommand("pluribus.refreshSidebar", () => tree.refresh()),
    vscode.commands.registerCommand("pluribus.checkHealth", async () => {
      await orch.checkHealth();
      vscode.window.showInformationMessage(
        tree.connectionOk ? "Pluribus: health OK" : "Pluribus: health check failed — see Output"
      );
    }),
    vscode.commands.registerCommand("pluribus.showMetrics", () => {
      metrics.log(output);
      output.show(true);
      vscode.window.showInformationMessage(metrics.summaryLine());
    }),
    vscode.commands.registerCommand("pluribus.showOutput", () => {
      output.show(true);
    }),
    output
  );

  updateStatusBar();
  metrics.log(output);
}

async function defaultTags(): Promise<string[]> {
  const raw = cfg().get<string>("defaultTags");
  if (raw && raw.trim().length > 0) {
    return raw
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);
  }
  return ["vscode"];
}

async function recallContext(): Promise<void> {
  const ws = vscode.workspace.name ?? "workspace";
  const q = await vscode.window.showInputBox({
    prompt: "Retrieval query (situation text)",
    value: `${ws}: `,
    placeHolder: "What are you trying to do?",
  });
  if (q === undefined) {
    return;
  }
  const tagStr = await vscode.window.showInputBox({
    prompt: "Tags (comma-separated, optional)",
    value: (await defaultTags()).join(", "),
  });
  if (tagStr === undefined) {
    return;
  }
  const tags = tagStr
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
  const base = getBaseUrl(cfg());
  try {
    const res = await postRecallCompile(
      base,
      await jsonHeaders(cfg()),
      q,
      tags.length ? tags : ["vscode"],
      32
    );
    metrics.manualRecall += 1;
    output.appendLine(`[recall manual] HTTP ${res.status}`);
    output.appendLine(res.body);
    output.show(true);
    tree.setRecall(res.body);
    if (!res.ok) {
      metrics.failedRecall += 1;
      vscode.window.showWarningMessage(`Pluribus recall: HTTP ${res.status}`);
    }
    tree.refresh();
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    vscode.window.showErrorMessage(`Pluribus recall failed: ${msg}`);
    output.appendLine(`[recall] error ${msg}`);
    output.show(true);
    metrics.failedRecall += 1;
  }
}

async function recordExperience(): Promise<void> {
  const summary = await vscode.window.showInputBox({
    prompt: "Experience summary (advisory episode)",
    placeHolder: "What happened? What did we learn?",
  });
  if (summary === undefined || summary.trim() === "") {
    return;
  }
  const base = getBaseUrl(cfg());
  try {
    const res = await postAdvisoryEpisode(
      base,
      await jsonHeaders(cfg()),
      summary.trim(),
      "vscode-manual",
      [...(await defaultTags()), "extension"]
    );
    metrics.manualRecord += 1;
    output.appendLine(`[record manual] HTTP ${res.status}`);
    output.appendLine(res.body);
    output.show(true);
    tree.setRecord(res.body);
    if (!res.ok) {
      metrics.failedRecord += 1;
      vscode.window.showWarningMessage(
        `Pluribus record: HTTP ${res.status} (similarity disabled on server?)`
      );
    } else {
      vscode.window.showInformationMessage("Pluribus: advisory episode recorded.");
    }
    tree.refresh();
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    vscode.window.showErrorMessage(`Pluribus record failed: ${msg}`);
    metrics.failedRecord += 1;
  }
}

async function viewLearnings(): Promise<void> {
  try {
    const base = getBaseUrl(cfg());
    const res = await fetch(`${base}/v1/curation/pending`, {
      method: "GET",
      headers: plainHeaders(cfg()),
    });
    const text = await res.text();
    output.appendLine(`[pending] HTTP ${res.status}`);
    output.appendLine(text);
    output.show(true);
    tree.setPending(text);
    if (!res.ok) {
      vscode.window.showWarningMessage(`Pluribus pending: HTTP ${res.status}`);
    }
    tree.refresh();
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    vscode.window.showErrorMessage(`Pluribus pending failed: ${msg}`);
  }
}

export function deactivate(): void {}
