import type { WorkspaceConfiguration } from "vscode";

export interface RecallResult {
  ok: boolean;
  status: number;
  body: string;
}

export interface HealthResult {
  ok: boolean;
  status: number;
  body: string;
}

function normalizeBase(u: string): string {
  return u.replace(/\/$/, "");
}

export function getBaseUrl(cfg: WorkspaceConfiguration): string {
  const u = cfg.get<string>("baseUrl") ?? "http://127.0.0.1:8123";
  return normalizeBase(u);
}

export async function jsonHeaders(cfg: WorkspaceConfiguration): Promise<Record<string, string>> {
  const h: Record<string, string> = { "Content-Type": "application/json" };
  const key = cfg.get<string>("apiKey") ?? "";
  if (key.trim().length > 0) {
    h["X-API-Key"] = key.trim();
  }
  return h;
}

export function plainHeaders(cfg: WorkspaceConfiguration): Record<string, string> {
  const h: Record<string, string> = {};
  const key = cfg.get<string>("apiKey") ?? "";
  if (key.trim().length > 0) {
    h["X-API-Key"] = key.trim();
  }
  return h;
}

export async function getHealth(
  baseUrl: string,
  headers: Record<string, string>
): Promise<HealthResult> {
  try {
    const res = await fetch(`${baseUrl}/healthz`, { method: "GET", headers });
    const body = await res.text();
    return { ok: res.ok, status: res.status, body };
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    return { ok: false, status: 0, body: msg };
  }
}

export async function postRecallCompile(
  baseUrl: string,
  headers: Record<string, string>,
  retrievalQuery: string,
  tags: string[],
  maxTotal: number
): Promise<RecallResult> {
  const body = JSON.stringify({
    retrieval_query: retrievalQuery,
    tags: tags.length ? tags : ["vscode"],
    max_total: maxTotal,
  });
  try {
    const res = await fetch(`${baseUrl}/v1/recall/compile`, {
      method: "POST",
      headers,
      body,
    });
    const text = await res.text();
    return { ok: res.ok, status: res.status, body: text };
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    return { ok: false, status: 0, body: msg };
  }
}

export async function postAdvisoryEpisode(
  baseUrl: string,
  headers: Record<string, string>,
  summary: string,
  source: string,
  tags: string[]
): Promise<RecallResult> {
  const body = JSON.stringify({
    summary: summary.trim(),
    source,
    tags,
  });
  try {
    const res = await fetch(`${baseUrl}/v1/advisory-episodes`, {
      method: "POST",
      headers,
      body,
    });
    const text = await res.text();
    return { ok: res.ok, status: res.status, body: text };
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    return { ok: false, status: 0, body: msg };
  }
}
