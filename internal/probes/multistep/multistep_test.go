package multistep

import (
	"context"
	"errors"
	"fmt"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubstitute(t *testing.T) {
	vars := map[string]string{"TOKEN": "abc", "user.id": "42"}

	assert.Equal(t, "https://x/users/42?t=abc", Substitute("https://x/users/{{user.id}}?t={{ TOKEN }}", vars))
	assert.Equal(t, "keep {{UNKNOWN}}", Substitute("keep {{UNKNOWN}}", vars))
	assert.Equal(t, "no vars", Substitute("no vars", nil))
}

func TestValidateVariable(t *testing.T) {
	assert.NoError(t, ValidateVariable(entities.MonitorStepVariable{Name: "ID", Source: "JSON_BODY", Property: "$.id"}))
	assert.NoError(t, ValidateVariable(entities.MonitorStepVariable{Name: "REQ", Source: "HEADER", Property: "X-Request-ID"}))
	assert.Error(t, ValidateVariable(entities.MonitorStepVariable{Name: "1bad", Source: "JSON_BODY", Property: "$.id"}))
	assert.Error(t, ValidateVariable(entities.MonitorStepVariable{Name: "X", Source: "NOPE", Property: "a"}))
	assert.Error(t, ValidateVariable(entities.MonitorStepVariable{Name: "X", Source: "HEADER"}))
}

func TestExtractVariable(t *testing.T) {
	res := &http.Result{Response: http.Response{
		StatusCode: 201,
		Header:     nethttp.Header{"X-Request-Id": []string{"r-1"}},
		Body:       []byte(`{"data":{"id":7,"name":"n"},"items":[1,2]}`),
	}}

	v, err := ExtractVariable(res, entities.MonitorStepVariable{Name: "A", Source: "JSON_BODY", Property: "data.id"})
	require.NoError(t, err)
	assert.Equal(t, "7", v)

	v, err = ExtractVariable(res, entities.MonitorStepVariable{Name: "A", Source: "JSON_BODY", Property: "$.items"})
	require.NoError(t, err)
	assert.Equal(t, "[1,2]", v)

	v, err = ExtractVariable(res, entities.MonitorStepVariable{Name: "A", Source: "HEADER", Property: "x-request-id"})
	require.NoError(t, err)
	assert.Equal(t, "r-1", v)

	_, err = ExtractVariable(res, entities.MonitorStepVariable{Name: "A", Source: "JSON_BODY", Property: "$.missing"})
	assert.Error(t, err)
}

type probeCall struct {
	method, url string
	headers     map[string]string
	body        string
}

type fakeProber struct {
	calls     []probeCall
	responses []*http.Result
	errs      []error
}

func (f *fakeProber) Probe(_ context.Context, method, url string, headers map[string]string, body io.Reader, _ time.Duration) (*http.Result, error) {
	c := probeCall{method: method, url: url, headers: headers}
	if body != nil {
		b, _ := io.ReadAll(body)
		c.body = string(b)
	}

	i := len(f.calls)
	f.calls = append(f.calls, c)

	if i < len(f.errs) && f.errs[i] != nil {
		return nil, f.errs[i]
	}

	return f.responses[i], nil
}

func jsonResult(status int, body string) *http.Result {
	return &http.Result{
		Response: http.Response{StatusCode: status, Header: nethttp.Header{}, Body: []byte(body)},
		Timing:   http.Timing{Phases: http.TimingPhases{Total: 10 * time.Millisecond}},
	}
}

func twoStepMonitor() *entities.Monitor {
	body := []byte(`{"owner":"{{ID}}"}`)

	return &entities.Monitor{
		ID: 1,
		Settings: entities.MonitorSettings{
			Auth: entities.MonitorSettingsAuth{Method: "BASIC", Username: "u", Password: "p"},
		},
		Steps: []entities.MonitorStep{
			{
				Name: "login", Method: "POST", URL: "https://api.test/login",
				Assertions: []entities.MonitorAssertion{{ID: 1, Source: "STATUS_CODE", Operator: "EQUAL", Target: "200"}},
				Variables:  []entities.MonitorStepVariable{{Name: "ID", Source: "JSON_BODY", Property: "$.id"}},
			},
			{
				Name: "get", Method: "PUT", URL: "https://api.test/things/{{ID}}",
				Headers:    []entities.MonitorSettingsHeader{{Key: "X-Id", Value: "{{ID}}"}},
				Body:       entities.MonitorSettingsBody{Type: "JSON", Content: &body},
				Assertions: []entities.MonitorAssertion{{ID: 2, Source: "JSON_BODY", Property: "$.ok", Operator: "EQUAL", Target: "true"}},
			},
		},
	}
}

func TestRunnerExecutesStepsWithVariables(t *testing.T) {
	p := &fakeProber{responses: []*http.Result{
		jsonResult(200, `{"id":"abc"}`),
		jsonResult(200, `{"ok":true}`),
	}}

	run := NewRunner(p, nil).Run(context.Background(), twoStepMonitor())

	require.True(t, run.OK())
	require.Len(t, p.calls, 2)
	assert.Equal(t, "https://api.test/things/abc", p.calls[1].url)
	assert.Equal(t, "PUT", p.calls[1].method)
	assert.Equal(t, "abc", p.calls[1].headers["X-Id"])
	assert.Equal(t, `{"owner":"abc"}`, p.calls[1].body)
	assert.Equal(t, "application/json", p.calls[1].headers["Content-Type"])
	assert.Equal(t, BasicAuthHeader("u", "p"), p.calls[0].headers["Authorization"])
}

func TestRunnerStopsAtFirstFailingStep(t *testing.T) {
	p := &fakeProber{responses: []*http.Result{jsonResult(500, `{}`), jsonResult(200, `{"ok":true}`)}}

	run := NewRunner(p, nil).Run(context.Background(), twoStepMonitor())

	assert.False(t, run.OK())
	assert.Equal(t, 0, run.FailedStep)
	assert.Len(t, p.calls, 1)
	require.Len(t, run.FailedAssertions(), 1)
	assert.Equal(t, uint(1), run.FailedAssertions()[0].ID)
}

func TestRunnerRequestErrorFailsAssertions(t *testing.T) {
	p := &fakeProber{errs: []error{errors.New("dial tcp: refused")}, responses: []*http.Result{nil}}

	run := NewRunner(p, nil).Run(context.Background(), twoStepMonitor())

	assert.Equal(t, 0, run.FailedStep)
	assert.Error(t, run.Steps[0].Err)
	assert.Len(t, run.FailedAssertions(), 1)
}

func TestStepsLegacyFallback(t *testing.T) {
	stepID := uint(3)
	m := &entities.Monitor{
		Settings:   entities.MonitorSettings{Method: "GET", URL: "https://legacy.test"},
		Assertions: []entities.MonitorAssertion{{ID: 1}, {ID: 2, StepID: &stepID}},
	}

	steps := Steps(m)
	require.Len(t, steps, 1)
	assert.Equal(t, "https://legacy.test", steps[0].URL)
	assert.Len(t, steps[0].Assertions, 1)
}

func TestCheckTLS(t *testing.T) {
	yes := true
	days := uint(30)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	res := &http.Result{TLS: &http.TLS{Certificate: http.Certificate{
		NotAfter:  now.Add(10 * 24 * time.Hour),
		HostValid: false,
	}}}

	errs := CheckTLS(entities.MonitorSettingsTLS{Enabled: true, VerifyHostname: &yes, CheckExpiration: &yes, ExpirationThresholdDays: &days}, res, now)
	assert.Len(t, errs, 2)

	assert.Empty(t, CheckTLS(entities.MonitorSettingsTLS{Enabled: false, VerifyHostname: &yes}, res, now))
}

func TestTokenSourceCachesToken(t *testing.T) {
	var hits int32

	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		atomic.AddInt32(&hits, 1)
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "client_credentials" || r.Form.Get("client_id") != "id" {
			w.WriteHeader(nethttp.StatusBadRequest)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":"tok-%d","expires_in":3600}`, atomic.LoadInt32(&hits))
	}))
	defer srv.Close()

	ts := NewTokenSource(srv.Client())
	now := time.Now()
	ts.now = func() time.Time { return now }

	auth := entities.MonitorSettingsAuth{Method: "OAUTH2_CLIENT_CREDENTIALS", TokenURL: srv.URL, ClientID: "id", ClientSecret: "s"}

	tok, err := ts.Token(context.Background(), auth)
	require.NoError(t, err)
	assert.Equal(t, "tok-1", tok)

	tok, err = ts.Token(context.Background(), auth)
	require.NoError(t, err)
	assert.Equal(t, "tok-1", tok)
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits))

	// After expiry a new token is fetched
	now = now.Add(2 * time.Hour)
	tok, err = ts.Token(context.Background(), auth)
	require.NoError(t, err)
	assert.Equal(t, "tok-2", tok)
}
