package tooling

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultLSPCommand is the default command used for Go LSP (gopls).
const DefaultLSPCommand = "gopls"

// lspMessage is the LSP wire format: Content-Length header + JSON body.
const lspHeaderPrefix = "Content-Length: "

// runLSP runs an LSP command (e.g. gopls) with stdio, sends init + didOpen, then runs fn which can send more requests.
func runLSP(ctx context.Context, root, path string, content []byte, fn func(*lspSession) error) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("root path: %w", err)
	}
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(absRoot, path)
	} else {
		absPath, err = filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("path: %w", err)
		}
	}
	uri := "file://" + filepath.ToSlash(absPath)
	rootURI := "file://" + filepath.ToSlash(absRoot)

	cmd := exec.CommandContext(ctx, DefaultLSPCommand, "serve")
	cmd.Dir = absRoot
	cmd.Stderr = nil
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", DefaultLSPCommand, err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	session := &lspSession{
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}
	// Initialize
	if err := session.initialize(rootURI); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	if err := session.initialized(); err != nil {
		return fmt.Errorf("initialized: %w", err)
	}
	if err := session.didOpen(uri, content); err != nil {
		return fmt.Errorf("didOpen: %w", err)
	}
	return fn(session)
}

type lspSession struct {
	stdin  io.Writer
	stdout *bufio.Reader
	mu     sync.Mutex
	nextID int
}

func (s *lspSession) nextRequestID() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	return s.nextID
}

func (s *lspSession) send(req interface{}) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("%s%d\r\n\r\n", lspHeaderPrefix, len(body))
	if _, err := s.stdin.Write([]byte(header)); err != nil {
		return err
	}
	_, err = s.stdin.Write(body)
	return err
}

// readMessage reads one LSP message (Content-Length: N\r\n\r\n + N bytes).
func (s *lspSession) readMessage() ([]byte, error) {
	line, err := s.stdout.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, lspHeaderPrefix) {
		return nil, fmt.Errorf("lsp: expected Content-Length header, got %q", line)
	}
	n, err := strconv.Atoi(strings.TrimSpace(line[len(lspHeaderPrefix):]))
	if err != nil || n <= 0 {
		return nil, fmt.Errorf("lsp: invalid Content-Length: %s", line)
	}
	// Consume blank line (\r\n or \n) after header
	_, _ = s.stdout.ReadString('\n')
	body := make([]byte, n)
	_, err = io.ReadFull(s.stdout, body)
	return body, err
}

func (s *lspSession) recv(id int) (json.RawMessage, error) {
	for {
		body, err := s.readMessage()
		if err != nil {
			return nil, err
		}
		var raw struct {
			ID     *int            `json:"id"`
			Result json.RawMessage `json:"result,omitempty"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error,omitempty"`
		}
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, err
		}
		if raw.Error != nil {
			return nil, fmt.Errorf("lsp error %d: %s", raw.Error.Code, raw.Error.Message)
		}
		if raw.ID != nil && *raw.ID == id {
			return raw.Result, nil
		}
		// Notification or other response; skip
	}
}

func (s *lspSession) initialize(rootURI string) error {
	id := s.nextRequestID()
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "initialize",
		"params": map[string]interface{}{
			"processId": nil,
			"rootUri":   rootURI,
			"capabilities": map[string]interface{}{},
		},
	}
	if err := s.send(req); err != nil {
		return err
	}
	_, err := s.recv(id)
	return err
}

func (s *lspSession) initialized() error {
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialized",
		"params":  map[string]interface{}{},
	}
	return s.send(req)
}

func (s *lspSession) didOpen(uri string, content []byte) error {
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri":        uri,
				"languageId": "go",
				"version":    1,
				"text":       string(content),
			},
		},
	}
	return s.send(req)
}

// documentSymbolRaw decodes LSP documentSymbol result (DocumentSymbol[] or SymbolInformation[]).
type documentSymbolRaw struct {
	Name          string             `json:"name"`
	Detail        string             `json:"detail"`
	Kind          int                `json:"kind"`
	Range         lspRange           `json:"range"`
	SelectionRange *lspRange          `json:"selectionRange,omitempty"`
	Children      []documentSymbolRaw `json:"children,omitempty"`
	Location      *struct {
		URI   string  `json:"uri"`
		Range lspRange `json:"range"`
	} `json:"location,omitempty"`
}

type lspRange struct {
	Start struct {
		Line      int `json:"line"`
		Character int `json:"character"`
	} `json:"start"`
	End struct {
		Line      int `json:"line"`
		Character int `json:"character"`
	} `json:"end"`
}

var symbolKindNames = map[int]string{
	1: "file", 2: "module", 3: "namespace", 4: "package", 5: "class", 6: "method",
	7: "property", 8: "field", 9: "constructor", 10: "enum", 11: "interface",
	12: "function", 13: "variable", 14: "constant", 15: "string", 16: "number",
	17: "boolean", 18: "array", 19: "object", 20: "key", 21: "null",
	22: "enum_member", 23: "struct", 24: "event", 25: "operator", 26: "type_parameter",
}

func symbolKindString(k int) string {
	if s, ok := symbolKindNames[k]; ok {
		return s
	}
	return "unknown"
}

func lspRangeToRange(lr lspRange) *Range {
	return &Range{
		StartLine: lr.Start.Line,
		StartCol:  lr.Start.Character,
		EndLine:   lr.End.Line,
		EndCol:    lr.End.Character,
	}
}

func (s *lspSession) documentSymbol(uri string) ([]Symbol, error) {
	id := s.nextRequestID()
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "textDocument/documentSymbol",
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
		},
	}
	if err := s.send(req); err != nil {
		return nil, err
	}
	rawResult, err := s.recv(id)
	if err != nil {
		return nil, err
	}
	var raw []documentSymbolRaw
	if err := json.Unmarshal(rawResult, &raw); err != nil {
		return nil, fmt.Errorf("documentSymbol result: %w", err)
	}
	return flattenDocumentSymbols(raw, nil), nil
}

func flattenDocumentSymbols(raw []documentSymbolRaw, out *[]Symbol) []Symbol {
	if out == nil {
		var s []Symbol
		out = &s
	}
	for _, r := range raw {
		sym := Symbol{
			Name:   r.Name,
			Kind:   symbolKindString(r.Kind),
			Detail: r.Detail,
		}
		if r.Range.Start.Line != 0 || r.Range.End.Line != 0 || r.SelectionRange != nil {
			if r.SelectionRange != nil {
				sym.Range = lspRangeToRange(*r.SelectionRange)
			} else {
				sym.Range = lspRangeToRange(r.Range)
			}
		}
		if len(r.Children) > 0 {
			sym.Children = flattenDocumentSymbols(r.Children, nil)
		}
		*out = append(*out, sym)
	}
	return *out
}

func (s *lspSession) references(uri string, line, col int) ([]Location, error) {
	id := s.nextRequestID()
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "textDocument/references",
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
			"position":     map[string]int{"line": line, "character": col},
			"context":     map[string]bool{"includeDeclaration": true},
		},
	}
	if err := s.send(req); err != nil {
		return nil, err
	}
	rawResult, err := s.recv(id)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		URI   string   `json:"uri"`
		Range lspRange `json:"range"`
	}
	if err := json.Unmarshal(rawResult, &raw); err != nil {
		return nil, fmt.Errorf("references result: %w", err)
	}
	out := make([]Location, 0, len(raw))
	for _, loc := range raw {
		path, err := uriToPath(loc.URI)
		if err != nil {
			continue
		}
		out = append(out, Location{
			Path:  path,
			Range: lspRangeToRange(loc.Range),
		})
	}
	return out, nil
}

func uriToPath(uri string) (string, error) {
	if !strings.HasPrefix(uri, "file://") {
		return "", fmt.Errorf("unsupported uri: %s", uri)
	}
	u, err := url.PathUnescape(uri[7:])
	if err != nil {
		return "", err
	}
	return filepath.FromSlash(u), nil
}

// LSPSymbols returns document symbols for the given file by calling gopls (or DefaultLSPCommand).
func LSPSymbols(ctx context.Context, root, path string) ([]Symbol, error) {
	var err error
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(root, path)
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return nil, err
	}
	var content []byte
	content, err = os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	fileURI := "file://" + filepath.ToSlash(absPath)
	var symbols []Symbol
	err = runLSP(ctx, root, path, content, func(s *lspSession) error {
		var err error
		symbols, err = s.documentSymbol(fileURI)
		return err
	})
	return symbols, err
}

// LSPReferences returns references for the symbol at the given position.
func LSPReferences(ctx context.Context, root, path string, line, col int) ([]Location, error) {
	var err error
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(root, path)
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return nil, err
	}
	var content []byte
	content, err = os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	fileURI := "file://" + filepath.ToSlash(absPath)
	var refs []Location
	err = runLSP(ctx, root, path, content, func(s *lspSession) error {
		var err error
		refs, err = s.references(fileURI, line, col)
		return err
	})
	return refs, err
}

// GoplsClient implements LSPClient using the default gopls command (Task 100).
type GoplsClient struct{}

// FindSymbols returns document symbols for the given file.
func (GoplsClient) FindSymbols(ctx context.Context, root, path string) ([]Symbol, error) {
	return LSPSymbols(ctx, root, path)
}

// FindReferences returns references for the symbol at the given position.
func (GoplsClient) FindReferences(ctx context.Context, root, path string, line, col int) ([]Reference, error) {
	return LSPReferences(ctx, root, path, line, col)
}

// Ensure GoplsClient implements LSPClient.
var _ LSPClient = GoplsClient{}
