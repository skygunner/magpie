package gateway

import (
	"regexp"

	"github.com/yetone/magpie/internal/provider"
)

// isGitee is Gitee AI's serverless API (ai.gitee.com, 模力方舟), by its
// preset or its host: its image models (z-image-turbo) are served only on
// the images API, and it takes width and height where OpenAI's images API
// takes size - a size it answers 参数无效 (#1105, GEEKCX).
var isGitee = func(p provider.Provider) bool {
	return p.Preset == "gitee" || provider.HostOf(p.Base(provider.Chat)) == "ai.gitee.com"
}

// whSize reads a size of the images API, "1024x1024".
var whSize = regexp.MustCompile(`^(\d+)x(\d+)$`)
