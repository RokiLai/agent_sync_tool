package lock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type ProcessInspector interface{ Alive(int) bool }
type OSProcesses struct{}

func (OSProcesses) Alive(pid int) bool { return syscall.Kill(pid, 0) == nil }

type Options struct {
	PID       int
	Attempts  int
	Interval  time.Duration
	Processes ProcessInspector
}
type Lock struct {
	dir  string
	pid  int
	held bool
}

func Acquire(ctx context.Context, dir string, opts Options) (*Lock, error) {
	if opts.PID == 0 {
		opts.PID = os.Getpid()
	}
	if opts.Attempts == 0 {
		opts.Attempts = 10
	}
	if opts.Interval == 0 {
		opts.Interval = time.Second
	}
	if opts.Processes == nil {
		opts.Processes = OSProcesses{}
	}
	missingPID := false
	for attempt := 0; attempt < opts.Attempts; attempt++ {
		if err := os.Mkdir(dir, 0700); err == nil {
			pidFile := filepath.Join(dir, "pid")
			if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d\n", opts.PID)), 0600); err != nil {
				_ = os.Remove(dir)
				return nil, err
			}
			return &Lock{dir: dir, pid: opts.PID, held: true}, nil
		} else if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		pidPath := filepath.Join(dir, "pid")
		owner, valid := readPID(pidPath)
		if !valid {
			if _, statErr := os.Stat(pidPath); os.IsNotExist(statErr) && !missingPID {
				missingPID = true
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(opts.Interval):
				}
				continue
			}
			_ = os.Remove(pidPath)
			_ = os.Remove(dir)
			missingPID = false
			continue
		}
		missingPID = false
		if !opts.Processes.Alive(owner) {
			_ = os.Remove(pidPath)
			_ = os.Remove(dir)
			continue
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(opts.Interval):
		}
	}
	return nil, errors.New("等待同步锁超时")
}

func (l *Lock) Release() error {
	if l == nil || !l.held {
		return nil
	}
	owner, valid := readPID(filepath.Join(l.dir, "pid"))
	if !valid || owner != l.pid {
		return errors.New("同步锁所有权已丢失")
	}
	if err := os.Remove(filepath.Join(l.dir, "pid")); err != nil {
		return err
	}
	if err := os.Remove(l.dir); err != nil {
		return err
	}
	l.held = false
	return nil
}

func readPID(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	raw := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(raw)
	return pid, err == nil && pid > 0
}
