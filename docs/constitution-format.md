# Constitution format

The constitution (`constitution.md` at project root) defines stable project law. It is loaded first in the retrieval order (see [retrieval-order.md](retrieval-order.md)).

## Expected sections

Structure the file with clear sections so humans and tooling can rely on it. Recommended sections:

- **Purpose of the system** — What the system is for and its primary goals.
- **Domain boundaries** — What is in scope and out of scope.
- **Naming rules** — Conventions for packages, types, functions, files.
- **Forbidden patterns** — What must not be done (e.g. no global mutable state, no sync HTTP in hot path).
- **Non-negotiable architecture rules** — Invariants (e.g. layers, boundaries, entrypoints).
- **Coding style rules** — Formatting, comments, error handling.
- **Performance priorities** — Latency, throughput, or resource constraints that drive design.
- **Entrypoints and invariants** — Main APIs and invariants that must hold across changes.

Sections can be markdown headings (`## Purpose`, etc.). The file is read as a single blob and injected into the prompt; no strict parser is required. Keeping a consistent structure makes it easier to maintain and to extend with section-aware tooling later.
