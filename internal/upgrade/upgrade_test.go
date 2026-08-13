package upgrade

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestRunUpgradeAndChecksumRollback(t *testing.T) {
	dir := t.TempDir()
	installed := filepath.Join(dir, "aic")
	if err := os.WriteFile(installed, []byte("#!/bin/sh\nprintf 'ai-instructions 1.0.0\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	candidate := []byte("#!/bin/sh\nprintf 'ai-instructions 2.0.0\\n'\n")
	sum := sha256.Sum256(candidate)
	var bad atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Base(r.URL.Path) == "checksums.txt" {
			if bad.Load() {
				fmt.Fprintln(w, "deadbeef  aic_Test")
			} else {
				fmt.Fprintf(w, "%x  aic_Test\n", sum)
			}
			return
		}
		_, _ = w.Write(candidate)
	}))
	defer server.Close()
	result, err := Run(context.Background(), Options{Installed: installed, BaseURL: server.URL, Version: "latest", Artifact: "aic_Test", Client: server.Client()})
	if err != nil || !result.Changed || result.Version != "2.0.0" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	before, _ := os.ReadFile(installed)
	bad.Store(true)
	if _, err := Run(context.Background(), Options{Installed: installed, BaseURL: server.URL, Artifact: "aic_Test", Client: server.Client()}); err == nil {
		t.Fatal("expected checksum error")
	}
	after, _ := os.ReadFile(installed)
	if string(before) != string(after) {
		t.Fatal("installed changed on failure")
	}
}
