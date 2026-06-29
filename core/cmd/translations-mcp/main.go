// Command translations-mcp is a Model Context Protocol (MCP) server that lets any
// AI tool summarize, query, and work on the Flarewifi per-language JSON translation
// catalogs (resources/translations/<lang>.json).
//
// Transport: newline-delimited JSON-RPC 2.0 over stdio (the MCP stdio transport),
// implemented with the standard library only — the repo pins its module graph and
// forbids `go mod tidy`, so no MCP SDK is pulled in.
//
// Run as an MCP server (default):
//
//	go run ./core/cmd/translations-mcp                 # speaks JSON-RPC on stdio
//
// Run the CI gate (no server):
//
//	go run ./core/cmd/translations-mcp -check          # fail if code keys missing from en.json
//	go run ./core/cmd/translations-mcp -check -min 80  # also require >=80% coverage per language
//
// Register with an MCP client (e.g. Claude) by pointing it at the built binary with
// no arguments; `-root` selects the repo working directory to scan (default ".").
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

const serverVersion = "1.0.0"

// translateCall matches api.Translate("<type>", "<text>"...) in .go and .templ
// source — the same shape the catalogs are keyed by. Only the leading two string
// literals are captured; params after them are interpolation values, not keys.
var translateCall = regexp.MustCompile(`\.?Translate\(\s*"([^"]+?)"\s*,\s*"([^"]+?)"`)

func main() {
	var root string
	var check bool
	var minCoverage float64
	flag.StringVar(&root, "root", ".", "Repo working directory to scan for components")
	flag.BoolVar(&check, "check", false, "Run the CI gate (report missing code keys / coverage) and exit")
	flag.Float64Var(&minCoverage, "min", 0, "Minimum per-language coverage percent required by -check")
	flag.Parse()

	srv := &server{root: root}
	if check {
		os.Exit(srv.runCheck(minCoverage))
	}
	srv.serve(os.Stdin, os.Stdout)
}

// =============================================================================
// JSON-RPC plumbing
// =============================================================================

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type server struct {
	root string
	out  *json.Encoder
}

func (s *server) serve(in io.Reader, out io.Writer) {
	s.out = json.NewEncoder(out)
	reader := bufio.NewReader(in)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			s.handleLine(line)
		}
		if err != nil {
			return // EOF or read error ends the session
		}
	}
}

func (s *server) handleLine(line []byte) {
	if len(strings.TrimSpace(string(line))) == 0 {
		return
	}
	var req rpcRequest
	if err := json.Unmarshal(line, &req); err != nil {
		return // malformed; cannot reply without an id
	}

	// Notifications (no id) get no response.
	notification := len(req.ID) == 0

	result, rerr := s.dispatch(req)
	if notification {
		return
	}
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	if rerr != nil {
		resp.Error = rerr
	} else {
		resp.Result = result
	}
	s.out.Encode(resp)
}

func (s *server) dispatch(req rpcRequest) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "flarewifi-translations", "version": serverVersion},
		}, nil
	case "notifications/initialized":
		return nil, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": toolDefs()}, nil
	case "tools/call":
		return s.callTool(req.Params)
	default:
		return nil, &rpcError{Code: -32601, Message: "method not found: " + req.Method}
	}
}

// callTool runs a tool and wraps its result as MCP text content. Tool errors are
// reported as isError content (not JSON-RPC errors) so the model sees the message.
func (s *server) callTool(params json.RawMessage) (any, *rpcError) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: -32602, Message: "invalid params"}
	}

	data, err := s.runTool(p.Name, p.Arguments)
	if err != nil {
		return map[string]any{
			"content": []map[string]any{{"type": "text", "text": "Error: " + err.Error()}},
			"isError": true,
		}, nil
	}
	pretty, _ := json.MarshalIndent(data, "", "  ")
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": string(pretty)}},
	}, nil
}

func (s *server) runTool(name string, args json.RawMessage) (any, error) {
	switch name {
	case "list_components":
		return s.toolListComponents()
	case "summarize":
		return s.toolSummarize(args)
	case "list_keys":
		return s.toolListKeys(args)
	case "get_translation":
		return s.toolGetTranslation(args)
	case "find_untranslated":
		return s.toolFindUntranslated(args)
	case "set_translation":
		return s.toolSetTranslation(args)
	case "set_translations":
		return s.toolSetTranslations(args)
	case "sync":
		return s.toolSync(args)
	case "check":
		return s.toolCheck(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}
