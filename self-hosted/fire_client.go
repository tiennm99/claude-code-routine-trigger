package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"text/template"
	"time"
)

const (
	headerVersion = "2023-06-01"
	headerBeta    = "experimental-cc-routine-2026-04-01"
)

// FireClient POSTs to the Anthropic /fire endpoint with the headers and body
// shape required by the Claude Code routines beta. Each call creates a new
// session; the daemon never retries on failure.
type FireClient struct {
	URL      string
	Token    string
	HTTP     *http.Client
	Template *template.Template
	TZ       *time.Location
	Log      *slog.Logger
	NowFunc  func() time.Time // overridable for tests; defaults to time.Now
}

type fireRequest struct {
	Text string `json:"text"`
}

type fireResponse struct {
	Type                 string `json:"type"`
	ClaudeCodeSessionID  string `json:"claude_code_session_id"`
	ClaudeCodeSessionURL string `json:"claude_code_session_url"`
}

// Fire renders the text template, POSTs to the routine /fire endpoint, and
// logs the outcome. Returns nil on transport-level success regardless of HTTP
// status — non-2xx is logged at error level so the daemon stays up.
// Returns a non-nil error only for unrecoverable client-side problems
// (template render failure or request build failure).
func (c *FireClient) Fire(ctx context.Context, cronExpr string) error {
	now := c.now()
	text, err := c.renderText(now, cronExpr)
	if err != nil {
		return fmt.Errorf("render text: %w", err)
	}

	body, err := json.Marshal(fireRequest{Text: text})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("anthropic-version", headerVersion)
	req.Header.Set("anthropic-beta", headerBeta)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		c.Log.Error("fire request failed", "cron", cronExpr, "err", err.Error())
		return nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.Log.Error("fire non-2xx response",
			"cron", cronExpr,
			"status", resp.StatusCode,
			"body", string(respBody))
		return nil
	}

	var fr fireResponse
	if err := json.Unmarshal(respBody, &fr); err != nil {
		c.Log.Warn("fire 2xx but body unparseable",
			"cron", cronExpr,
			"status", resp.StatusCode,
			"body", string(respBody))
		return nil
	}
	c.Log.Info("fire ok",
		"cron", cronExpr,
		"session_url", fr.ClaudeCodeSessionURL,
		"session_id", fr.ClaudeCodeSessionID)
	return nil
}

func (c *FireClient) now() time.Time {
	if c.NowFunc != nil {
		return c.NowFunc()
	}
	return time.Now()
}

func (c *FireClient) renderText(now time.Time, cronExpr string) (string, error) {
	data := map[string]any{
		"Now":       now,
		"LocalTime": now.In(c.TZ).Format("2006-01-02 15:04 MST"),
		"Cron":      cronExpr,
	}
	var buf bytes.Buffer
	if err := c.Template.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
