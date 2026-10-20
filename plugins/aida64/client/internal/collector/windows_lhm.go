package collector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var tempRegex = regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)
var fanRegex = regexp.MustCompile(`[-+]?\d+`)

type WindowsLHMProvider struct {
	httpClient *http.Client
	url        string
	file       string
}

func NewWindowsLHMProvider(url, file string) *WindowsLHMProvider {
	url = strings.TrimSpace(url)
	file = strings.TrimSpace(file)
	if url == "" {
		url = "http://127.0.0.1:8085/data.json"
	}
	return &WindowsLHMProvider{
		httpClient: &http.Client{Timeout: 2 * time.Second},
		url:        url,
		file:       file,
	}
}

func (p *WindowsLHMProvider) TemperatureC(ctx context.Context) (float64, error) {
	sources := []func(context.Context) ([]byte, error){
		p.readFromURL,
	}
	if p.file != "" {
		sources = append([]func(context.Context) ([]byte, error){p.readFromFile}, sources...)
	}

	var lastErr error
	for _, src := range sources {
		body, err := src(ctx)
		if err != nil {
			lastErr = err
			continue
		}
		temp, err := parseLHMTemperature(body)
		if err == nil {
			return temp, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no LHM source")
	}
	return 0, lastErr
}

func (p *WindowsLHMProvider) FanRPM(ctx context.Context) (int, error) {
	sources := []func(context.Context) ([]byte, error){
		p.readFromURL,
	}
	if p.file != "" {
		sources = append([]func(context.Context) ([]byte, error){p.readFromFile}, sources...)
	}

	var lastErr error
	for _, src := range sources {
		body, err := src(ctx)
		if err != nil {
			lastErr = err
			continue
		}
		rpm, err := parseLHMFanRPM(body)
		if err == nil {
			return rpm, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no LHM source")
	}
	return 0, lastErr
}

func (p *WindowsLHMProvider) readFromURL(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lhm url request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("lhm url status: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lhm url read: %w", err)
	}
	return body, nil
}

func (p *WindowsLHMProvider) readFromFile(_ context.Context) ([]byte, error) {
	body, err := os.ReadFile(p.file)
	if err != nil {
		return nil, fmt.Errorf("lhm file read: %w", err)
	}
	return body, nil
}

func parseLHMTemperature(raw []byte) (float64, error) {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return 0, fmt.Errorf("parse lhm json: %w", err)
	}

	best, ok := findBestTemperature(root)
	if !ok {
		return 0, errors.New("temperature not found in LHM json")
	}
	return best, nil
}

func parseLHMFanRPM(raw []byte) (int, error) {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return 0, fmt.Errorf("parse lhm json: %w", err)
	}

	best, ok := findBestFanRPM(root)
	if !ok {
		return 0, errors.New("fan rpm not found in LHM json")
	}
	return best, nil
}

func findBestTemperature(node any) (float64, bool) {
	temps := make([]float64, 0, 8)
	walkAny(node, func(obj map[string]any) {
		if !looksLikeTempNode(obj) {
			return
		}
		if t, ok := extractTempFromObject(obj); ok {
			temps = append(temps, t)
		}
	})
	if len(temps) == 0 {
		return 0, false
	}

	best := temps[0]
	for _, t := range temps[1:] {
		if t > best {
			best = t
		}
	}
	return best, true
}

func findBestFanRPM(node any) (int, bool) {
	rpms := make([]int, 0, 8)
	walkAny(node, func(obj map[string]any) {
		if !looksLikeFanNode(obj) {
			return
		}
		if rpm, ok := extractFanRPMFromObject(obj); ok {
			rpms = append(rpms, rpm)
		}
	})
	if len(rpms) == 0 {
		return 0, false
	}

	best := rpms[0]
	for _, rpm := range rpms[1:] {
		if rpm > best {
			best = rpm
		}
	}
	return best, true
}

func walkAny(node any, visit func(map[string]any)) {
	switch v := node.(type) {
	case map[string]any:
		visit(v)
		for _, child := range v {
			walkAny(child, visit)
		}
	case []any:
		for _, item := range v {
			walkAny(item, visit)
		}
	}
}

func looksLikeTempNode(obj map[string]any) bool {
	for _, key := range []string{"SensorType", "Type", "Text", "Name"} {
		s := strings.ToLower(toString(obj[key]))
		if strings.Contains(s, "temperature") || strings.Contains(s, "temp") || strings.Contains(s, "°c") {
			return true
		}
	}
	for _, key := range []string{"Value", "Current", "Text"} {
		s := strings.ToLower(toString(obj[key]))
		if strings.Contains(s, "°c") {
			return true
		}
	}
	return false
}

func looksLikeFanNode(obj map[string]any) bool {
	for _, key := range []string{"SensorType", "Type", "Text", "Name"} {
		s := strings.ToLower(toString(obj[key]))
		if strings.Contains(s, "fan") || strings.Contains(s, "rpm") {
			return true
		}
	}
	for _, key := range []string{"Value", "Current", "Text"} {
		s := strings.ToLower(toString(obj[key]))
		if strings.Contains(s, "rpm") {
			return true
		}
	}
	return false
}

func extractTempFromObject(obj map[string]any) (float64, bool) {
	candidates := []string{
		toString(obj["Value"]),
		toString(obj["Current"]),
		toString(obj["Text"]),
	}
	for _, s := range candidates {
		if t, ok := parseTempString(s); ok {
			return t, true
		}
	}
	for _, key := range []string{"Value", "Current"} {
		if n, ok := toFloat(obj[key]); ok {
			if n > -40 && n < 200 {
				return n, true
			}
		}
	}
	return 0, false
}

func extractFanRPMFromObject(obj map[string]any) (int, bool) {
	candidates := []string{
		toString(obj["Value"]),
		toString(obj["Current"]),
		toString(obj["Text"]),
	}
	for _, s := range candidates {
		if rpm, ok := parseFanString(s); ok {
			return rpm, true
		}
	}
	for _, key := range []string{"Value", "Current"} {
		if n, ok := toFloat(obj[key]); ok {
			rpm := int(n)
			if rpm > 0 && rpm < 20000 {
				return rpm, true
			}
		}
	}
	return 0, false
}

func parseTempString(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, false
	}
	if !strings.Contains(s, "c") && !strings.Contains(s, "temp") && !strings.Contains(s, "°") {
		return 0, false
	}
	match := tempRegex.FindString(s)
	if match == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(match, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	if v < -40 || v > 200 {
		return 0, false
	}
	return v, true
}

func parseFanString(s string) (int, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, false
	}
	if !strings.Contains(s, "fan") && !strings.Contains(s, "rpm") {
		return 0, false
	}
	match := fanRegex.FindString(s)
	if match == "" {
		return 0, false
	}
	v, err := strconv.Atoi(match)
	if err != nil || v <= 0 || v > 20000 {
		return 0, false
	}
	return v, true
}

func toString(v any) string {
	s, _ := v.(string)
	return s
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		if err == nil {
			return f, true
		}
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}
