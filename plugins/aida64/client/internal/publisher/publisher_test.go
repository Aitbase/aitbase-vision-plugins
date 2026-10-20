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
		StorageUsedPct: "60", CPUName: "x‮evil" + strings.Repeat("y", 80)}
	v := FaceValues(m)
	if v["face.1"] != "DDR5-5600" || v["face.4"] != "45%" || v["face.6"] != "12%" || v["face.11"] != "1.2TB/2.0TB" ||
		v["face.13"] != "60" {
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
