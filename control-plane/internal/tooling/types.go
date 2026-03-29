package tooling

import "context"

// GitDiffRequest is the payload for POST /tools/git/diff.
type GitDiffRequest struct {
	RepoPath string `json:"repo_path"` // working directory
	Base     string `json:"base"`      // base ref (e.g. main, HEAD~1)
	Head     string `json:"head"`      // head ref (e.g. HEAD)
}

// GitDiffResponse is the response from git diff.
type GitDiffResponse struct {
	Output string `json:"output"`
}

// RipgrepRequest is the payload for POST /tools/search/rg.
type RipgrepRequest struct {
	Pattern string   `json:"pattern"`
	Path    string   `json:"path"`    // directory to search
	Glob    []string `json:"glob,omitempty"` // e.g. ["*.go"]
}

// RipgrepResponse is the response from ripgrep.
type RipgrepResponse struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
}

// RunCommandRequest is the payload for POST /tools/test/run or /tools/build/run.
type RunCommandRequest struct {
	Cwd  string   `json:"cwd"`  // working directory
	Args []string `json:"args,omitempty"` // extra args (e.g. ./...)
}

// RunCommandResponse is the response from run command.
type RunCommandResponse struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
}

// --- LSP adapter (OE-4) ---

// LSPSymbolsRequest is the payload for POST /tools/lsp/symbols.
type LSPSymbolsRequest struct {
	Root string `json:"root"` // project root directory (working dir for the language server)
	Path string `json:"path"` // file path (relative to root or absolute)
}

// Symbol is a single document symbol for recall/drift (name, kind, range).
type Symbol struct {
	Name   string  `json:"name"`
	Kind   string  `json:"kind"`   // e.g. "function", "struct", "variable"
	Range  *Range  `json:"range"`  // start/end in file
	Detail string  `json:"detail,omitempty"`
	Children []Symbol `json:"children,omitempty"`
}

// Range is a start/end range (LSP-style 0-based line and character).
type Range struct {
	StartLine   int `json:"start_line"`
	StartCol    int `json:"start_col"`
	EndLine     int `json:"end_line"`
	EndCol      int `json:"end_col"`
}

// LSPSymbolsResponse is the response from LSP documentSymbol.
type LSPSymbolsResponse struct {
	Symbols []Symbol `json:"symbols"`
}

// LSPReferencesRequest is the payload for POST /tools/lsp/references.
type LSPReferencesRequest struct {
	Root   string `json:"root"`
	Path   string `json:"path"`
	Line   int    `json:"line"`   // 0-based line
	Column int    `json:"column"` // 0-based character
}

// Location is a single reference location (file path + range).
type Location struct {
	Path  string `json:"path"`
	Range *Range `json:"range"`
}

// LSPReferencesResponse is the response from LSP references.
type LSPReferencesResponse struct {
	References []Location `json:"references"`
}

// Reference is a single reference location (alias for Location for LSP client API). Task 100.
type Reference = Location

// LSPClient is the interface for LSP symbol and reference lookups (Task 100).
// Implementations wrap gopls or other language servers.
type LSPClient interface {
	FindSymbols(ctx context.Context, root, path string) ([]Symbol, error)
	FindReferences(ctx context.Context, root, path string, line, col int) ([]Reference, error)
}
