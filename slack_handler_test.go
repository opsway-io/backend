package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/rest/controllers/webhooks"
)

func TestPostSlackInteractiveJSON(t *testing.T) {
	e := echo.New()
	
	// Construct the payload as JSON
	payload := `{"type":"block_actions","actions":[{"action_id":"acknowledge_incident","value":"ack_123"}],"response_url":"http://localhost:8001/test"}`
	
	req := httptest.NewRequest(http.MethodPost, "/webhooks/slack/interactive", bytes.NewBufferString(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	
	h := &webhooks.Handlers{}
	err := h.PostSlackInteractive(c)
	if err != nil {
		t.Logf("Error: %v", err)
	}
	t.Logf("Response code: %d, body: %s", rec.Code, rec.Body.String())
}
