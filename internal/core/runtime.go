package core

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/RokiLai/agent_sync_tool/internal/managedfs"
)

type State struct {
	Valid    bool
	Revision string
}

type Candidate struct {
	Data     []byte
	Revision string
}

func Revision(data []byte) string {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d%c", len(data), byte(0))
	_, _ = h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func Size(data []byte) string { return strconv.Itoa(len(data)) }

func NewCandidate(data []byte) (Candidate, error) {
	if len(data) == 0 {
		return Candidate{}, errors.New("AGENTS.md 内容为空")
	}
	copyData := append([]byte(nil), data...)
	return Candidate{Data: copyData, Revision: Revision(copyData)}, nil
}

func InspectRuntime(dir string) State {
	agents := filepath.Join(dir, "AGENTS.md")
	revision := filepath.Join(dir, "REVISION")
	if !nonEmpty(agents) || !nonEmpty(revision) {
		return State{Revision: "none"}
	}
	current := filepath.Join(dir, "current")
	if info, err := os.Lstat(current); err == nil && info.Mode()&os.ModeSymlink == 0 {
		return State{Revision: "none"}
	} else if os.IsNotExist(err) && (isSymlink(agents) || isSymlink(revision)) {
		return State{Revision: "none"}
	}
	data, err := os.ReadFile(revision)
	if err != nil {
		return State{Revision: "none"}
	}
	value := strings.TrimSuffix(string(data), "\n")
	if value == "" || strings.Contains(value, "\n") {
		return State{Revision: "none"}
	}
	return State{Valid: true, Revision: value}
}

type Publisher struct{ Dir string }

func (p Publisher) Publish(candidate Candidate) error {
	versions := filepath.Join(p.Dir, "versions")
	if err := os.MkdirAll(versions, 0755); err != nil {
		return err
	}
	if err := p.migrateLegacy(); err != nil {
		return fmt.Errorf("无法迁移旧 runtime 布局: %w", err)
	}
	versionDir := filepath.Join(versions, candidate.Revision)
	if info, err := os.Stat(versionDir); err == nil && info.IsDir() {
		if err := compareVersion(versionDir, candidate); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	} else {
		temp, err := os.MkdirTemp(versions, "."+candidate.Revision+".tmp.*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(temp)
		if err := os.WriteFile(filepath.Join(temp, "AGENTS.md"), candidate.Data, 0444); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(temp, "REVISION"), []byte(candidate.Revision+"\n"), 0444); err != nil {
			return err
		}
		if err := os.Chmod(filepath.Join(temp, "AGENTS.md"), 0444); err != nil {
			return err
		}
		if err := os.Chmod(filepath.Join(temp, "REVISION"), 0444); err != nil {
			return err
		}
		if err := os.Rename(temp, versionDir); err != nil {
			return err
		}
	}
	if err := managedfs.EnsureSymlink(filepath.Join(p.Dir, "AGENTS.md"), "current/AGENTS.md"); err != nil {
		return err
	}
	if err := managedfs.EnsureSymlink(filepath.Join(p.Dir, "REVISION"), "current/REVISION"); err != nil {
		return err
	}
	return managedfs.AtomicSymlink(filepath.Join(p.Dir, "current"), filepath.Join("versions", candidate.Revision))
}

func (p Publisher) migrateLegacy() error {
	current := filepath.Join(p.Dir, "current")
	if _, err := os.Lstat(current); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	agents, revision := filepath.Join(p.Dir, "AGENTS.md"), filepath.Join(p.Dir, "REVISION")
	if isSymlink(agents) || isSymlink(revision) {
		return errors.New("legacy runtime 链接不完整")
	}
	if !nonEmpty(agents) || !nonEmpty(revision) {
		return nil
	}
	data, err := os.ReadFile(agents)
	if err != nil {
		return err
	}
	revData, err := os.ReadFile(revision)
	if err != nil {
		return err
	}
	rev := strings.TrimSpace(string(revData))
	if rev == "" || strings.ContainsAny(rev, " \t\r\n") {
		return errors.New("legacy REVISION 无效")
	}
	candidate := Candidate{Data: data, Revision: rev}
	versionDir := filepath.Join(p.Dir, "versions", rev)
	if err := os.MkdirAll(filepath.Dir(versionDir), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(versionDir, 0755); err == nil {
		if err := os.WriteFile(filepath.Join(versionDir, "AGENTS.md"), data, 0444); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(versionDir, "REVISION"), []byte(rev+"\n"), 0444); err != nil {
			return err
		}
		if err := os.Chmod(filepath.Join(versionDir, "AGENTS.md"), 0444); err != nil {
			return err
		}
		if err := os.Chmod(filepath.Join(versionDir, "REVISION"), 0444); err != nil {
			return err
		}
	} else if !os.IsExist(err) {
		return err
	}
	if err := compareVersion(versionDir, candidate); err != nil {
		return err
	}
	if err := managedfs.AtomicSymlink(current, filepath.Join("versions", rev)); err != nil {
		return err
	}
	if err := os.Remove(agents); err != nil {
		return err
	}
	return os.Remove(revision)
}

func compareVersion(dir string, candidate Candidate) error {
	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		return err
	}
	rev, err := os.ReadFile(filepath.Join(dir, "REVISION"))
	if err != nil {
		return err
	}
	if !bytes.Equal(data, candidate.Data) || string(rev) != candidate.Revision+"\n" {
		return errors.New("已有 runtime 版本目录内容不一致")
	}
	return nil
}

func nonEmpty(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}
