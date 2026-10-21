package publisher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aida64-client/internal/collector"
)

func TestFaceValuesFollowTheFaceContract(t *testing.T) {
	m := collector.Metrics{RAMSpeed: "DDR5-5600", RAMLoad: "45%", CPULoad: "12%", StorageUsage: "1.2TB/2.0TB",
		CPUName: "x‮evil" + strings.Repeat("y", 80)}
	v := FaceValues(m)
	if v["face.1"] != "DDR5-5600" || v["face.4"] != "45%" || v["face.6"] != "12%" || v["face.11"] != "1.2TB/2.0TB" ||
		v["face.13"] != "60%" || v["face.15"] != "2.0TB" || !strings.HasPrefix(v["face.16"], "CPU - xevil") {
		t.Fatalf("keys: %v", v)
	}
	if _, ok := v["face.2"]; ok {
		t.Fatalf("an empty metric is left out: %v", v)
	}
	if strings.ContainsRune(v["face.5"], 0x202e) || len(v["face.5"]) > 64 {
		t.Fatalf("bidi characters go and values fit 64 bytes: %q", v["face.5"])
	}
}

func TestPushSendsTheTokenToTheFaceRoute(t *testing.T) {
	var auth, path string
	var body map[string]map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = w.Write([]byte(`{"success":true,"written":1}`))
	}))
	defer server.Close()
	c := New(server.URL, "aitbase_face_abc", 0)
	if _, err := c.PushMetrics(context.Background(), collector.Metrics{RAMLoad: "45%"}); err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer aitbase_face_abc" || path != ValuesPath || body["values"]["face.4"] != "45%" {
		t.Fatalf("auth=%q path=%q body=%v", auth, path, body)
	}
	if _, err := New(server.URL, "", 0).PushMetrics(context.Background(), collector.Metrics{}); err == nil {
		t.Fatal("no token, no push")
	}
}

func TestDerivedTextsMatchTheOldCard(t *testing.T) {
	percents := map[string]int{"1.2TB/2.0TB": 60, "400GB/2TB": 20, "512 MB / 1 GB": 50, "45%": 45, "3TB/2TB": 100,
		"abc": 0, "1XB/2TB": 0, "1TB/0TB": 0, "N/A": 0}
	for in, want := range percents {
		if got := StoragePercent(in); got != want {
			t.Errorf("StoragePercent(%q) = %d, want %d", in, got, want)
		}
	}
	usage := map[[2]string]string{{"14GB", "18GB"}: "14 / 32 GB", {"14.2GB", "17.8GB"}: "14.2 / 32.0 GB",
		{"12.3", "4"}: "12.3 / 16.3", {"14GB", "18000MB"}: "14GB", {"N/A", "0B"}: "N/A", {"0B", "0B"}: "0 / 0 B"}
	for in, want := range usage {
		if got := RAMUsageText(in[0], in[1]); got != want {
			t.Errorf("RAMUsageText(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
	totals := map[string]string{"1.2TB/ 2.0TB": "2.0TB", "2TB": "2TB", "": "--", "1TB/": "--", "N/A": "A"}
	for in, want := range totals {
		if got := StorageTotalText(in); got != want {
			t.Errorf("StorageTotalText(%q) = %q, want %q", in, got, want)
		}
	}
	if CPUHeaderText("") != "CPU - Unknown CPU" || CPUHeaderText("Ryzen 9") != "CPU - Ryzen 9" {
		t.Errorf("CPU header")
	}
}
