// Command mcp-stdio-proof exercises pluribus-mcp stdio JSON-RPC against a running control-plane.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	binary := flag.String("binary", "", "path to pluribus-mcp binary")
	baseURL := flag.String("url", "http://127.0.0.1:8123", "control-plane base URL")
	apiKey := flag.String("api-key", "", "optional CONTROL_PLANE_API_KEY")
	flag.Parse()
	if strings.TrimSpace(*binary) == "" {
		fmt.Fprintln(os.Stderr, "missing --binary")
		os.Exit(2)
	}

	pass, fail := 0, 0
	ok := func(label string) {
		pass++
		fmt.Printf("PASS: %s\n", label)
	}
	bad := func(label, detail string) {
		fail++
		fmt.Fprintf(os.Stderr, "FAIL: %s — %s\n", label, detail)
	}

	cmd := exec.Command(*binary)
	cmd.Env = append(os.Environ(),
		"CONTROL_PLANE_URL="+strings.TrimRight(*baseURL, "/"),
	)
	if k := strings.TrimSpace(*apiKey); k != "" {
		cmd.Env = append(cmd.Env, "CONTROL_PLANE_API_KEY="+k)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		bad("start stdio process", err.Error())
		summary(pass, fail)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		bad("stdout pipe", err.Error())
		summary(pass, fail)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		bad("pluribus-mcp start", err.Error())
		summary(pass, fail)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	reader := bufio.NewReader(stdout)
	call := func(id int, method string, params any) map[string]any {
		body := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
		if params != nil {
			body["params"] = params
		}
		raw, _ := json.Marshal(body)
		if _, err := io.WriteString(stdin, string(raw)+"\n"); err != nil {
			bad(method, "write stdin: "+err.Error())
			return nil
		}
		line, err := readLineWithTimeout(reader, 30*time.Second)
		if err != nil {
			bad(method, "read stdout: "+err.Error())
			return nil
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(line), &out); err != nil {
			bad(method, "parse JSON: "+err.Error())
			return nil
		}
		return out
	}

	init := call(1, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "mcp-stdio-proof", "version": "0.1.0"},
	})
	if init == nil {
		summary(pass, fail)
	}
	if init["jsonrpc"] != "2.0" || fmt.Sprint(init["id"]) != "1" {
		bad("initialize shape", fmt.Sprintf("%v", init))
	} else if init["error"] != nil {
		bad("initialize", fmt.Sprintf("%v", init["error"]))
	} else if init["result"] == nil {
		bad("initialize", "missing result")
	} else {
		ok("stdio initialize")
	}

	list := call(2, "tools/list", map[string]any{})
	if list == nil {
		summary(pass, fail)
	}
	if list["error"] != nil {
		bad("tools/list", fmt.Sprintf("%v", list["error"]))
	} else {
		res, _ := list["result"].(map[string]any)
		tools, _ := res["tools"].([]any)
		if len(tools) < 30 {
			bad("tools/list count", fmt.Sprintf("got %d", len(tools)))
		} else {
			ok("stdio tools/list")
		}
		foundRecall := false
		for _, raw := range tools {
			tool, _ := raw.(map[string]any)
			if tool["name"] == "recall_context" {
				foundRecall = true
				if tool["inputSchema"] == nil {
					bad("recall_context schema", "missing inputSchema")
				}
			}
		}
		if foundRecall {
			ok("stdio tools/list includes recall_context schema")
		} else {
			bad("tools/list", "recall_context missing")
		}
	}

	recall := call(3, "tools/call", map[string]any{
		"name": "recall_context",
		"arguments": map[string]any{
			"task": "stdio adapter proof recall for phase 1 close-out",
		},
	})
	if recall == nil {
		summary(pass, fail)
	}
	if recall["error"] != nil {
		bad("tools/call recall_context", fmt.Sprintf("%v", recall["error"]))
	} else {
		ok("stdio tools/call recall_context")
	}

	unknown := call(4, "tools/call", map[string]any{"name": "does_not_exist", "arguments": map[string]any{}})
	if unknown == nil {
		summary(pass, fail)
	}
	if unknown["error"] == nil {
		bad("unknown tool", "expected JSON-RPC error")
	} else {
		errObj, _ := unknown["error"].(map[string]any)
		code, _ := errObj["code"].(float64)
		if code != -32602 {
			bad("unknown tool code", fmt.Sprintf("got %v", code))
		} else {
			ok("stdio unknown tool returns -32602")
		}
	}

	missing := call(5, "tools/call", map[string]any{"name": "recall_context", "arguments": map[string]any{}})
	if missing == nil {
		summary(pass, fail)
	}
	if missing["error"] == nil {
		bad("missing argument", "expected JSON-RPC error")
	} else {
		errObj, _ := missing["error"].(map[string]any)
		code, _ := errObj["code"].(float64)
		if code != -32602 {
			bad("missing argument code", fmt.Sprintf("got %v", code))
		} else {
			ok("stdio missing argument returns -32602")
		}
	}

	unknownMethod := call(6, "nope/method", map[string]any{})
	if unknownMethod == nil {
		summary(pass, fail)
	}
	if unknownMethod["error"] == nil {
		bad("unknown method", "expected error")
	} else {
		ok("stdio unknown method error")
	}

	summary(pass, fail)
}

func readLineWithTimeout(r *bufio.Reader, timeout time.Duration) (string, error) {
	type res struct {
		line string
		err  error
	}
	ch := make(chan res, 1)
	go func() {
		line, err := r.ReadString('\n')
		ch <- res{strings.TrimSpace(line), err}
	}()
	select {
	case out := <-ch:
		return out.line, out.err
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout after %s", timeout)
	}
}

func summary(pass, fail int) {
	fmt.Printf("\n======== SUMMARY ========\nPASS: %d  FAIL: %d\n", pass, fail)
	if fail > 0 {
		fmt.Println("RESULT: FAIL")
		os.Exit(1)
	}
	fmt.Println("RESULT: PASS")
	os.Exit(0)
}
