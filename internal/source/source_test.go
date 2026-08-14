package source

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestDownloadNormalizesGitHubBlobURL(t *testing.T) {
	var requested string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requested = req.URL.String()
		return response(http.StatusOK, "text/plain", "rules\n"), nil
	})}
	data, err := Download(context.Background(), client, "https://github.com/RokiLai/agents/blob/main/AGENTS.md?plain=1")
	if err != nil || string(data) != "rules\n" || requested != "https://raw.githubusercontent.com/RokiLai/agents/main/AGENTS.md" {
		t.Fatalf("requested=%q data=%q err=%v", requested, data, err)
	}
}

func TestDownloadRejectsHTML(t *testing.T) {
	for _, contentType := range []string{"text/html", "text/html; charset=utf-8"} {
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return response(http.StatusOK, contentType, "<html>rules</html>"), nil
		})}
		if _, err := Download(context.Background(), client, "https://example.test/AGENTS.md"); err == nil {
			t.Fatalf("accepted %s", contentType)
		}
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func response(status int, contentType, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{contentType}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
