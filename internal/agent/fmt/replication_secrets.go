package fmt

import (
	"fmt"

	"github.com/soerenschneider/sc-agent/pkg/api"
)

func PrintSecretReplicationItems(items []api.ReplicationSecretsItem) {
	for i, item := range items {
		fmt.Printf("Replication Item Set %d:\n", i+1)
		PrintSecretReplicationItem(item)
		fmt.Println() // Add a blank line between item sets
	}
}

// PrintSecretReplicationItem prints a single SecretReplicationItem struct.
func PrintSecretReplicationItem(item api.ReplicationSecretsItem) {
	status := "N/A"
	if item.Status != nil {
		status = string(*item.Status)
	}

	fmt.Printf(`  ID          : %s
  Secret Path : %s
  Destination : %s
  Formatter   : %s
  Status      : %s
`,
		item.Id,
		item.SecretPath,
		item.DestUri,
		item.Formatter,
		status,
	)
}
