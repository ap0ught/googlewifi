package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusFallbackURL(t *testing.T) {
	got, err := statusFallbackURL("http://192.168.86.23/api/v1/diagnostic-report")
	if err != nil {
		t.Fatal(err)
	}
	want := "http://192.168.86.23/api/v1/status"
	if got != want {
		t.Fatalf("statusFallbackURL() = %q, want %q", got, want)
	}
}

func TestFetchReportFallsBackToStatusOnNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/diagnostic-report":
			http.NotFound(w, r)
		case "/api/v1/status":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"system":{"groupRole":"leaf"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	body, contentType, source, err := fetchReport(server.URL + "/api/v1/diagnostic-report")
	if err != nil {
		t.Fatal(err)
	}
	if source != server.URL+"/api/v1/status" {
		t.Fatalf("source = %q, want status fallback", source)
	}
	if contentType != "application/json" {
		t.Fatalf("content type = %q, want application/json", contentType)
	}
	if string(body) != `{"system":{"groupRole":"leaf"}}` {
		t.Fatalf("body = %s", body)
	}
}
