package release

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestArtifactAndURL(t *testing.T) {
	name, err := Artifact("darwin", "amd64")
	if err != nil || name != "aic_Darwin_x86_64" {
		t.Fatalf("%s %v", name, err)
	}
	if _, err := Artifact("windows", "amd64"); err == nil {
		t.Fatal("expected error")
	}
	if got := URL("https://x/releases", "v1", "aic"); got != "https://x/releases/download/v1/aic" {
		t.Fatal(got)
	}
}
func TestDownloadChecksumsVerify(t *testing.T) {
	data := []byte("binary")
	sum := sha256.Sum256(data)
	checks := ParseChecksums([]byte(fmt.Sprintf("%x  aic\n", sum)))
	if err := Verify(data, checks["aic"]); err != nil {
		t.Fatal(err)
	}
	if err := Verify([]byte("bad"), checks["aic"]); err == nil {
		t.Fatal("expected mismatch")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(data) }))
	defer server.Close()
	got, err := Download(context.Background(), server.Client(), server.URL)
	if err != nil || string(got) != "binary" {
		t.Fatalf("%q %v", got, err)
	}
}
