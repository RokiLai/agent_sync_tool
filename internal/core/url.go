package core

import (
	"errors"
	"net/url"
	"strings"
)

func ValidateURL(raw string) error {
	if raw == "" || strings.ContainsAny(raw, "\r\n") {
		return errors.New("AGENTS.md 来源必须是 HTTP(S) 文件链接")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("AGENTS.md 来源必须是 HTTP(S) 文件链接")
	}
	return nil
}
