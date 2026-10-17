package cmd

import (
	"context"
	"fmt"

	"github.com/soerenschneider/sc/internal/k8s"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var k8sRenovateListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "Interact with Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		namespace := pkg.MustGetString(cmd, k8sNamespace)

		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

		restConfig, err := kubeConfig.ClientConfig()
		if err != nil {
			panic(fmt.Errorf("building kubeconfig: %w", err))
		}

		client, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			panic(fmt.Errorf("creating kubernetes client: %w", err))
		}

		ctx := context.Background()

		// List all CronJobs with backup=yes across every namespace
		cronJobs, err := k8s.ListBackupCronJobs(ctx, client, namespace)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Found %d backup CronJob(s):\n", len(cronJobs))

		for _, cj := range cronJobs {
			fmt.Printf("  %s/%s\n", cj.Namespace, cj.Name)
		}
	},
}

func init() {
	k8sRenovateCmd.AddCommand(k8sRenovateListCmd)
}
