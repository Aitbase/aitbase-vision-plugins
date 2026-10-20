// Package publisher writes the metrics into the device's AIDA64 face (example.aida64) with a face
// push token (device decision D35): POST /api/v1/plugins/face-values, Authorization: Bearer.
package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"aida64-client/internal/collector"
)

// ValuesPath is the device's face push route (D35).
const ValuesPath = "/api/v1/plugins/face-values"

// maxValueBytes is the device's limit for one face value.
const maxValueBytes = 64

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func New(baseURL, token string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &Client{
		baseURL:    normalizeBaseURL(baseURL),
		token:      strings.TrimSpace(token),
		httpClient: &http.Client{Timeout: timeout},
	}
}

// FaceValues maps the metrics to the face's keys (plugins/aida64/README.md): the 12 metrics as
// face.1 to face.12 in their order, and the disk bar's percentage as face.13. Empty values are left
// out, and each value is cut to the device's 64 bytes on a character boundary.
func FaceValues(m collector.Metrics) map[string]string {
	ordered := []string{m.RAMSpeed, m.RAMUsage, m.RAMFree, m.RAMLoad, m.CPUName, m.CPULoad, m.CPUTemp,
		m.CPUClock, m.CPUFan, m.StorageDisk, m.StorageUsage, m.StorageTemp, m.StorageUsedPct}
	values := map[string]string{}
	for i, v := range ordered {
		v = clean(v)
		if v != "" {
			values[fmt.Sprintf("face.%d", i+1)] = v
		}
	}
	return values
}

// clean drops control characters, which the device refuses, and cuts to maxValueBytes.
func clean(v string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(v) {
		if r < 0x20 || r == 0x7f || (r >= 0x200b && r <= 0x200f) || (r >= 0x202a && r <= 0x202e) ||
			(r >= 0x2066 && r <= 0x2069) || r == 0xfeff {
			continue
		}
		if b.Len()+len(string(r)) > maxValueBytes {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (c *Client) PushMetrics(ctx context.Context, m collector.Metrics) (string, error) {
	if c.token == "" {
		return "", fmt.Errorf("no face push token: create one on the Device Console (Display, Push tokens)")
	}
	return c.postJSON(ctx, ValuesPath, map[string]any{"values": FaceValues(m)})
}

func (c *Client) postJSON(ctx context.Context, path string, payload any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	respText := strings.TrimSpace(string(respBody))
	if respText == "" {
		respText = "{}"
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("%s status %d: %s", path, resp.StatusCode, respText)
	}
	return respText, nil
}

func normalizeBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "http://127.0.0.1"
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "http://" + raw
	}
	return strings.TrimRight(raw, "/")
}
