package monitors

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/probes/http/asserter"
	"github.com/opsway-io/backend/internal/probes/multistep"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

const (
	maxStepsPerMonitor      = 25
	maxAssertionsPerStep    = 50
	maxVariablesPerStep     = 25
	maxHeadersPerStep       = 50
	maxBodyContentBytes     = 1 << 20 // 1 MB
	maxMonitorsPerBulk      = 100
	maxURLLength            = 2048
	maxHeaderLength         = 4096
	maxAssertionFieldLength = 1024
)

/*
	Shared structs (the JSON contract of the dashboard)
*/

type Monitor struct {
	ID        uint            `json:"id"`
	State     string          `json:"state"`
	Name      string          `json:"name"`
	Settings  MonitorSettings `json:"settings"`
	Steps     []MonitorStep   `json:"steps"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type MonitorSettings struct {
	FrequencySeconds uint64              `json:"frequencySeconds" validate:"required,monitorFrequency"`
	TLS              MonitorSettingsTLS  `json:"tls"`
	Auth             MonitorSettingsAuth `json:"auth"`
	Locations        []string            `json:"locations" validate:"omitempty,max=50,dive,required,max=255"`
}

type MonitorSettingsTLS struct {
	Enabled                 bool  `json:"enabled"`
	VerifyHostname          *bool `json:"verifyHostname"`
	CheckExpiration         *bool `json:"checkExpiration"`
	ExpirationThresholdDays *uint `json:"expirationThresholdDays" validate:"omitempty,max=3650"`
}

type MonitorSettingsAuth struct {
	Method       string `json:"method"`
	TokenURL     string `json:"tokenUrl,omitempty" validate:"max=2048"`
	ClientID     string `json:"clientId,omitempty" validate:"max=1024"`
	ClientSecret string `json:"clientSecret,omitempty" validate:"max=4096"`
	Username     string `json:"username,omitempty" validate:"max=1024"`
	Password     string `json:"password,omitempty" validate:"max=4096"`
}

type MonitorStep struct {
	Name       string                  `json:"name" validate:"max=255"`
	Method     string                  `json:"method" validate:"required,monitorMethod"`
	URL        string                  `json:"url" validate:"required"`
	Headers    []MonitorSettingsHeader `json:"headers" validate:"omitempty,dive"`
	Body       MonitorSettingsBody     `json:"body"`
	Assertions []MonitorAssertion      `json:"assertions"`
	Variables  []MonitorVariable       `json:"variables"`
}

type MonitorSettingsHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type MonitorSettingsBody struct {
	Type    string  `json:"type"`
	Content *string `json:"content"`
}

type MonitorAssertion struct {
	Source   string `json:"source"`
	Property string `json:"property,omitempty"`
	Operator string `json:"operator"`
	Target   string `json:"target,omitempty"`
}

type MonitorVariable struct {
	Name     string `json:"name"`
	Source   string `json:"source"`
	Property string `json:"property"`
}

// ValidationError is returned for semantically invalid monitor input.
type ValidationError struct {
	msg string
}

func (e *ValidationError) Error() string {
	return e.msg
}

func invalid(format string, args ...interface{}) error {
	return &ValidationError{msg: fmt.Sprintf(format, args...)}
}

//nolint:gochecknoglobals
var assertionValidator = asserter.New()

/*
	Request -> entity
*/

// newMonitorEntity validates the request input and builds the monitor entity.
func newMonitorEntity(teamID uint, name string, settings MonitorSettings, steps []MonitorStep, allowedLocations []string) (*entities.Monitor, error) {
	if strings.TrimSpace(name) == "" {
		return nil, invalid("name is required")
	}

	if len(steps) == 0 {
		return nil, invalid("at least one step is required")
	}

	if len(steps) > maxStepsPerMonitor {
		return nil, invalid("a monitor can have at most %d steps", maxStepsPerMonitor)
	}

	ms, err := newSettingsEntity(settings, allowedLocations)
	if err != nil {
		return nil, err
	}

	m := &entities.Monitor{
		TeamID:   teamID,
		Name:     strings.TrimSpace(name),
		Settings: *ms,
		Steps:    make([]entities.MonitorStep, len(steps)),
	}

	for i, s := range steps {
		step, err := newStepEntity(i, s)
		if err != nil {
			return nil, err
		}

		m.Steps[i] = *step
	}

	return m, nil
}

func newSettingsEntity(s MonitorSettings, allowedLocations []string) (*entities.MonitorSettings, error) {
	ms := &entities.MonitorSettings{
		TLS: entities.MonitorSettingsTLS{
			Enabled:                 s.TLS.Enabled,
			VerifyHostname:          s.TLS.VerifyHostname,
			CheckExpiration:         s.TLS.CheckExpiration,
			ExpirationThresholdDays: s.TLS.ExpirationThresholdDays,
		},
	}

	if !containsUint(helpers.AllowedMonitorFrequencies, s.FrequencySeconds) {
		return nil, invalid("invalid frequencySeconds: %d", s.FrequencySeconds)
	}

	ms.SetFrequencySeconds(s.FrequencySeconds)

	auth, err := newAuthEntity(s.Auth)
	if err != nil {
		return nil, err
	}

	ms.Auth = *auth

	locations, err := normalizeRequestLocations(s.Locations, allowedLocations)
	if err != nil {
		return nil, err
	}

	ms.Locations = locations

	return ms, nil
}

func newAuthEntity(a MonitorSettingsAuth) (*entities.MonitorSettingsAuth, error) {
	method := a.Method
	if method == "" {
		method = entities.MonitorAuthMethodNone
	}

	auth := &entities.MonitorSettingsAuth{Method: method}

	switch method {
	case entities.MonitorAuthMethodNone:
		// Ignore credentials of other auth methods
	case entities.MonitorAuthMethodBasic:
		auth.Username = a.Username
		auth.Password = a.Password
	case entities.MonitorAuthMethodOAuth2ClientCredentials:
		if !isHTTPURL(a.TokenURL) {
			return nil, invalid("auth.tokenUrl must be an absolute http(s) url")
		}

		if a.ClientID == "" {
			return nil, invalid("auth.clientId is required")
		}

		auth.TokenURL = a.TokenURL
		auth.ClientID = a.ClientID
		auth.ClientSecret = a.ClientSecret
	default:
		return nil, invalid("invalid auth.method: %s", method)
	}

	return auth, nil
}

func normalizeRequestLocations(locations []string, allowed []string) ([]string, error) {
	locations = monitor.NormalizeLocations(locations)

	for _, l := range locations {
		if l == monitor.DefaultLocation {
			continue
		}

		if !containsString(allowed, l) {
			return nil, invalid("unknown location: %s", l)
		}
	}

	return locations, nil
}

func newStepEntity(index int, s MonitorStep) (*entities.MonitorStep, error) {
	prefix := fmt.Sprintf("steps[%d]", index)

	if !containsString(helpers.AllowedMonitorMethods, s.Method) {
		return nil, invalid("%s: invalid method: %s", prefix, s.Method)
	}

	if err := validateStepURL(s.URL); err != nil {
		return nil, invalid("%s: %s", prefix, err.Error())
	}

	if len(s.Headers) > maxHeadersPerStep {
		return nil, invalid("%s: at most %d headers are allowed", prefix, maxHeadersPerStep)
	}

	if len(s.Assertions) > maxAssertionsPerStep {
		return nil, invalid("%s: at most %d assertions are allowed", prefix, maxAssertionsPerStep)
	}

	if len(s.Variables) > maxVariablesPerStep {
		return nil, invalid("%s: at most %d variables are allowed", prefix, maxVariablesPerStep)
	}

	name := strings.TrimSpace(s.Name)
	if name == "" {
		name = fmt.Sprintf("Step %d", index+1)
	}

	step := &entities.MonitorStep{
		Position:   index,
		Name:       name,
		Method:     s.Method,
		URL:        strings.TrimSpace(s.URL),
		Headers:    make([]entities.MonitorSettingsHeader, 0, len(s.Headers)),
		Assertions: make([]entities.MonitorAssertion, len(s.Assertions)),
		Variables:  make([]entities.MonitorStepVariable, len(s.Variables)),
	}

	for j, h := range s.Headers {
		if strings.TrimSpace(h.Key) == "" {
			return nil, invalid("%s.headers[%d]: key is required", prefix, j)
		}

		if len(h.Key) > 255 || len(h.Value) > maxHeaderLength {
			return nil, invalid("%s.headers[%d]: header is too long", prefix, j)
		}

		step.Headers = append(step.Headers, entities.MonitorSettingsHeader{Key: h.Key, Value: h.Value})
	}

	bodyType := s.Body.Type
	if bodyType == "" {
		bodyType = "NONE"
	}

	if !containsString(helpers.AllowedBodyTypes, bodyType) {
		return nil, invalid("%s: invalid body type: %s", prefix, bodyType)
	}

	if s.Body.Content != nil && len(*s.Body.Content) > maxBodyContentBytes {
		return nil, invalid("%s: body is larger than 1 MB", prefix)
	}

	step.Body.Type = bodyType
	if bodyType != "NONE" {
		step.Body.SetContentString(s.Body.Content)
	}

	for j, a := range s.Assertions {
		ea, err := newAssertionEntity(a)
		if err != nil {
			return nil, invalid("%s.assertions[%d]: %s", prefix, j, err.Error())
		}

		ea.Position = j
		step.Assertions[j] = *ea
	}

	for j, v := range s.Variables {
		ev := entities.MonitorStepVariable{
			Name:     strings.TrimSpace(v.Name),
			Source:   v.Source,
			Property: strings.TrimSpace(v.Property),
		}

		if ev.Source == "" {
			ev.Source = multistep.VariableSourceJSONBody
		}

		if err := multistep.ValidateVariable(ev); err != nil {
			return nil, invalid("%s.variables[%d]: %s", prefix, j, err.Error())
		}

		step.Variables[j] = ev
	}

	return step, nil
}

func newAssertionEntity(a MonitorAssertion) (*entities.MonitorAssertion, error) {
	if len(a.Property) > maxAssertionFieldLength || len(a.Target) > maxAssertionFieldLength {
		return nil, fmt.Errorf("property/target is too long")
	}

	rule := asserter.Rule{
		Source:   a.Source,
		Property: strings.TrimSpace(a.Property),
		Operator: a.Operator,
		Target:   strings.TrimSpace(a.Target),
	}

	// These sources take no property; ignore leftovers from the form
	if rule.Source == "STATUS_CODE" || rule.Source == "RAW_BODY" {
		rule.Property = ""
	}

	if err := assertionValidator.IsRuleValid(rule); err != nil {
		return nil, err
	}

	return &entities.MonitorAssertion{
		Source:   rule.Source,
		Property: rule.Property,
		Operator: rule.Operator,
		Target:   rule.Target,
	}, nil
}

func validateStepURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("url is required")
	}

	if len(raw) > maxURLLength {
		return fmt.Errorf("url is too long")
	}

	// A URL that starts with a variable (e.g. "{{BASE_URL}}/users") is only
	// known at runtime.
	if strings.HasPrefix(raw, "{{") {
		return nil
	}

	if !isHTTPURL(multistep.StripPlaceholders(raw)) {
		return fmt.Errorf("url must be an absolute http(s) url")
	}

	return nil
}

func isHTTPURL(raw string) bool {
	u, err := url.Parse(raw)

	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func containsUint(list []uint64, v uint64) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}

	return false
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}

	return false
}

/*
	Entity -> response
*/

func newMonitorResponse(m *entities.Monitor) Monitor {
	steps := multistep.Steps(m)

	resp := Monitor{
		ID:        m.ID,
		State:     m.GetStateString(),
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		Settings:  newSettingsResponse(&m.Settings),
		Steps:     make([]MonitorStep, len(steps)),
	}

	for i := range steps {
		resp.Steps[i] = newStepResponse(&steps[i])
	}

	return resp
}

func newSettingsResponse(s *entities.MonitorSettings) MonitorSettings {
	authMethod := s.Auth.Method
	if authMethod == "" {
		authMethod = entities.MonitorAuthMethodNone
	}

	return MonitorSettings{
		FrequencySeconds: s.GetFrequencySeconds(),
		TLS: MonitorSettingsTLS{
			Enabled:                 s.TLS.Enabled,
			VerifyHostname:          s.TLS.VerifyHostname,
			CheckExpiration:         s.TLS.CheckExpiration,
			ExpirationThresholdDays: s.TLS.ExpirationThresholdDays,
		},
		Auth: MonitorSettingsAuth{
			Method:       authMethod,
			TokenURL:     s.Auth.TokenURL,
			ClientID:     s.Auth.ClientID,
			ClientSecret: s.Auth.ClientSecret,
			Username:     s.Auth.Username,
			Password:     s.Auth.Password,
		},
		Locations: monitor.NormalizeLocations(s.Locations),
	}
}

func newStepResponse(s *entities.MonitorStep) MonitorStep {
	step := MonitorStep{
		Name:       s.Name,
		Method:     s.Method,
		URL:        s.URL,
		Headers:    make([]MonitorSettingsHeader, len(s.Headers)),
		Assertions: make([]MonitorAssertion, len(s.Assertions)),
		Variables:  make([]MonitorVariable, len(s.Variables)),
		Body: MonitorSettingsBody{
			Type:    s.Body.Type,
			Content: s.Body.GetContentString(),
		},
	}

	if step.Body.Type == "" {
		step.Body.Type = "NONE"
	}

	for i, h := range s.Headers {
		step.Headers[i] = MonitorSettingsHeader{Key: h.Key, Value: h.Value}
	}

	for i, a := range s.Assertions {
		step.Assertions[i] = MonitorAssertion{
			Source:   a.Source,
			Property: a.Property,
			Operator: a.Operator,
			Target:   a.Target,
		}
	}

	for i, v := range s.Variables {
		step.Variables[i] = MonitorVariable{Name: v.Name, Source: v.Source, Property: v.Property}
	}

	return step
}
