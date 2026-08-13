package runtime

import "github.com/RokiLai/agent_sync_tool/internal/core"

type Candidate = core.Candidate
type Publisher = core.Publisher

func NewCandidate(data []byte) (Candidate, error) { return core.NewCandidate(data) }
