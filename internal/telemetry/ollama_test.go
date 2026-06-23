package telemetry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnalyzeJobIncludesOllamaErrorDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Fatalf("request path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"model 'llama3' not found"}`))
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "llama3")
	client.HTTP = server.Client()

	_, err := client.AnalyzeJob("Build reliable systems.")
	if err == nil {
		t.Fatal("AnalyzeJob returned nil error")
	}
	if !strings.Contains(err.Error(), "model 'llama3' not found") {
		t.Fatalf("error = %q, want Ollama response detail", err)
	}
}
