// Command pluribus-mcp is a thin stdio MCP server that proxies to control-plane HTTP only.
// Prefer MCP over HTTP on the service (POST /v1/mcp); see docs/mcp-migration-stdio-to-http.md.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"control-plane/internal/mcp"
)

func main() {
	log.SetOutput(os.Stderr)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	sc := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 64*1024)
	sc.Buffer(buf, 16*1024*1024)

	base := strings.TrimSpace(os.Getenv("CONTROL_PLANE_URL"))
	if base == "" {
		base = "http://127.0.0.1:8123"
	}
	base = strings.TrimRight(base, "/")
	apiKey := strings.TrimSpace(os.Getenv("CONTROL_PLANE_API_KEY"))
	client := &http.Client{Timeout: 10 * time.Minute}

	for sc.Scan() {
		line := sc.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			writeErr(nil, -32700, "parse error", nil)
			continue
		}
		if req.JSONRPC != "2.0" {
			writeErr(req.ID, -32600, "invalid request", nil)
			continue
		}
		if req.ID == nil && strings.HasPrefix(req.Method, "notifications/") {
			continue
		}
		if req.Method == "" {
			writeErr(req.ID, -32600, "missing method", nil)
			continue
		}

		switch req.Method {
		case "initialize":
			writeResult(req.ID, mcp.InitializeResult("pluribus-mcp", mcp.Version))
		case "ping":
			writeResult(req.ID, map[string]any{})
		case "tools/list":
			writeResult(req.ID, map[string]any{"tools": mcp.ToolDefinitions()})
		case "tools/call":
			res, err := mcp.HandleToolsCall(client, base, apiKey, req.Params, mcp.DefaultMemoryFormationPolicy())
			if err != nil {
				writeErr(req.ID, -32000, err.Error(), nil)
				continue
			}
			writeResult(req.ID, res)
		default:
			if req.ID == nil {
				continue
			}
			writeErr(req.ID, -32601, "method not found: "+req.Method, nil)
		}
	}
	return sc.Err()
}

type jsonRPCRequest struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params"`
}

func writeResult(id *json.RawMessage, result any) {
	out := map[string]any{"jsonrpc": "2.0", "result": result}
	if id != nil {
		out["id"] = json.RawMessage(*id)
	}
	b, _ := json.Marshal(out)
	fmt.Println(string(b))
}

func writeErr(id *json.RawMessage, code int, message string, data any) {
	errObj := map[string]any{"code": code, "message": message}
	if data != nil {
		errObj["data"] = data
	}
	out := map[string]any{"jsonrpc": "2.0", "error": errObj}
	if id != nil {
		out["id"] = json.RawMessage(*id)
	}
	b, _ := json.Marshal(out)
	fmt.Println(string(b))
}
