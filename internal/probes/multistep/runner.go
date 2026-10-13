package multistep

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/opsway-io/backend/internal/probes/http/asserter"
)

const DefaultStepTimeout = 10 * time.Second

// Prober performs a single HTTP request (implemented by http.Service).
type Prober interface {
	Probe(ctx context.Context, method, url string, headers map[string]string, body io.Reader, timeout time.Duration) (*http.Result, error)
}

// StepResult is the outcome of one executed step.
type StepResult struct {
	Index  int
	Step   *entities.MonitorStep
	Method string
	URL    string // URL after variable substitution

	Result *http.Result
	// Err is set when the request could not be performed (or auth failed).
	Err error

	Passed []entities.MonitorAssertion
	Failed []entities.MonitorAssertion

	// TLSErrors lists violated monitor TLS settings (hostname / expiration).
	TLSErrors []string
	// VariableErrors lists variables that could not be extracted.
	VariableErrors []string
}

// OK reports whether the step succeeded.
func (r *StepResult) OK() bool {
	return r.Err == nil && len(r.Failed) == 0 && len(r.TLSErrors) == 0
}

// RunResult is the outcome of running all steps of a monitor.
type RunResult struct {
	Steps []StepResult
	// FailedStep is the index of the first failing step, or -1.
	FailedStep int
	Variables  map[string]string
}

func (r *RunResult) OK() bool {
	return r.FailedStep < 0
}

// FailedAssertions returns all failed assertions of the run.
func (r *RunResult) FailedAssertions() []entities.MonitorAssertion {
	var out []entities.MonitorAssertion
	for _, s := range r.Steps {
		out = append(out, s.Failed...)
	}

	return out
}

type Runner struct {
	prober   Prober
	asserter *asserter.HTTPResultAsserter
	tokens   *TokenSource
	timeout  time.Duration
	now      func() time.Time
}

func NewRunner(prober Prober, tokens *TokenSource) *Runner {
	if tokens == nil {
		tokens = NewTokenSource(nil)
	}

	return &Runner{
		prober:   prober,
		asserter: asserter.New(),
		tokens:   tokens,
		timeout:  DefaultStepTimeout,
		now:      time.Now,
	}
}

// Run executes the monitor's steps in order, stopping at the first failing step.
func (r *Runner) Run(ctx context.Context, m *entities.Monitor) *RunResult {
	run := &RunResult{FailedStep: -1, Variables: map[string]string{}}

	steps := Steps(m)

	authHeader, authErr := r.authorization(ctx, m.Settings.Auth)

	for i := range steps {
		step := &steps[i]
		sr := r.runStep(ctx, m, i, step, run.Variables, authHeader, authErr)
		run.Steps = append(run.Steps, sr)

		if !sr.OK() {
			run.FailedStep = i

			break
		}
	}

	return run
}

// Steps returns the monitor's steps, falling back to the legacy single request
// settings for monitors scheduled before steps existed.
func Steps(m *entities.Monitor) []entities.MonitorStep {
	if len(m.Steps) > 0 {
		return m.Steps
	}

	if m.Settings.URL == "" {
		return nil
	}

	var legacy []entities.MonitorAssertion
	for _, a := range m.Assertions {
		if a.StepID == nil {
			legacy = append(legacy, a)
		}
	}

	return []entities.MonitorStep{{
		MonitorID:  m.ID,
		Name:       "Step 1",
		Method:     m.Settings.Method,
		URL:        m.Settings.URL,
		Headers:    m.Settings.Headers,
		Body:       m.Settings.Body,
		Assertions: legacy,
	}}
}

func (r *Runner) authorization(ctx context.Context, auth entities.MonitorSettingsAuth) (string, error) {
	switch auth.Method {
	case entities.MonitorAuthMethodBasic:
		return BasicAuthHeader(auth.Username, auth.Password), nil
	case entities.MonitorAuthMethodOAuth2ClientCredentials:
		token, err := r.tokens.Token(ctx, auth)
		if err != nil {
			return "", err
		}

		return "Bearer " + token, nil
	default:
		return "", nil
	}
}

func (r *Runner) runStep(
	ctx context.Context,
	m *entities.Monitor,
	index int,
	step *entities.MonitorStep,
	vars map[string]string,
	authHeader string,
	authErr error,
) StepResult {
	sr := StepResult{
		Index:  index,
		Step:   step,
		Method: step.Method,
		URL:    Substitute(step.URL, vars),
	}

	if sr.Method == "" {
		sr.Method = "GET"
	}

	if authErr != nil {
		sr.Err = fmt.Errorf("authentication failed: %w", authErr)
		sr.Failed = step.Assertions

		return sr
	}

	headers := BuildHeaders(step, vars, authHeader)

	var body io.Reader
	if content := step.Body.GetContentString(); content != nil && step.Body.Type != "NONE" && step.Body.Type != "" {
		body = strings.NewReader(Substitute(*content, vars))
	}

	res, err := r.prober.Probe(ctx, sr.Method, sr.URL, headers, body, r.timeout)
	if err != nil {
		sr.Err = err
		// A request that could not be performed fails all of its assertions
		sr.Failed = step.Assertions

		return sr
	}

	sr.Result = res

	sr.Passed, sr.Failed, err = r.assert(res, step.Assertions)
	if err != nil {
		sr.Err = err

		return sr
	}

	sr.TLSErrors = CheckTLS(m.Settings.TLS, res, r.now())

	for _, v := range step.Variables {
		value, err := ExtractVariable(res, v)
		if err != nil {
			sr.VariableErrors = append(sr.VariableErrors, fmt.Sprintf("%s: %v", v.Name, err))

			continue
		}

		vars[v.Name] = value
	}

	return sr
}

// BuildHeaders renders a step's headers (with variables substituted), adds a
// content type matching the body type and the auth header unless the step
// sets them explicitly.
func BuildHeaders(step *entities.MonitorStep, vars map[string]string, authHeader string) map[string]string {
	headers := map[string]string{}
	has := map[string]bool{}

	for _, h := range step.Headers {
		if strings.TrimSpace(h.Key) == "" {
			continue
		}

		key := Substitute(h.Key, vars)
		headers[key] = Substitute(h.Value, vars)
		has[strings.ToLower(key)] = true
	}

	if authHeader != "" && !has["authorization"] {
		headers["Authorization"] = authHeader
	}

	if !has["content-type"] {
		if ct := contentTypeForBody(step.Body.Type); ct != "" && step.Body.Content != nil {
			headers["Content-Type"] = ct
		}
	}

	return headers
}

func contentTypeForBody(bodyType string) string {
	switch bodyType {
	case "JSON", "GRAPHQL":
		return "application/json"
	case "XML":
		return "application/xml"
	case "RAW":
		return "text/plain"
	default:
		return ""
	}
}

func (r *Runner) assert(res *http.Result, assertions []entities.MonitorAssertion) (passed, failed []entities.MonitorAssertion, err error) {
	if len(assertions) == 0 {
		return nil, nil, nil
	}

	rules := make([]asserter.Rule, len(assertions))
	for i, a := range assertions {
		rules[i] = asserter.Rule{
			Source:   a.Source,
			Property: a.Property,
			Operator: a.Operator,
			Target:   a.Target,
		}
	}

	oks, err := r.asserter.Assert(res, rules)
	if err != nil {
		return nil, nil, err
	}

	for i, ok := range oks {
		if ok {
			passed = append(passed, assertions[i])
		} else {
			failed = append(failed, assertions[i])
		}
	}

	return passed, failed, nil
}

// CheckTLS validates the response certificate against the monitor TLS settings.
func CheckTLS(settings entities.MonitorSettingsTLS, res *http.Result, now time.Time) []string {
	if !settings.Enabled || res == nil || res.TLS == nil {
		return nil
	}

	var errs []string

	cert := res.TLS.Certificate

	if settings.VerifyHostname != nil && *settings.VerifyHostname && !cert.HostValid {
		errs = append(errs, "certificate is not valid for the hostname")
	}

	if settings.CheckExpiration != nil && *settings.CheckExpiration {
		threshold := time.Duration(0)
		if settings.ExpirationThresholdDays != nil {
			threshold = time.Duration(*settings.ExpirationThresholdDays) * 24 * time.Hour
		}

		if now.Add(threshold).After(cert.NotAfter) {
			errs = append(errs, fmt.Sprintf("certificate expires at %s", cert.NotAfter.Format(time.RFC3339)))
		}
	}

	return errs
}
