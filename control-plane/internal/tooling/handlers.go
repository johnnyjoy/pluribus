package tooling

import (
	"net/http"

	"control-plane/internal/httpx"
)

// Handlers provides HTTP handlers for tooling (git diff, ripgrep, test/build run).
type Handlers struct {
	Allowlist *Allowlist // optional; when set, enforces allowed_commands and allowed_dirs
}

// GitDiff handles POST /tools/git/diff.
func (h *Handlers) GitDiff(w http.ResponseWriter, r *http.Request) {
	var req GitDiffRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.RepoPath == "" {
		httpx.WriteError(w, http.StatusBadRequest, "repo_path required")
		return
	}
	if h.Allowlist != nil {
		if !h.Allowlist.CheckCommand("git") {
			httpx.WriteError(w, http.StatusForbidden, "command not allowed")
			return
		}
		if !h.Allowlist.CheckDir(req.RepoPath) {
			httpx.WriteError(w, http.StatusForbidden, "path not allowed")
			return
		}
	}
	if req.Base == "" {
		req.Base = "HEAD"
	}
	if req.Head == "" {
		req.Head = "HEAD"
	}
	output, err := GitDiff(r.Context(), req.RepoPath, req.Base, req.Head)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, GitDiffResponse{Output: output})
}

// Ripgrep handles POST /tools/search/rg.
func (h *Handlers) Ripgrep(w http.ResponseWriter, r *http.Request) {
	var req RipgrepRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Pattern == "" || req.Path == "" {
		httpx.WriteError(w, http.StatusBadRequest, "pattern and path required")
		return
	}
	if h.Allowlist != nil {
		if !h.Allowlist.CheckCommand("rg") {
			httpx.WriteError(w, http.StatusForbidden, "command not allowed")
			return
		}
		if !h.Allowlist.CheckDir(req.Path) {
			httpx.WriteError(w, http.StatusForbidden, "path not allowed")
			return
		}
	}
	output, exitCode, err := Ripgrep(r.Context(), req.Pattern, req.Path, req.Glob)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, RipgrepResponse{Output: output, ExitCode: exitCode})
}

// RunTest handles POST /tools/test/run.
func (h *Handlers) RunTest(w http.ResponseWriter, r *http.Request) {
	var req RunCommandRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Cwd == "" {
		httpx.WriteError(w, http.StatusBadRequest, "cwd required")
		return
	}
	if h.Allowlist != nil {
		if !h.Allowlist.CheckCommand("go") {
			httpx.WriteError(w, http.StatusForbidden, "command not allowed")
			return
		}
		if !h.Allowlist.CheckDir(req.Cwd) {
			httpx.WriteError(w, http.StatusForbidden, "path not allowed")
			return
		}
	}
	output, exitCode, err := RunTest(r.Context(), req.Cwd, req.Args)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, RunCommandResponse{Output: output, ExitCode: exitCode})
}

// RunBuild handles POST /tools/build/run.
func (h *Handlers) RunBuild(w http.ResponseWriter, r *http.Request) {
	var req RunCommandRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Cwd == "" {
		httpx.WriteError(w, http.StatusBadRequest, "cwd required")
		return
	}
	if h.Allowlist != nil {
		if !h.Allowlist.CheckCommand("go") {
			httpx.WriteError(w, http.StatusForbidden, "command not allowed")
			return
		}
		if !h.Allowlist.CheckDir(req.Cwd) {
			httpx.WriteError(w, http.StatusForbidden, "path not allowed")
			return
		}
	}
	output, exitCode, err := RunBuild(r.Context(), req.Cwd, req.Args)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, RunCommandResponse{Output: output, ExitCode: exitCode})
}

// LSPSymbols handles POST /tools/lsp/symbols (OE-4).
func (h *Handlers) LSPSymbols(w http.ResponseWriter, r *http.Request) {
	var req LSPSymbolsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Root == "" || req.Path == "" {
		httpx.WriteError(w, http.StatusBadRequest, "root and path required")
		return
	}
	if h.Allowlist != nil {
		if !h.Allowlist.CheckCommand(DefaultLSPCommand) {
			httpx.WriteError(w, http.StatusForbidden, "lsp command not allowed")
			return
		}
		if !h.Allowlist.CheckDir(req.Root) {
			httpx.WriteError(w, http.StatusForbidden, "path not allowed")
			return
		}
	}
	symbols, err := LSPSymbols(r.Context(), req.Root, req.Path)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, LSPSymbolsResponse{Symbols: symbols})
}

// LSPReferences handles POST /tools/lsp/references (OE-4).
func (h *Handlers) LSPReferences(w http.ResponseWriter, r *http.Request) {
	var req LSPReferencesRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Root == "" || req.Path == "" {
		httpx.WriteError(w, http.StatusBadRequest, "root and path required")
		return
	}
	if h.Allowlist != nil {
		if !h.Allowlist.CheckCommand(DefaultLSPCommand) {
			httpx.WriteError(w, http.StatusForbidden, "lsp command not allowed")
			return
		}
		if !h.Allowlist.CheckDir(req.Root) {
			httpx.WriteError(w, http.StatusForbidden, "path not allowed")
			return
		}
	}
	refs, err := LSPReferences(r.Context(), req.Root, req.Path, req.Line, req.Column)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, LSPReferencesResponse{References: refs})
}
