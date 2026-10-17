package cmd

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/cmd/deps"
	"github.com/soerenschneider/sc/internal"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

const (
	commandName = "sc"

	rootCmdFlagsVerbose     = "verbose"
	rootCmdFlagsProfile     = "profile"
	rootCmdFlagsNoTelemetry = "no-telemetry"

	profileEnvKey      = "SC_PROFILE"
	defaultProfileName = "default"
)

var (
	checkUpdateWaitGroup = &sync.WaitGroup{}
	profile              string
)

func getCommandName(cmd *cobra.Command) string {
	commandPath := cmd.CommandPath()
	commandPath = strings.ReplaceAll(strings.TrimSpace(strings.Replace(commandPath, commandName, "", 1)), " ", "-")
	return commandPath
}

var rootCmd = &cobra.Command{
	Use:               commandName,
	Short:             "Universal Command Line Interface for soeren.cloud",
	DisableAutoGenTag: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		setupLogger()

		if profile == "" {
			var found bool
			profile, found = os.LookupEnv(profileEnvKey)
			if found {
				log.Info().Msgf("Using profile %q defined by env var", profile)
			}
		}

		if profile != "" {
			commandPath := getCommandName(cmd)
			if err := internal.ApplyFlags(commandPath, cmd, profile); err != nil {
				log.Warn().Err(err).Msgf("could not apply flags for profile %q", profile)
			}
		}

		verbose, _ := cmd.Flags().GetBool(rootCmdFlagsVerbose)
		setLogLevel(verbose)

		conditionallyLogLatestReleaseInfo(cmd)

		return nil
	},
}

func Execute(ctx context.Context) {
	defer checkUpdateWaitGroup.Wait()

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&profile, rootCmdFlagsProfile, "", "Profile to use")
	rootCmd.PersistentFlags().BoolP(rootCmdFlagsVerbose, "v", false, "Print debug logs")
	rootCmd.PersistentFlags().Bool(rootCmdFlagsNoTelemetry, false, "Do not perform check for updated version")
}

func initConfig() {
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func setupLogger() {
	//#nosec:G115
	if term.IsTerminal(int(os.Stdout.Fd())) {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out: os.Stderr,
			PartsExclude: []string{
				zerolog.TimestampFieldName,
			},
		})
	}
}

func setLogLevel(debug bool) {
	level := zerolog.InfoLevel
	if debug {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)
}

func conditionallyLogLatestReleaseInfo(cmd *cobra.Command) {
	disableTelemetry := isDisableTelemetry(cmd)
	//nolint G404: no cryptographic randomness required
	if disableTelemetry || rand.Float32() > 0.2 || !strings.HasPrefix(internal.BuildVersion, "v") {
		log.Debug().Str("local_version", internal.BuildVersion).Msg("not performing check for update")
		return
	}

	checkUpdateWaitGroup.Add(1)
	go func() {
		defer checkUpdateWaitGroup.Done()

		httpClient := deps.GetHttpClient()
		releaseNotifier, err := internal.NewReleaseNotifier(httpClient, internal.BuildVersion)
		if err != nil {
			log.Warn().Msg("could not build release notifier")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		releaseNotifier.CheckRelease(ctx)
	}()
}

func isDisableTelemetry(cmd *cobra.Command) bool {
	if cmd.Use == "version" || cmd.Use == "help" || cmd.Use == "docs" || cmd == updateCmd {
		return true
	}

	disableTelemetry, err := cmd.Flags().GetBool(rootCmdFlagsNoTelemetry)
	pkg.DieOnErr(err, "could not get flag")
	return disableTelemetry
}
