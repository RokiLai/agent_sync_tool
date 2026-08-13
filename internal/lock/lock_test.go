package lock

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type processes map[int]bool

func (p processes) Alive(pid int) bool { return p[pid] }

func TestAcquireReleaseAndOwnership(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "runtime.lock")
	l, err := Acquire(context.Background(), dir, Options{PID: 42, Attempts: 1, Processes: processes{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("lock remains: %v", err)
	}
}

func TestAcquireCleansStaleAndMalformed(t *testing.T) {
	for _, pid := range []string{"bad\n", "99\n"} {
		dir := filepath.Join(t.TempDir(), "runtime.lock")
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "pid"), []byte(pid), 0600); err != nil {
			t.Fatal(err)
		}
		l, err := Acquire(context.Background(), dir, Options{PID: 42, Attempts: 2, Interval: time.Millisecond, Processes: processes{99: false}})
		if err != nil {
			t.Fatal(err)
		}
		if err := l.Release(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAcquireTimeoutAndLostOwnership(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "runtime.lock")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pid"), []byte("99\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(context.Background(), dir, Options{PID: 42, Attempts: 1, Interval: time.Millisecond, Processes: processes{99: true}}); err == nil {
		t.Fatal("expected timeout")
	}
	if err := os.WriteFile(filepath.Join(dir, "pid"), []byte("42\n"), 0600); err != nil {
		t.Fatal(err)
	}
	l := &Lock{dir: dir, pid: 42, held: true}
	if err := os.WriteFile(filepath.Join(dir, "pid"), []byte("43\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := l.Release(); err == nil {
		t.Fatal("expected ownership error")
	}
}

func TestAcquireContextCancellation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "runtime.lock")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pid"), []byte("99\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Acquire(ctx, dir, Options{PID: 42, Attempts: 10, Interval: time.Second, Processes: processes{99: true}}); err == nil {
		t.Fatal("expected cancellation")
	}
}

func TestReleaseIsIdempotent(t *testing.T) {
	var l *Lock
	if err := l.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestAcquireCleansAbandonedDirectoryWithoutPID(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "runtime.lock")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	l, err := Acquire(context.Background(), dir, Options{PID: 42, Attempts: 3, Interval: time.Millisecond, Processes: processes{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Release(); err != nil {
		t.Fatal(err)
	}
}
