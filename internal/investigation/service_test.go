package investigation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSynthesisResult_FencedJSON(t *testing.T) {
	raw := "```json\n" + `{
		"rootCauseSummary": "Upstream payment gateway returned HTTP 504 timeouts.",
		"detailedDiagnosis": "### Findings\n- Probe latency spiked to 10s",
		"confidenceScore": 0.88,
		"remediationActions": [
			{
				"title": "Restart payments worker",
				"description": "Clear wedged HTTP keep-alive pool",
				"actionType": "WEBHOOK_CALL",
				"targetService": "payments-api",
				"diffOrCommand": "POST https://hooks.internal/restart",
				"payloadJson": "{\"target_url\":\"https://hooks.internal/restart\"}"
			}
		]
	}` + "\n```"

	res := parseSynthesisResult(raw, "payments-api")
	assert.Equal(t, "Upstream payment gateway returned HTTP 504 timeouts.", res.RootCauseSummary)
	assert.InDelta(t, 0.88, res.ConfidenceScore, 0.001)
	require.Len(t, res.RemediationActions, 1)
	assert.Equal(t, "WEBHOOK_CALL", res.RemediationActions[0].ActionType)
}

func TestParseSynthesisResult_PlainTextFallback(t *testing.T) {
	res := parseSynthesisResult("DNS resolution failed for api.example.com across 2 regions.", "edge-proxy")
	assert.Contains(t, res.RootCauseSummary, "DNS resolution failed")
	assert.Equal(t, 0.65, res.ConfidenceScore)
}

func TestExtractCorrelatedLogs_RealLogSearchPayload(t *testing.T) {
	svc := &serviceImpl{logSearchPublicURL: "https://logs.opsway.eu"}
	payload := `{
		"logs": [
			{
				"id": "log-101",
				"timestamp": "2026-11-23T13:00:00Z",
				"service_name": "billing-worker",
				"log_level": "error",
				"message": "context deadline exceeded calling stripe.com",
				"labels": {"pod": "billing-worker-abc"}
			}
		]
	}`

	entries := svc.extractCorrelatedLogs(payload, "billing-worker")
	require.Len(t, entries, 1)
	assert.Equal(t, "log-101", entries[0].LogID)
	assert.Equal(t, "ERROR", entries[0].LogLevel)
	assert.Equal(t, "billing-worker", entries[0].ServiceName)
	assert.Contains(t, entries[0].Message, "context deadline exceeded")
	assert.Contains(t, entries[0].DeepLinkURL, "https://logs.opsway.eu/?query=")
}

func TestRunRemediationAction_ExecutesSignedWebhook(t *testing.T) {
	var receivedSig string
	var receivedBody map[string]interface{}

	webhookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get("X-Opsway-Signature-256")
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer webhookSrv.Close()

	secret := "top-secret-hmac"
	svc := &serviceImpl{httpClient: webhookSrv.Client()}
	incID := uint(99)
	inv := &entities.IncidentInvestigation{
		UUID:        "inv_test123",
		TeamID:      7,
		IncidentID:  &incID,
		ServiceName: "auth-service",
	}
	cfg := &entities.AgentConfig{
		ByoaWebhookSecret: &secret,
		RemediationPlaybooks: []entities.RemediationPlaybookConfig{
			{
				ID:        "pb_restart",
				Name:      "Restart Auth Service",
				TargetURL: webhookSrv.URL,
				Method:    "POST",
			},
		},
	}
	action := &entities.RemediationAction{
		Title:       "Restart Auth Service",
		ActionType:  "WEBHOOK_CALL",
		PayloadJSON: `{"playbook_id":"pb_restart"}`,
	}

	summary, err := svc.runRemediationAction(context.Background(), inv, action, cfg)
	require.NoError(t, err)
	assert.Contains(t, summary, "HTTP 202")
	assert.True(t, len(receivedSig) > 10 && receivedSig[:7] == "sha256=")
	assert.Equal(t, "auth-service", receivedBody["service"])
}

func TestExternalMCPDiscoveryAndCall(t *testing.T) {
	extSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rpcReq mcp.JSONRPCRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&rpcReq))
		w.Header().Set("Content-Type", "application/json")
		if rpcReq.Method == "tools/list" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      rpcReq.ID,
				"result": map[string]interface{}{
					"tools": []mcp.Tool{
						{Name: "get_pod_events", Description: "List K8s pod events"},
					},
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      rpcReq.ID,
			"result": mcp.ToolCallResult{
				Content: []mcp.ContentItem{{Type: "text", Text: "OOMKilled on container api"}},
			},
		})
	}))
	defer extSrv.Close()

	svc := &serviceImpl{httpClient: extSrv.Client()}
	cfg := &entities.AgentConfig{
		ExternalMCPServers: []entities.ExternalMCPServerConfig{
			{Name: "k8s-ops", URL: extSrv.URL, Enabled: true},
		},
	}
	defs, routes := svc.buildAgentTools(context.Background(), 1, cfg)
	require.Len(t, defs, 1)
	assert.Equal(t, "ext.k8s_ops.get_pod_events", defs[0].Function.Name)

	res, err := svc.invokeAgentTool(context.Background(), 1, "ext.k8s_ops.get_pod_events", map[string]interface{}{"ns": "prod"}, routes)
	require.NoError(t, err)
	assert.Equal(t, "OOMKilled on container api", extractFirstText(res))
}
