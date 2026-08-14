package source

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/RokiLai/agent_sync_tool/internal/core"
)

func ValidateURL(raw string) error {
	return core.ValidateURL(raw)
}

func NormalizeURL(raw string) (string, bool, error) { return core.NormalizeURL(raw) }

func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			return ValidateURL(req.URL.String())
		},
	}
}

func Download(ctx context.Context, client *http.Client, raw string) ([]byte, error) {
	normalized, _, err := NormalizeURL(raw)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, normalized, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP 状态：%s", resp.Status)
	}
	contentType := resp.Header.Get("Content-Type")
	mediaType, _, parseErr := mime.ParseMediaType(contentType)
	if (parseErr == nil && strings.EqualFold(mediaType, "text/html")) || strings.HasPrefix(strings.ToLower(contentType), "text/html;") {
		return nil, errors.New("下载地址返回 HTML 页面，请使用原始 AGENTS.md 文件地址")
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("下载的 AGENTS.md 不存在或为空")
	}
	return data, nil
}
