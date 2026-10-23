package mcp

import (
	"context"
	"testing"
)

func TestGatewayJSONRPC(t *testing.T) {
	gw := NewGateway(
		GatewayConfig{
			LogSearchPublicURL: "https://logs.opsway.eu",
		},
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	// Test Initialize
	initResp := gw.HandleJSONRPC(context.Background(), 1, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	})
	if initResp == nil || initResp.Result == nil {
		t.Fatalf("expected initialize result, got nil")
	}

	// Test Ping
	pingResp := gw.HandleJSONRPC(context.Background(), 1, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "ping",
	})
	if pingResp == nil || pingResp.Error != nil {
		t.Fatalf("expected ping success, got %v", pingResp)
	}

	// Test Tools List
	tools := gw.ListTools(context.Background(), 1)
	if len(tools) == 0 {
		t.Fatalf("expected non-empty tool list for default team")
	}

	hasMonitoring := false
	hasLogs := false
	for _, tool := range tools {
		if tool.Name == "monitoring.get_incident_details" {
			hasMonitoring = true
		}
		if tool.Name == "logs.get_error_summary" {
			hasLogs = true
		}
	}
	if !hasMonitoring || !hasLogs {
		t.Errorf("expected both monitoring and log tools, got hasMonitoring=%v, hasLogs=%v", hasMonitoring, hasLogs)
	}
}
