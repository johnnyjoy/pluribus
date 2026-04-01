import * as vscode from "vscode";

const output = vscode.window.createOutputChannel("Pluribus");

function cfg(): vscode.WorkspaceConfiguration {
  return vscode.workspace.getConfiguration("pluribus");
}

async function jsonHeaders(): Promise<Record<string, string>> {
  const h: Record<string, string> = { "Content-Type": "application/json" };
  const key = cfg().get<string>("apiKey") ?? "";
  if (key.trim().length > 0) {
    h["X-API-Key"] = key.trim();
  }
  return h;
}

function getHeaders(): Record<string, string> {
  const h: Record<string, string> = {};
  const key = cfg().get<string>("apiKey") ?? "";
  if (key.trim().length > 0) {
    h["X-API-Key"] = key.trim();
  }
  return h;
}

function baseUrl(): string {
  const u = cfg().get<string>("baseUrl") ?? "http://127.0.0.1:8123";
  return u.replace(/\/$/, "");
}

class PluribusTree implements vscode.TreeDataProvider<vscode.TreeItem> {
  private _onDidChange = new vscode.EventEmitter<void>();
  readonly onDidChangeTreeData = this._onDidChange.event;

  lastRecall = "";
  lastRecord = "";
  lastPending = "";

  refresh(): void {
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

  getTreeItem(element: vscode.TreeItem): vscode.TreeItem {
    return element;
  }

  getChildren(): vscode.TreeItem[] {
    const trunc = (s: string, n: number) =>
      s.length <= n ? s : s.slice(0, n) + "…";
    const mk = (label: string, body: string, tip: string) => {
      const it = new vscode.TreeItem(label, vscode.TreeItemCollapsibleState.None);
      it.description = body ? trunc(body.replace(/\s+/g, " "), 80) : "(empty)";
      it.tooltip = tip || body;
      return it;
    };
    return [
      mk("Last recall", this.lastRecall, this.lastRecall),
      mk("Last record", this.lastRecord, this.lastRecord),
      mk("Pending candidates", this.lastPending, this.lastPending),
    ];
  }
}

let tree: PluribusTree;

export function activate(context: vscode.ExtensionContext): void {
  tree = new PluribusTree();

  context.subscriptions.push(
    vscode.window.registerTreeDataProvider("pluribusSidebar", tree),
    vscode.commands.registerCommand("pluribus.recallContext", recallContext),
    vscode.commands.registerCommand("pluribus.recordExperience", recordExperience),
    vscode.commands.registerCommand("pluribus.viewLearnings", viewLearnings),
    vscode.commands.registerCommand("pluribus.refreshSidebar", () => tree.refresh()),
    output
  );
}

async function recallContext(): Promise<void> {
  const q = await vscode.window.showInputBox({
    prompt: "Retrieval query (situation text)",
    placeHolder: "What are you trying to do?",
  });
  if (q === undefined) {
    return;
  }
  const tagStr = await vscode.window.showInputBox({
    prompt: "Tags (comma-separated, optional)",
    value: "vscode",
  });
  if (tagStr === undefined) {
    return;
  }
  const tags = tagStr
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
  const body = JSON.stringify({
    retrieval_query: q,
    tags: tags.length ? tags : ["vscode"],
    max_total: 32,
  });
  try {
    const res = await fetch(`${baseUrl()}/v1/recall/compile`, {
      method: "POST",
      headers: await jsonHeaders(),
      body,
    });
    const text = await res.text();
    output.appendLine(`[recall] HTTP ${res.status}`);
    output.appendLine(text);
    output.show(true);
    tree.setRecall(text);
    if (!res.ok) {
      vscode.window.showWarningMessage(`Pluribus recall: HTTP ${res.status}`);
    }
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    vscode.window.showErrorMessage(`Pluribus recall failed: ${msg}`);
    output.appendLine(`[recall] error ${msg}`);
    output.show(true);
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
  const body = JSON.stringify({
    summary: summary.trim(),
    source: "manual",
    tags: ["vscode", "extension"],
  });
  try {
    const res = await fetch(`${baseUrl()}/v1/advisory-episodes`, {
      method: "POST",
      headers: await jsonHeaders(),
      body,
    });
    const text = await res.text();
    output.appendLine(`[record] HTTP ${res.status}`);
    output.appendLine(text);
    output.show(true);
    tree.setRecord(text);
    if (!res.ok) {
      vscode.window.showWarningMessage(
        `Pluribus record: HTTP ${res.status} (similarity disabled on server?)`
      );
    } else {
      vscode.window.showInformationMessage("Pluribus: advisory episode recorded.");
    }
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    vscode.window.showErrorMessage(`Pluribus record failed: ${msg}`);
  }
}

async function viewLearnings(): Promise<void> {
  try {
    const res = await fetch(`${baseUrl()}/v1/curation/pending`, {
      method: "GET",
      headers: getHeaders(),
    });
    const text = await res.text();
    output.appendLine(`[pending] HTTP ${res.status}`);
    output.appendLine(text);
    output.show(true);
    tree.setPending(text);
    if (!res.ok) {
      vscode.window.showWarningMessage(`Pluribus pending: HTTP ${res.status}`);
    }
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    vscode.window.showErrorMessage(`Pluribus pending failed: ${msg}`);
  }
}

export function deactivate(): void {}
