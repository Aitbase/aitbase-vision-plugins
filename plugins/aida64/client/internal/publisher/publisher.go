// Package publisher writes the metrics into the device's AIDA64 face (example.aida64) with a face
// push token (device decision D35): POST /api/v1/plugins/face-values, Authorization: Bearer.
package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

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
// face.1 to face.12 in their order, then the texts the device's old built-in card worked out from
// them: face.13 the disk's used percentage ("60%"), face.14 the RAM's "used / total", face.15 the
// disk's total and face.16 the CPU header. Empty metrics are left out, and each value is cut to the
// device's 64 bytes on a character boundary. The device's old AIDA64 route derives the same texts.
func FaceValues(m collector.Metrics) map[string]string {
	ordered := []string{m.RAMSpeed, m.RAMUsage, m.RAMFree, m.RAMLoad, m.CPUName, m.CPULoad, m.CPUTemp,
		m.CPUClock, m.CPUFan, m.StorageDisk, m.StorageUsage, m.StorageTemp}
	values := map[string]string{}
	for i, v := range ordered {
		v = clean(v)
		if v != "" {
			values[fmt.Sprintf("face.%d", i+1)] = v
		}
	}
	values["face.13"] = fmt.Sprintf("%d%%", StoragePercent(m.StorageUsage))
	values["face.14"] = clean(RAMUsageText(m.RAMUsage, m.RAMFree))
	values["face.15"] = clean(StorageTotalText(m.StorageUsage))
	values["face.16"] = clean(CPUHeaderText(m.CPUName))
	return values
}

// amount is "14.2GB" or "512 MB" split into its number and unit, as the old card read them.
func amount(text string) (float64, string, bool) {
	text = strings.TrimLeftFunc(text, unicode.IsSpace)
	end := 0
	for end < len(text) && strings.ContainsRune("+-.0123456789eE", rune(text[end])) {
		end++
	}
	for end > 0 {
		if v, err := strconv.ParseFloat(text[:end], 64); err == nil {
			unit := strings.TrimLeftFunc(text[end:], unicode.IsSpace)
			if i := strings.IndexFunc(unit, unicode.IsSpace); i >= 0 {
				unit = unit[:i]
			}
			return v, unit, true
		}
		end--
	}
	return 0, "", false
}

// unitScale is a unit's bytes (B, KB, MB, GB, TB, any case; none is 1), or 0 for another unit.
func unitScale(unit string) float64 {
	scales := map[string]float64{"": 1, "B": 1, "KB": 1 << 10, "MB": 1 << 20, "GB": 1 << 30, "TB": 1 << 40}
	return scales[strings.ToUpper(unit)]
}

// StoragePercent is "1.2TB/2.0TB" as a whole percentage (60), a plain "45%" as 45, 0 when it can't tell.
func StoragePercent(ratio string) int {
	clamp := func(v int) int { return min(max(v, 0), 100) }
	used, total, found := strings.Cut(ratio, "/")
	if !found {
		v, _, ok := amount(ratio)
		if !ok {
			return 0
		}
		return clamp(int(v))
	}
	u, uu, ok1 := amount(used)
	t, tu, ok2 := amount(total)
	if !ok1 || !ok2 || unitScale(uu) == 0 || unitScale(tu) == 0 || t <= 0 {
		return 0
	}
	return clamp(int(math.Round(u * unitScale(uu) / (t * unitScale(tu)) * 100)))
}

// RAMUsageText is "14 / 32 GB" from used "14GB" and free "18GB" (one decimal unless both are whole);
// the used text alone when the units differ or don't parse.
func RAMUsageText(used, free string) string {
	u, uu, ok1 := amount(used)
	f, fu, ok2 := amount(free)
	if !ok1 || !ok2 || uu != fu {
		if used == "" {
			return "--"
		}
		return used
	}
	total := u + f
	format := "%.1f / %.1f"
	if math.Abs(u-math.Round(u)) < 0.05 && math.Abs(total-math.Round(total)) < 0.05 {
		format, u, total = "%.0f / %.0f", math.Round(u), math.Round(total)
	}
	text := fmt.Sprintf(format, u, total)
	if uu != "" {
		text += " " + uu
	}
	return text
}

// StorageTotalText is the total of "1.2TB/2.0TB" ("2.0TB"), the whole text without a slash, "--" when
// empty. Like the old card, it cuts "N/A" at its slash ("A").
func StorageTotalText(ratio string) string {
	if ratio == "" {
		return "--"
	}
	total := ratio
	if _, after, found := strings.Cut(ratio, "/"); found {
		total = after
	}
	total = strings.TrimLeftFunc(total, unicode.IsSpace)
	if total == "" {
		return "--"
	}
	return total
}

// CPUHeaderText is "CPU - <name>", "CPU - Unknown CPU" without one.
func CPUHeaderText(name string) string {
	if name == "" {
		name = "Unknown CPU"
	}
	return "CPU - " + name
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
