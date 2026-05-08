package main

import (
	"strings"
	"testing"
	"time"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"ROUTINE_FIRE_URL", "ROUTINE_FIRE_TOKEN", "CRON_SCHEDULE", "TZ", "TEXT_TEMPLATE", "LOG_LEVEL"} {
		t.Setenv(k, "")
	}
}

func setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ROUTINE_FIRE_URL", "https://example.test/fire")
	t.Setenv("ROUTINE_FIRE_TOKEN", "tok")
	t.Setenv("CRON_SCHEDULE", "0 9 * * *")
}

func TestLoad_Happy(t *testing.T) {
	clearEnv(t)
	setBaseEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Schedules) != 1 || cfg.Schedules[0] != "0 9 * * *" {
		t.Fatalf("schedules = %v", cfg.Schedules)
	}
	if cfg.Location != time.UTC {
		t.Fatalf("default tz should be UTC, got %v", cfg.Location)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("default log level should be info, got %q", cfg.LogLevel)
	}
}

func TestLoad_MultiCronSemicolon(t *testing.T) {
	clearEnv(t)
	setBaseEnv(t)
	t.Setenv("CRON_SCHEDULE", "0 17 * * *;0 22 * * *")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Schedules) != 2 {
		t.Fatalf("want 2, got %d: %v", len(cfg.Schedules), cfg.Schedules)
	}
}

func TestLoad_MultiCronNewline(t *testing.T) {
	clearEnv(t)
	setBaseEnv(t)
	t.Setenv("CRON_SCHEDULE", "0 17 * * *\n0 22 * * *")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Schedules) != 2 {
		t.Fatalf("want 2, got %d: %v", len(cfg.Schedules), cfg.Schedules)
	}
}

func TestLoad_MixedSeparatorsAndWhitespace(t *testing.T) {
	clearEnv(t)
	setBaseEnv(t)
	t.Setenv("CRON_SCHEDULE", " 0 17 * * * ;\n  0 22 * * *  ;;\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Schedules) != 2 {
		t.Fatalf("want 2 trimmed schedules, got %v", cfg.Schedules)
	}
	if cfg.Schedules[0] != "0 17 * * *" || cfg.Schedules[1] != "0 22 * * *" {
		t.Fatalf("unexpected trimmed values: %v", cfg.Schedules)
	}
}

func TestLoad_MissingFireURL(t *testing.T) {
	clearEnv(t)
	t.Setenv("ROUTINE_FIRE_TOKEN", "tok")
	t.Setenv("CRON_SCHEDULE", "0 9 * * *")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "ROUTINE_FIRE_URL") {
		t.Fatalf("want missing url error, got %v", err)
	}
}

func TestLoad_MissingToken(t *testing.T) {
	clearEnv(t)
	t.Setenv("ROUTINE_FIRE_URL", "https://example.test/fire")
	t.Setenv("CRON_SCHEDULE", "0 9 * * *")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "ROUTINE_FIRE_TOKEN") {
		t.Fatalf("want missing token error, got %v", err)
	}
}

func TestLoad_EmptyCron(t *testing.T) {
	clearEnv(t)
	t.Setenv("ROUTINE_FIRE_URL", "https://example.test/fire")
	t.Setenv("ROUTINE_FIRE_TOKEN", "tok")
	t.Setenv("CRON_SCHEDULE", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "CRON_SCHEDULE") {
		t.Fatalf("want missing cron error, got %v", err)
	}
}

func TestLoad_InvalidCron(t *testing.T) {
	clearEnv(t)
	setBaseEnv(t)
	t.Setenv("CRON_SCHEDULE", "not-a-cron")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "invalid cron") {
		t.Fatalf("want invalid cron error, got %v", err)
	}
}

func TestLoad_InvalidTZ(t *testing.T) {
	clearEnv(t)
	setBaseEnv(t)
	t.Setenv("TZ", "Atlantis/Lost")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "invalid TZ") {
		t.Fatalf("want invalid TZ error, got %v", err)
	}
}

func TestLoad_DefaultTemplate(t *testing.T) {
	clearEnv(t)
	setBaseEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out, err := renderTemplate(cfg, time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC), "0 9 * * *")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.HasPrefix(out, "Scheduled trigger at ") {
		t.Fatalf("default template output unexpected: %q", out)
	}
}

func TestLoad_CustomTemplate(t *testing.T) {
	clearEnv(t)
	setBaseEnv(t)
	t.Setenv("TEXT_TEMPLATE", "hi {{.Cron}}")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out, err := renderTemplate(cfg, time.Now(), "0 9 * * *")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if out != "hi 0 9 * * *" {
		t.Fatalf("custom template output: %q", out)
	}
}

func TestLoad_InvalidTemplate(t *testing.T) {
	clearEnv(t)
	setBaseEnv(t)
	t.Setenv("TEXT_TEMPLATE", "{{.broken")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "invalid TEXT_TEMPLATE") {
		t.Fatalf("want invalid template error, got %v", err)
	}
}

func TestLoad_LogLevelDefault(t *testing.T) {
	clearEnv(t)
	setBaseEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("got %q", cfg.LogLevel)
	}
}

// renderTemplate runs cfg.Template with the documented vars; mirror of
// FireClient.renderText for test isolation.
func renderTemplate(cfg *Config, now time.Time, expr string) (string, error) {
	data := map[string]any{
		"Now":       now,
		"LocalTime": now.In(cfg.Location).Format("2006-01-02 15:04 MST"),
		"Cron":      expr,
	}
	var sb strings.Builder
	if err := cfg.Template.Execute(&sb, data); err != nil {
		return "", err
	}
	return sb.String(), nil
}
