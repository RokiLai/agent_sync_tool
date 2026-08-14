package upgrade

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestCheckFindsReleaseVersionWithoutDownloadingArtifact(t *testing.T) {
	dir := t.TempDir()
	installed := filepath.Join(dir, "aic")
	if err := os.WriteFile(installed, []byte("old"), 0700); err != nil {
		t.Fatal(err)
	}
	var artifactRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest/download/checksums.txt":
			http.Redirect(w, r, "/download/v3.2.0/checksums.txt", http.StatusFound)
		case "/download/v3.2.0/checksums.txt":
			fmt.Fprintln(w, "abc  aic_Test")
		default:
			artifactRequests.Add(1)
			http.Error(w, "unexpected", http.StatusNotFound)
		}
	}))
	defer server.Close()
	plan, err := Check(context.Background(), Options{Installed: installed, BaseURL: server.URL, CurrentVersion: "3.1.0", Artifact: "aic_Test", Client: server.Client()})
	if err != nil || plan.CurrentVersion != "3.1.0" || plan.TargetVersion != "3.2.0" || artifactRequests.Load() != 0 {
		t.Fatalf("plan=%#v artifactRequests=%d err=%v", plan, artifactRequests.Load(), err)
	}
}

func TestApplyReportsProgressAndRejectsVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	installed := filepath.Join(dir, "aic")
	if err := os.WriteFile(installed, []byte("#!/bin/sh\nprintf 'ai-instructions 1.0.0\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	candidate := []byte("#!/bin/sh\nprintf 'ai-instructions 2.0.0\\n'\n")
	sum := sha256.Sum256(candidate)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(candidate)))
		_, _ = w.Write(candidate)
	}))
	defer server.Close()
	var events []Progress
	plan := Plan{CurrentVersion: "1.0.0", TargetVersion: "3.0.0", Artifact: "aic_Test", checksums: []byte(fmt.Sprintf("%x  aic_Test\n", sum))}
	_, err := Apply(context.Background(), Options{Installed: installed, BaseURL: server.URL, Version: "latest", Artifact: "aic_Test", Client: server.Client(), Progress: func(p Progress) { events = append(events, p) }}, plan)
	if err == nil || !strings.Contains(err.Error(), "候选版本不匹配") {
		t.Fatalf("events=%#v err=%v", events, err)
	}
	if len(events) == 0 || events[len(events)-1].Stage != "candidate" {
		t.Fatalf("missing progress events: %#v", events)
	}
	got, _ := os.ReadFile(installed)
	if string(got) != "#!/bin/sh\nprintf 'ai-instructions 1.0.0\\n'\n" {
		t.Fatal("installed binary changed on mismatch")
	}
}
