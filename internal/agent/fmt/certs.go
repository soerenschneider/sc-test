package fmt

import (
	"fmt"
	"strings"

	"github.com/soerenschneider/sc-agent/pkg/api"
)

func agentFormatPostHooks(hooks []api.PostHooks) string {
	var sb strings.Builder
	for _, hook := range hooks {
		_, _ = fmt.Fprintf(&sb, `  Name: %s
  Cmd : %s
`, hook.Name, hook.Cmd)
	}
	return sb.String()
}
