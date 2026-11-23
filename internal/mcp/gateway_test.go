package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallPantomathMCP_StreamableHTTPSSE(t *testing.T) {
	var receivedAccept string
	var receivedParams ToolCallParams

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			http.NotFound(w, r)
			return
		}
		receivedAccept = r.Header.Get("Accept")

		var req JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if err := json.Unmarshal(req.Params, &receivedParams); err != nil {
			t.Fatalf("failed to decode params: %v", err)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"content\":[{\"type\":\"text\",\"text\":\"Live GraphRAG answer from Pantomath\"}]}}\n\n"))
	}))
	defer srv.Close()

	gw := NewGateway(GatewayConfig{
		PantomathURL:     srv.URL,
		PantomathEnabled: true,
	}, nil, nil, nil, nil, nil)

	res, err := gw.CallTool(context.Background(), 1, "pantomath.ask_codebase", map[string]interface{}{
		"query":   "How is postgres configured?",
		"product": "status-monitoring",
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if !strings.Contains(receivedAccept, "text/event-stream") {
		t.Fatalf("expected Accept header to include text/event-stream, got %q", receivedAccept)
	}
	if receivedParams.Name != "ask_pantomath" {
		t.Fatalf("expected upstream tool ask_pantomath, got %q", receivedParams.Name)
	}
	if got := receivedParams.Arguments["product"]; got != "status_monitoring" {
		t.Fatalf("expected hyphen-normalized product status_monitoring, got %v", got)
	}
	if res == nil || len(res.Content) == 0 || res.Content[0].Text != "Live GraphRAG answer from Pantomath" {
		t.Fatalf("unexpected tool result: %+v", res)
	}
}

func TestParseMCPToolResponse_PlainJSON(t *testing.T) {
	raw := []byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"ok"}]}}`)
	res, err := parseMCPToolResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Content) != 1 || res.Content[0].Text != "ok" {
		t.Fatalf("unexpected result: %+v", res)
	}
}
