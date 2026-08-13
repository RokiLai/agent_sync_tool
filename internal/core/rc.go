package core

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/RokiLai/agent_sync_tool/internal/config"
	"github.com/RokiLai/agent_sync_tool/internal/managedfs"
)

func RCPath(home, shell string) string {
	switch shell {
	case "zsh":
		return home + "/.zshrc"
	case "bash":
		return home + "/.bashrc"
	default:
		return ""
	}
}

func ValidateRC(path string) error {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	text := string(data)
	begin, end := strings.Contains(text, config.BlockBegin), strings.Contains(text, config.BlockEnd)
	if begin != end {
		return errors.New("Shell 配置存在不完整的受管块")
	}
	return nil
}

func InstallRC(path, shellFile string) error {
	if path == "" {
		return nil
	}
	if err := ValidateRC(path); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		data = nil
	} else if err != nil {
		return err
	}
	if strings.Contains(string(data), config.BlockBegin) {
		return nil
	}
	block := fmt.Sprintf("\n%s\n[ -r \"%s\" ] && . \"%s\"\n%s\n", config.BlockBegin, shellFile, shellFile, config.BlockEnd)
	return managedfs.AtomicWrite(path, append(data, []byte(block)...), 0600)
}

func RemoveRC(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	text := string(data)
	begin := strings.Index(text, config.BlockBegin)
	if begin < 0 {
		return nil
	}
	endRel := strings.Index(text[begin:], config.BlockEnd)
	if endRel < 0 {
		return errors.New("Shell 配置受管块不完整")
	}
	end := begin + endRel + len(config.BlockEnd)
	if end < len(text) && text[end] == '\n' {
		end++
	}
	return managedfs.AtomicWrite(path, []byte(text[:begin]+text[end:]), 0600)
}
