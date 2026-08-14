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

func TestLastCheckedReadWrite(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "runtime")
	if _, err := ReadLastChecked(dir); err == nil {
		t.Fatal("expected error reading non-existent LAST_CHECKED")
	}
	now := time.Unix(1723640000, 0)
	if err := WriteLastChecked(dir, now); err != nil {
		t.Fatalf("WriteLastChecked failed: %v", err)
	}
	got, err := ReadLastChecked(dir)
	if err != nil || !got.Equal(now) {
		t.Fatalf("ReadLastChecked=%v, err=%v, want=%v", got, err, now)
	}
}

func TestSyncAutoTTL(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "runtime")
	downloadCount := 0
	s := Syncer{
		RuntimeDir: dir,
		Download: func(context.Context, string) ([]byte, error) {
			downloadCount++
			return []byte("rules v1\n"), nil
		},
		LockOptions: lock.Options{Attempts: 2, Interval: time.Millisecond, Processes: deadProcesses{}},
	}

	// 1. 首次调用，本地无缓存，即使有 TTL 也必须触发下载
	state, skipped, err := s.SyncAuto(context.Background(), "https://example.test", time.Hour)
	if err != nil || !state.Valid || skipped || downloadCount != 1 {
		t.Fatalf("first auto sync: state=%#v skipped=%v err=%v count=%d", state, skipped, err, downloadCount)
	}

	// 2. 在 TTL 内再次调用，应该跳过下载 (skipped == true, downloadCount 保持 1)
	state, skipped, err = s.SyncAuto(context.Background(), "https://example.test", time.Hour)
	if err != nil || !state.Valid || !skipped || downloadCount != 1 {
		t.Fatalf("second auto sync within TTL: state=%#v skipped=%v err=%v count=%d", state, skipped, err, downloadCount)
	}

	// 3. 将 LAST_CHECKED 改为 2 小时前，模拟 TTL 过期
	twoHoursAgo := time.Now().Add(-2 * time.Hour)
	if err := WriteLastChecked(dir, twoHoursAgo); err != nil {
		t.Fatal(err)
	}
	state, skipped, err = s.SyncAuto(context.Background(), "https://example.test", time.Hour)
	if err != nil || !state.Valid || skipped || downloadCount != 2 {
		t.Fatalf("third auto sync after TTL expiry: state=%#v skipped=%v err=%v count=%d", state, skipped, err, downloadCount)
	}

	// 4. TTL 为 0 时始终触发下载
	state, skipped, err = s.SyncAuto(context.Background(), "https://example.test", 0)
	if err != nil || !state.Valid || skipped || downloadCount != 3 {
		t.Fatalf("zero TTL sync: state=%#v skipped=%v err=%v count=%d", state, skipped, err, downloadCount)
	}
}
