package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"text/template"
	"time"
)

func TestScheduler_RegistersAllEntries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	tpl, _ := template.New("t").Parse("x")
	cfg := &Config{
		FireURL:   srv.URL,
		Token:     "tok",
		Schedules: []string{"0 17 * * *", "0 22 * * *", "*/5 * * * *"},
		Location:  time.UTC,
		Template:  tpl,
		LogLevel:  "info",
	}
	logger, _ := captureLogs()
	fire := &FireClient{
		URL:      srv.URL,
		Token:    "tok",
		HTTP:     &http.Client{Timeout: 1 * time.Second},
		Template: tpl,
		TZ:       time.UTC,
		Log:      logger,
	}

	s, err := NewScheduler(cfg, fire, logger)
	if err != nil {
		t.Fatalf("NewScheduler: %v", err)
	}
	if got := s.EntryCount(); got != 3 {
		t.Fatalf("EntryCount = %d, want 3", got)
	}
}

func TestScheduler_StartStop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	tpl, _ := template.New("t").Parse("x")
	cfg := &Config{
		FireURL:   srv.URL,
		Token:     "tok",
		Schedules: []string{"0 17 * * *"},
		Location:  time.UTC,
		Template:  tpl,
		LogLevel:  "info",
	}
	logger, _ := captureLogs()
	fire := &FireClient{
		URL:      srv.URL,
		Token:    "tok",
		HTTP:     &http.Client{Timeout: 1 * time.Second},
		Template: tpl,
		TZ:       time.UTC,
		Log:      logger,
	}

	s, err := NewScheduler(cfg, fire, logger)
	if err != nil {
		t.Fatalf("NewScheduler: %v", err)
	}
	s.Start()

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() {
		s.Stop(stopCtx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop did not return within 3s")
	}
}

func TestNewLogger_UnknownLevelFallsBackToInfo(t *testing.T) {
	l := newLogger("zoinks")
	if l == nil {
		t.Fatal("logger is nil")
	}
}

func TestNewLogger_KnownLevels(t *testing.T) {
	for _, lvl := range []string{"", "debug", "info", "warn", "warning", "error", "ERROR", "  Debug  "} {
		if l := newLogger(lvl); l == nil {
			t.Fatalf("nil logger for %q", lvl)
		}
	}
}
