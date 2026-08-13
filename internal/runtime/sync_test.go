package runtime

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/RokiLai/agent_sync_tool/internal/lock"
)

type deadProcesses struct{}

func (deadProcesses) Alive(int) bool { return false }

func TestSyncAndLastKnownGood(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "runtime")
	s := Syncer{RuntimeDir: dir, Download: func(context.Context, string) ([]byte, error) { return []byte("rules\n"), nil }, LockOptions: lock.Options{Attempts: 2, Interval: time.Millisecond, Processes: deadProcesses{}}}
	state, err := s.Sync(context.Background(), "https://example.test/rules", false)
	if err != nil || !state.Valid {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	s.Download = func(context.Context, string) ([]byte, error) { return nil, errors.New("offline") }
	state, err = s.Sync(context.Background(), "https://example.test/rules", false)
	if !errors.Is(err, ErrUsingCache) || !state.Valid {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	if _, err = s.Sync(context.Background(), "https://example.test/rules", true); err == nil || errors.Is(err, ErrUsingCache) {
		t.Fatalf("strict err=%v", err)
	}
}

func TestSyncNoCacheFails(t *testing.T) {
	s := Syncer{RuntimeDir: filepath.Join(t.TempDir(), "runtime"), Download: func(context.Context, string) ([]byte, error) { return nil, errors.New("offline") }, LockOptions: lock.Options{Attempts: 1, Processes: deadProcesses{}}}
	if _, err := s.Sync(context.Background(), "https://example.test/rules", false); err == nil {
		t.Fatal("expected failure")
	}
}

func TestSyncCandidate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "runtime")
	candidate, _ := NewCandidate([]byte("prepared\n"))
	called := false
	s := Syncer{RuntimeDir: dir, Download: func(context.Context, string) ([]byte, error) {
		called = true
		return nil, errors.New("must not download")
	}, LockOptions: lock.Options{Attempts: 1, Processes: deadProcesses{}}}
	state, err := s.SyncCandidate(context.Background(), "https://example.test", true, &candidate)
	if err != nil || !state.Valid || called {
		t.Fatalf("state=%#v called=%v err=%v", state, called, err)
	}
}
