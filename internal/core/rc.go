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
	legacyBegin, legacyEnd := strings.Contains(text, config.LegacyBlockBegin), strings.Contains(text, config.LegacyBlockEnd)
	if legacyBegin != legacyEnd {
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
	text := string(data)
	if strings.Contains(text, config.LegacyBlockBegin) {
		text = removeBlockFromText(text, config.LegacyBlockBegin, config.LegacyBlockEnd)
	}
	if strings.Contains(text, config.BlockBegin) {
		if text != string(data) {
			return managedfs.AtomicWrite(path, []byte(text), 0600)
		}
		return nil
	}
	block := fmt.Sprintf("\n%s\n[ -r \"%s\" ] && . \"%s\"\n%s\n", config.BlockBegin, shellFile, shellFile, config.BlockEnd)
	return managedfs.AtomicWrite(path, append([]byte(text), []byte(block)...), 0600)
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
	if strings.Contains(text, config.BlockBegin) {
		if !strings.Contains(text, config.BlockEnd) {
			return errors.New("Shell 配置受管块不完整")
		}
		text = removeBlockFromText(text, config.BlockBegin, config.BlockEnd)
	}
	if strings.Contains(text, config.LegacyBlockBegin) {
		if !strings.Contains(text, config.LegacyBlockEnd) {
			return errors.New("Shell 配置受管块不完整")
		}
		text = removeBlockFromText(text, config.LegacyBlockBegin, config.LegacyBlockEnd)
	}
	if text == string(data) {
		return nil
	}
	return managedfs.AtomicWrite(path, []byte(text), 0600)
}

func removeBlockFromText(text, beginMarker, endMarker string) string {
	begin := strings.Index(text, beginMarker)
	if begin < 0 {
		return text
	}
	endRel := strings.Index(text[begin:], endMarker)
	if endRel < 0 {
		return text
	}
	end := begin + endRel + len(endMarker)
	if end < len(text) && text[end] == '\n' {
		end++
	}
	return text[:begin] + text[end:]
}
