package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"text/template"
	"time"
)

func captureLogs() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(h), &buf
}

func mustTemplate(t *testing.T, raw string) *template.Template {
	t.Helper()
	tpl, err := template.New("t").Parse(raw)
	if err != nil {
		t.Fatalf("template parse: %v", err)
	}
	return tpl
}

func TestFire_HappyPath(t *testing.T) {
	var gotMethod, gotAuth, gotVersion, gotBeta, gotContentType string
	var gotBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotVersion = r.Header.Get("anthropic-version")
		gotBeta = r.Header.Get("anthropic-beta")
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"type":"routine_fire","claude_code_session_id":"sess_1","claude_code_session_url":"https://claude.ai/code/sess_1"}`))
	}))
	defer srv.Close()

	logger, logbuf := captureLogs()
	c := &FireClient{
		URL:      srv.URL,
		Token:    "tok",
		HTTP:     &http.Client{Timeout: 5 * time.Second},
		Template: mustTemplate(t, "fire {{.Cron}}"),
		TZ:       time.UTC,
		Log:      logger,
	}

	if err := c.Fire(context.Background(), "0 9 * * *"); err != nil {
		t.Fatalf("Fire returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q", gotMethod)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("auth = %q", gotAuth)
	}
	if gotVersion != headerVersion {
		t.Errorf("anthropic-version = %q", gotVersion)
	}
	if gotBeta != headerBeta {
		t.Errorf("anthropic-beta = %q", gotBeta)
	}
	if gotContentType != "application/json" {
		t.Errorf("content-type = %q", gotContentType)
	}

	var body fireRequest
	if err := json.Unmarshal(gotBody, &body); err != nil {
		t.Fatalf("body unmarshal: %v", err)
	}
	if body.Text != "fire 0 9 * * *" {
		t.Errorf("body.text = %q", body.Text)
	}

	if !strings.Contains(logbuf.String(), "fire ok") {
		t.Errorf("expected 'fire ok' log, got: %s", logbuf.String())
	}
	if !strings.Contains(logbuf.String(), "https://claude.ai/code/sess_1") {
		t.Errorf("expected session url in log, got: %s", logbuf.String())
	}
}

func TestFire_NonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	logger, logbuf := captureLogs()
	c := &FireClient{
		URL:      srv.URL,
		Token:    "tok",
		HTTP:     &http.Client{Timeout: 5 * time.Second},
		Template: mustTemplate(t, "x"),
		TZ:       time.UTC,
		Log:      logger,
	}

	if err := c.Fire(context.Background(), "0 9 * * *"); err != nil {
		t.Fatalf("Fire should swallow non-2xx, got error: %v", err)
	}
	out := logbuf.String()
	if !strings.Contains(out, "non-2xx") {
		t.Errorf("expected non-2xx log, got: %s", out)
	}
	if !strings.Contains(out, "401") {
		t.Errorf("expected status 401 in log, got: %s", out)
	}
}

func TestFire_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // close immediately so requests fail

	logger, logbuf := captureLogs()
	c := &FireClient{
		URL:      srv.URL,
		Token:    "tok",
		HTTP:     &http.Client{Timeout: 1 * time.Second},
		Template: mustTemplate(t, "x"),
		TZ:       time.UTC,
		Log:      logger,
	}

	if err := c.Fire(context.Background(), "0 9 * * *"); err != nil {
		t.Fatalf("Fire should swallow network error, got: %v", err)
	}
	if !strings.Contains(logbuf.String(), "fire request failed") {
		t.Errorf("expected request-failed log, got: %s", logbuf.String())
	}
}

func TestFire_TemplateUsesNowFunc(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	fixed := time.Date(2026, 5, 8, 23, 30, 0, 0, time.UTC)
	logger, _ := captureLogs()
	c := &FireClient{
		URL:      srv.URL,
		Token:    "tok",
		HTTP:     &http.Client{Timeout: 5 * time.Second},
		Template: mustTemplate(t, "{{.LocalTime}}"),
		TZ:       time.UTC,
		Log:      logger,
		NowFunc:  func() time.Time { return fixed },
	}

	if err := c.Fire(context.Background(), "0 9 * * *"); err != nil {
		t.Fatalf("Fire: %v", err)
	}
	var body fireRequest
	if err := json.Unmarshal(gotBody, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Text != "2026-05-08 23:30 UTC" {
		t.Errorf("body.text = %q", body.Text)
	}
}
