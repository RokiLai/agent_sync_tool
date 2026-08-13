package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateURL(t *testing.T) {
	for _, raw := range []string{"", "file:///tmp/a", "https://", "https://ok.test/a\nhttps://bad.test/b", "https://ok.test/a\r"} {
		if ValidateURL(raw) == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	for _, raw := range []string{"http://example.test/a", "https://example.test/a"} {
		if err := ValidateURL(raw); err != nil {
			t.Errorf("rejected %q: %v", raw, err)
		}
	}
}

func TestDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("rules\n")) }))
	defer server.Close()
	data, err := Download(context.Background(), NewHTTPClient(), server.URL)
	if err != nil || string(data) != "rules\n" {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestDownloadRejectsEmptyAndStatus(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
		_, err := Download(context.Background(), NewHTTPClient(), server.URL)
		server.Close()
		if err == nil {
			t.Fatalf("expected error for status %d", status)
		}
	}
}

func TestClientRejectsRedirectToFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "file:///tmp/rules", http.StatusFound)
	}))
	defer server.Close()
	if _, err := Download(context.Background(), NewHTTPClient(), server.URL); err == nil {
		t.Fatal("expected redirect error")
	}
}

func TestDownloadHonorsCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Download(ctx, server.Client(), server.URL); err == nil {
		t.Fatal("expected cancellation")
	}
}
