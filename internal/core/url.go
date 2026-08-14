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

func NormalizeURL(raw string) (string, bool, error) {
	if err := ValidateURL(raw); err != nil {
		return "", false, err
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false, err
	}
	if u.User != nil || u.Port() != "" || !strings.EqualFold(u.Hostname(), "github.com") {
		return raw, false, nil
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) < 5 || parts[0] == "" || parts[1] == "" || parts[2] != "blob" || !safeGitHubRef(parts[3]) {
		return raw, false, nil
	}
	normalized := (&url.URL{
		Scheme: "https",
		Host:   "raw.githubusercontent.com",
		Path:   "/" + strings.Join(append([]string{parts[0], parts[1], parts[3]}, parts[4:]...), "/"),
	}).String()
	return normalized, true, nil
}

func safeGitHubRef(ref string) bool {
	if ref == "main" || ref == "master" {
		return true
	}
	if len(ref) != 40 {
		return false
	}
	for _, char := range ref {
		if !strings.ContainsRune("0123456789abcdefABCDEF", char) {
			return false
		}
	}
	return true
}
