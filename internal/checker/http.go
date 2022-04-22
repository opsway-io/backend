package checker

import (
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const (
	BodySizeLimit = 2048
	UserAgent     = "opsway 1.0.0"
)

type APICheckStatus struct {
	StatusCode   int
	ResponseTime int64
	Body         []byte
}

func APICheck(method, url string, headers map[string]string, body io.Reader, timeout time.Duration) (*APICheckStatus, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	req.Header.Set("User-Agent", UserAgent)

	if body != nil {
		req.Body = ioutil.NopCloser(body)
	}

	client := &http.Client{
		Timeout: timeout,
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}

	end := time.Now()

	respBody, err := ioutil.ReadAll(io.LimitReader(resp.Body, BodySizeLimit))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	resp.Body.Close()

	return &APICheckStatus{
		StatusCode:   resp.StatusCode,
		ResponseTime: end.Sub(start).Milliseconds(),
		Body:         respBody,
	}, nil
}
