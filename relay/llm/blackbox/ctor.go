package blackbox

import (
	"chatgpt-adapter/core/gin/inter"
	"chatgpt-adapter/core/gin/response"
	"github.com/iocgo/sdk/env"

	_ "github.com/iocgo/sdk"
)

// @Inject(name = "blackbox-adapter")
func New(env *env.Environment, holder *response.ContentHolder) inter.Adapter {
	return &api{env: env, holder: holder}
}
