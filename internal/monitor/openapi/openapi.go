// Package openapi turns OpenAPI 3 / Swagger 2 documents into monitor step
// suggestions.
package openapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	maxSpecBytes          = 10 << 20 // 10 MB
	maxSchemaDepth        = 6
	maxBodyKeyAssertions  = 10
	defaultSuccessStatus  = "200"
	fetchTimeout          = 15 * time.Second
	authOAuth2ClientCreds = "OAUTH2_CLIENT_CREDENTIALS"
	authBasic             = "BASIC"
)

var (
	ErrInvalidURL  = errors.New("invalid url")
	ErrInvalidSpec = errors.New("invalid OpenAPI document")
)

type Assertion struct {
	Source   string `json:"source"`
	Property string `json:"property,omitempty"`
	Operator string `json:"operator"`
	Target   string `json:"target,omitempty"`
}

type Endpoint struct {
	Method      string      `json:"method"`
	Path        string      `json:"path"`
	Summary     string      `json:"summary"`
	RequestBody *string     `json:"requestBody,omitempty"`
	StatusCode  string      `json:"statusCode"`
	Assertions  []Assertion `json:"assertions,omitempty"`
}

type Auth struct {
	Method   string `json:"method"`
	TokenURL string `json:"tokenUrl,omitempty"`
}

type Preview struct {
	Endpoints []Endpoint `json:"endpoints"`
	Auth      *Auth      `json:"auth,omitempty"`
	// BaseURL is the absolute API base URL (server URL resolved against the
	// spec URL). Endpoint paths are relative to the origin of the spec URL.
	BaseURL string `json:"baseUrl,omitempty"`
}

// Fetch downloads the document at rawURL and parses it.
func Fetch(ctx context.Context, client *http.Client, rawURL string) (*Preview, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, ErrInvalidURL
	}

	if client == nil {
		client = &http.Client{Timeout: fetchTimeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, ErrInvalidURL
	}

	req.Header.Set("Accept", "application/json, application/yaml;q=0.9, */*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch document")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("failed to fetch document: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSpecBytes+1))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read document")
	}

	if len(data) > maxSpecBytes {
		return nil, fmt.Errorf("document is larger than %d bytes", maxSpecBytes)
	}

	return Parse(data, u)
}

type doc = map[string]interface{}

// Parse parses an OpenAPI 3 or Swagger 2 document (JSON or YAML). specURL is
// used to resolve relative server URLs; it may be nil.
func Parse(data []byte, specURL *url.URL) (*Preview, error) {
	var root doc
	if err := json.Unmarshal(data, &root); err != nil {
		if yerr := yaml.Unmarshal(data, &root); yerr != nil || root == nil {
			return nil, ErrInvalidSpec
		}
	}

	_, isV3 := root["openapi"]
	_, isV2 := root["swagger"]

	if !isV3 && !isV2 {
		return nil, ErrInvalidSpec
	}

	p := &parser{root: root}

	basePath, baseURL := p.serverBase(isV3, specURL)

	preview := &Preview{
		Endpoints: []Endpoint{},
		Auth:      p.auth(isV3),
		BaseURL:   baseURL,
	}

	paths := asMap(root["paths"])

	pathKeys := make([]string, 0, len(paths))
	for k := range paths {
		pathKeys = append(pathKeys, k)
	}

	sort.Strings(pathKeys)

	for _, path := range pathKeys {
		item := p.resolve(asMap(paths[path]))

		for _, method := range []string{"get", "post", "put", "patch", "delete", "head", "options"} {
			op := asMap(item[method])
			if op == nil {
				continue
			}

			preview.Endpoints = append(preview.Endpoints, p.endpoint(isV3, method, joinPath(basePath, path), item, op))
		}
	}

	return preview, nil
}

type parser struct {
	root doc
}

func (p *parser) endpoint(isV3 bool, method, path string, item, op doc) Endpoint {
	summary := asString(op["summary"])
	if summary == "" {
		summary = asString(op["operationId"])
	}

	e := Endpoint{
		Method:  strings.ToUpper(method),
		Path:    path,
		Summary: summary,
	}

	status, respSchema := p.successResponse(isV3, op)
	e.StatusCode = status

	if body := p.requestBody(isV3, item, op); body != nil {
		e.RequestBody = body
	}

	e.Assertions = []Assertion{{Source: "STATUS_CODE", Operator: "EQUAL", Target: status}}

	if respSchema != nil {
		for _, key := range p.requiredKeys(respSchema) {
			e.Assertions = append(e.Assertions, Assertion{
				Source:   "JSON_BODY",
				Property: "$." + key,
				Operator: "HAS_KEY",
			})
		}
	}

	return e
}

// successResponse returns the first 2xx status code and its JSON schema (if any).
func (p *parser) successResponse(isV3 bool, op doc) (string, doc) {
	responses := asMap(op["responses"])

	codes := make([]string, 0, len(responses))
	for code := range responses {
		if strings.HasPrefix(code, "2") && len(code) == 3 {
			if _, err := strconv.Atoi(code); err == nil {
				codes = append(codes, code)
			}
		}
	}

	if len(codes) == 0 {
		return defaultSuccessStatus, nil
	}

	sort.Strings(codes)
	code := codes[0]
	resp := p.resolve(asMap(responses[code]))

	if isV3 {
		return code, p.resolve(asMap(jsonMediaType(asMap(resp["content"]))["schema"]))
	}

	return code, p.resolve(asMap(resp["schema"]))
}

func (p *parser) requestBody(isV3 bool, item, op doc) *string {
	var (
		schema  doc
		example interface{}
	)

	if isV3 {
		rb := p.resolve(asMap(op["requestBody"]))
		media := jsonMediaType(asMap(rb["content"]))
		if media == nil {
			return nil
		}

		schema = p.resolve(asMap(media["schema"]))
		example = media["example"]

		if example == nil {
			for _, ex := range asMap(media["examples"]) {
				example = p.resolve(asMap(ex))["value"]
				if example != nil {
					break
				}
			}
		}
	} else {
		params := append(asSlice(item["parameters"]), asSlice(op["parameters"])...)
		for _, raw := range params {
			param := p.resolve(asMap(raw))
			if asString(param["in"]) == "body" {
				schema = p.resolve(asMap(param["schema"]))
			}
		}

		if schema == nil {
			return nil
		}
	}

	if example == nil && schema != nil {
		example = schema["example"]
	}

	if example == nil && schema != nil {
		example = p.sample(schema, 0)
	}

	if example == nil {
		return nil
	}

	b, err := json.MarshalIndent(example, "", "  ")
	if err != nil {
		return nil
	}

	s := string(b)

	return &s
}

// sample generates an example value for a schema.
func (p *parser) sample(schema doc, depth int) interface{} {
	schema = p.resolve(schema)
	if schema == nil || depth > maxSchemaDepth {
		return nil
	}

	if ex, ok := schema["example"]; ok {
		return ex
	}

	if def, ok := schema["default"]; ok {
		return def
	}

	if enum := asSlice(schema["enum"]); len(enum) > 0 {
		return enum[0]
	}

	for _, key := range []string{"allOf", "oneOf", "anyOf"} {
		if parts := asSlice(schema[key]); len(parts) > 0 {
			if key != "allOf" {
				return p.sample(asMap(parts[0]), depth+1)
			}

			merged := map[string]interface{}{}
			for _, part := range parts {
				if obj, ok := p.sample(asMap(part), depth+1).(map[string]interface{}); ok {
					for k, v := range obj {
						merged[k] = v
					}
				}
			}

			return merged
		}
	}

	typ := asString(schema["type"])
	if typ == "" && schema["properties"] != nil {
		typ = "object"
	}

	switch typ {
	case "object":
		obj := map[string]interface{}{}
		for name, prop := range asMap(schema["properties"]) {
			obj[name] = p.sample(asMap(prop), depth+1)
		}

		return obj
	case "array":
		item := p.sample(asMap(schema["items"]), depth+1)
		if item == nil {
			return []interface{}{}
		}

		return []interface{}{item}
	case "integer":
		return 0
	case "number":
		return 0.0
	case "boolean":
		return true
	case "string":
		switch asString(schema["format"]) {
		case "date-time":
			return "2024-01-01T00:00:00Z"
		case "date":
			return "2024-01-01"
		case "email":
			return "user@example.com"
		case "uuid":
			return "00000000-0000-0000-0000-000000000000"
		default:
			return "string"
		}
	default:
		return nil
	}
}

// requiredKeys returns the required top-level properties of an object schema.
func (p *parser) requiredKeys(schema doc) []string {
	schema = p.resolve(schema)
	if schema == nil {
		return nil
	}

	var keys []string
	for _, k := range asSlice(schema["required"]) {
		if s, ok := k.(string); ok && s != "" {
			keys = append(keys, s)
		}
	}

	for _, part := range asSlice(schema["allOf"]) {
		keys = append(keys, p.requiredKeys(asMap(part))...)
	}

	if len(keys) > maxBodyKeyAssertions {
		keys = keys[:maxBodyKeyAssertions]
	}

	return keys
}

func (p *parser) auth(isV3 bool) *Auth {
	var schemes doc
	if isV3 {
		schemes = asMap(asMap(p.root["components"])["securitySchemes"])
	} else {
		schemes = asMap(p.root["securityDefinitions"])
	}

	names := make([]string, 0, len(schemes))
	for name := range schemes {
		names = append(names, name)
	}

	sort.Strings(names)

	var basic *Auth

	for _, name := range names {
		s := p.resolve(asMap(schemes[name]))
		typ := asString(s["type"])

		switch {
		case typ == "oauth2" && isV3:
			if cc := asMap(asMap(s["flows"])["clientCredentials"]); cc != nil {
				return &Auth{Method: authOAuth2ClientCreds, TokenURL: asString(cc["tokenUrl"])}
			}
		case typ == "oauth2" && !isV3:
			if asString(s["flow"]) == "application" {
				return &Auth{Method: authOAuth2ClientCreds, TokenURL: asString(s["tokenUrl"])}
			}
		case typ == "basic", typ == "http" && strings.EqualFold(asString(s["scheme"]), "basic"):
			basic = &Auth{Method: authBasic}
		}
	}

	return basic
}

// serverBase returns the path prefix to put in front of every endpoint path
// and the absolute base URL of the API.
func (p *parser) serverBase(isV3 bool, specURL *url.URL) (string, string) {
	var raw string

	if isV3 {
		if servers := asSlice(p.root["servers"]); len(servers) > 0 {
			server := asMap(servers[0])
			raw = asString(server["url"])

			// Substitute server variables with their defaults
			for name, v := range asMap(server["variables"]) {
				raw = strings.ReplaceAll(raw, "{"+name+"}", asString(asMap(v)["default"]))
			}
		}
	} else {
		basePath := asString(p.root["basePath"])
		if host := asString(p.root["host"]); host != "" {
			scheme := "https"
			if schemes := asSlice(p.root["schemes"]); len(schemes) > 0 {
				scheme = asString(schemes[0])
			}

			raw = scheme + "://" + host + basePath
		} else {
			raw = basePath
		}
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", ""
	}

	if specURL != nil {
		u = specURL.ResolveReference(u)
		// A server URL like "/" resolves to the spec path's directory
		if raw == "" {
			u.Path = ""
		}
	}

	base := strings.TrimRight(u.Path, "/")

	if !u.IsAbs() {
		return base, ""
	}

	u.RawQuery = ""
	u.Fragment = ""
	u.Path = base

	return base, u.String()
}

// resolve follows local $ref pointers ("#/components/schemas/X").
func (p *parser) resolve(m doc) doc {
	for i := 0; i < 10 && m != nil; i++ {
		ref, ok := m["$ref"].(string)
		if !ok {
			return m
		}

		m = p.lookup(ref)
	}

	return m
}

func (p *parser) lookup(ref string) doc {
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}

	var cur interface{} = p.root
	for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		cur = asMap(cur)[part]
	}

	return asMap(cur)
}

func jsonMediaType(content doc) doc {
	if content == nil {
		return nil
	}

	if m := asMap(content["application/json"]); m != nil {
		return m
	}

	keys := make([]string, 0, len(content))
	for k := range content {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		if strings.Contains(k, "json") {
			return asMap(content[k])
		}
	}

	return nil
}

func joinPath(base, path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return strings.TrimRight(base, "/") + path
}

func asMap(v interface{}) doc {
	switch m := v.(type) {
	case map[string]interface{}:
		return m
	case map[interface{}]interface{}:
		out := make(doc, len(m))
		for k, val := range m {
			out[fmt.Sprintf("%v", k)] = val
		}

		return out
	default:
		return nil
	}
}

func asSlice(v interface{}) []interface{} {
	s, _ := v.([]interface{})

	return s
}

func asString(v interface{}) string {
	switch s := v.(type) {
	case string:
		return s
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", s)
	}
}
