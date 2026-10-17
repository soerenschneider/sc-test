package cmd

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/k8s"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var k8sRenovateRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Interact with Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatal().Msg("No job name specified")
		}
		name := args[0]
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
			if cj.Name == name {
				running, err := k8s.IsManualJobRunning(ctx, client, &cj)
				if err != nil {
					log.Fatal().Err(err).Msgf("Checking if job %s is running", cj.Name)
				}
				if running {
					fmt.Println("A manual job is already running, skipping.")
					return
				}

				if err := k8s.CreateJobAndAttach(ctx, client, restConfig, &cj); err != nil {
					log.Fatal().Err(err).Msg("failed to create job")
				}
				return
			}
		}

	},
}

func init() {
	k8sRenovateCmd.AddCommand(k8sRenovateRunCmd)
}
