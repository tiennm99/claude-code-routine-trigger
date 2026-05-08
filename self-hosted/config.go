package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/robfig/cron/v3"
)

const defaultTextTemplate = "Scheduled trigger at {{.LocalTime}}"

// Config is the validated runtime configuration assembled from env vars.
type Config struct {
	FireURL   string
	Token     string
	Schedules []string
	Location  *time.Location
	Template  *template.Template
	LogLevel  string
}

// Load reads env vars and returns a fully validated Config or an error.
// All required vars must be present and all parsed values must pass validation.
func Load() (*Config, error) {
	cfg := &Config{}

	cfg.FireURL = strings.TrimSpace(os.Getenv("ROUTINE_FIRE_URL"))
	if cfg.FireURL == "" {
		return nil, errors.New("missing required env var ROUTINE_FIRE_URL")
	}

	cfg.Token = strings.TrimSpace(os.Getenv("ROUTINE_FIRE_TOKEN"))
	if cfg.Token == "" {
		return nil, errors.New("missing required env var ROUTINE_FIRE_TOKEN")
	}

	rawCron := os.Getenv("CRON_SCHEDULE")
	schedules, err := parseSchedules(rawCron)
	if err != nil {
		return nil, err
	}
	cfg.Schedules = schedules

	tz := strings.TrimSpace(os.Getenv("TZ"))
	if tz == "" {
		cfg.Location = time.UTC
	} else {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return nil, fmt.Errorf("invalid TZ %q: %w", tz, err)
		}
		cfg.Location = loc
	}

	rawTpl := os.Getenv("TEXT_TEMPLATE")
	if rawTpl == "" {
		rawTpl = defaultTextTemplate
	}
	tpl, err := template.New("text").Parse(rawTpl)
	if err != nil {
		return nil, fmt.Errorf("invalid TEXT_TEMPLATE: %w", err)
	}
	cfg.Template = tpl

	cfg.LogLevel = strings.TrimSpace(os.Getenv("LOG_LEVEL"))
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	return cfg, nil
}

var scheduleSeparators = regexp.MustCompile(`[;\n]+`)

// parseSchedules splits CRON_SCHEDULE on `;` or newlines, trims, drops empties,
// and validates each expression against the standard 5-field parser.
func parseSchedules(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("missing required env var CRON_SCHEDULE")
	}
	parts := scheduleSeparators.Split(raw, -1)
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		if _, err := parser.Parse(s); err != nil {
			return nil, fmt.Errorf("invalid cron %q: %w", s, err)
		}
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil, errors.New("CRON_SCHEDULE has no non-empty expressions")
	}
	return out, nil
}
