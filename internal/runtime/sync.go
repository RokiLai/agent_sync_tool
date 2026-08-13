package runtime

import (
	"context"
	"fmt"

	"github.com/RokiLai/agent_sync_tool/internal/lock"
)

type Downloader func(context.Context, string) ([]byte, error)
type Syncer struct {
	RuntimeDir  string
	Download    Downloader
	LockOptions lock.Options
}

func (s Syncer) Sync(ctx context.Context, rawURL string, strict bool) (State, error) {
	return s.SyncCandidate(ctx, rawURL, strict, nil)
}

func (s Syncer) SyncCandidate(ctx context.Context, rawURL string, strict bool, prepared *Candidate) (State, error) {
	l, err := lock.Acquire(ctx, s.RuntimeDir+".lock", s.LockOptions)
	if err != nil {
		return s.cacheOrError(strict, err)
	}
	defer l.Release()
	candidate := Candidate{}
	if prepared != nil {
		candidate = *prepared
	} else {
		data, downloadErr := s.Download(ctx, rawURL)
		if downloadErr != nil {
			return s.cacheOrError(strict, fmt.Errorf("AGENTS.md 下载或校验失败"))
		}
		candidate, err = NewCandidate(data)
		if err != nil {
			return s.cacheOrError(strict, err)
		}
	}
	if err := (Publisher{Dir: s.RuntimeDir}).Publish(candidate); err != nil {
		return s.cacheOrError(strict, err)
	}
	return Inspect(s.RuntimeDir), nil
}

func (s Syncer) cacheOrError(strict bool, err error) (State, error) {
	state := Inspect(s.RuntimeDir)
	if !strict && state.Valid {
		return state, fmt.Errorf("%w；使用最后一次成功部署的有效缓存：%s", ErrUsingCache, state.Revision)
	}
	if strict {
		return state, fmt.Errorf("%w；当前操作要求成功获取新来源", err)
	}
	return state, fmt.Errorf("%w，且没有可用的 runtime 缓存", err)
}

var ErrUsingCache = fmt.Errorf("using last-known-good")
