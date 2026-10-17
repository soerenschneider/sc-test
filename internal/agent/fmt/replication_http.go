package fmt

import (
	"fmt"

	"github.com/soerenschneider/sc-agent/pkg/api"
)

func PrintReplicationHttpItemsList(items []api.ReplicationHttpItem) {
	for i, item := range items {
		fmt.Printf("HTTP Replication Item Set %d:\n", i+1)
		PrintReplicationHttpItem(item)
		fmt.Println()
	}
}

func PrintReplicationHttpItem(item api.ReplicationHttpItem) {
	fileValidation := "N/A"

	if item.FileValidation != nil {
		fileValidation = fmt.Sprintf("%v", item.FileValidation)
	}

	fmt.Printf(`  ID                  : %s
  Source              : %s
  Destinations        : %v
  File validation     : %v
  Status              : %s
`,
		item.Id,
		item.Source,
		item.DestUris,
		fileValidation,
		item.Status,
	)

	if len(item.PostHooks) > 0 {
		fmt.Println("  Post Hooks:")
		for _, hook := range item.PostHooks {
			fmt.Printf("    - Name: %s, Cmd: %s\n", hook.Name, hook.Cmd)
		}
	} else {
		fmt.Println("  Post Hooks        : N/A")
	}
}
