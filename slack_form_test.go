package main

import (
	"bytes"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFormValue(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("payload=%7B%22type%22%3A%22block_actions%22%7D"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if c.FormValue("payload") != `{"type":"block_actions"}` {
		t.Errorf("expected parsed payload, got %s", c.FormValue("payload"))
	}
}
