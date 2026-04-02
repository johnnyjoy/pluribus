# Pluribus AI — Institutional Memory for AI

**Memory-enhanced context. Make your AI actually remember.**

Pluribus adds **persistent, structured memory** to your development workflow.
It helps AI agents retain knowledge across sessions, learn from failures, and build understanding over time.

---

## 🧠 What this extension does

This extension connects your editor to a running Pluribus instance and makes memory:

* **automatic** — recall happens during real work
* **continuous** — experience is captured as you go
* **persistent** — knowledge survives across sessions

You keep using your tools the same way — Pluribus works alongside them.

---

## ⚙️ What you get

### 🔁 Automatic memory loop

* Recall context before tasks and debug runs
* Record experience after failures (and optionally after debug sessions)

### 📡 Live connection to Pluribus

* Health status in the status bar
* Clear indication when memory is active or disconnected

### 📊 Activity & metrics

* See recall/record activity in real time
* Lightweight logging in the Output panel

### 🛠 Manual controls (when you need them)

* Recall context on demand
* Record experience manually
* View pending learnings

---

## 🧩 How it fits

This extension is a **local orchestrator**, not the memory system itself.

* Pluribus (server) → owns memory, ranking, and truth
* Extension → connects your editor and triggers usage at the right moments

This keeps behavior consistent and avoids split logic.

---

## 🚀 Getting started

### 1. Run Pluribus

Start a local or remote Pluribus instance (Docker or existing deployment).

### 2. Configure the extension

Set in VS Code settings (default API port is **8123**):

```json
{
  "pluribus.baseUrl": "http://127.0.0.1:8123"
}
```

If authentication is enabled:

```json
{
  "pluribus.apiKey": "your-api-key"
}
```

### 3. Start working

* Run a task
* Start debugging
* Fix a failure

Pluribus will begin capturing and recalling context automatically.

**Extension ID:** `pluribus.pluribus-ai`. If you previously installed **`pluribus.pluribus-memory`**, uninstall the old extension before installing this one — the identifier changed.

---

## 🧠 How it helps

Without memory:

* every session starts from scratch
* failures are rediscovered
* decisions are lost

With Pluribus:

* past failures are remembered
* constraints are preserved
* context builds over time

---

## 🔌 MCP / Agent Integration

This extension handles editor-side orchestration.

If you want AI agents to use Pluribus directly (recommended):

* configure MCP separately
* use tools like `recall_context` and `record_experience`

See the [Pluribus repository](https://github.com/johnnyjoy/pluribus) for setup instructions.

---

## ⚠️ Requirements

* A running Pluribus control plane
* Network access to that instance
* Optional: API key if enabled

---

## 🧪 Current behavior (v0.2.x)

* Automatic recall on:

  * task start
  * debug start
* Automatic recording on:

  * task failure
  * optional debug end
* Optional recall on save (disabled by default)

This is **soft enforcement** — no blocking, no interruptions.

Full settings table, events, and install-from-source steps: **[`../README.md`](../README.md)** (parent folder in this repo).

---

## 🧭 Philosophy

Pluribus is not a search tool.

It is:

> **a persistent memory layer for AI systems**

This extension brings that memory into your editor — quietly, continuously, and usefully.

## License

Same terms as the [Pluribus repository](https://github.com/johnnyjoy/pluribus) (see root `LICENSE.md`).
