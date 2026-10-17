package cmd

import (
	"context"
	"runtime"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/cmd/deps"
	"github.com/soerenschneider/sc/internal"
	"github.com/soerenschneider/sc/internal/selfupdate"
	"github.com/spf13/cobra"
)

const (
	updateCmdFlagCheck = "check"
	updateCmdFlagForce = "force"

	updateDefaultTimeout = 5 * time.Minute
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"self-update", "upgrade"},
	Short:   "Update sc to the latest release",
	Long: `Check ` + selfupdate.Repository + ` for a newer release and replace the
running binary with it.

The binary is downloaded next to the current one, verified against the
release's sha256 checksum, whose signature (signify, cosign or cosign keyless)
is checked against the signers built into sc, and then atomically moved into
place, so the directory containing the binary must be writable.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		checkOnly, _ := cmd.Flags().GetBool(updateCmdFlagCheck)
		force, _ := cmd.Flags().GetBool(updateCmdFlagForce)

		ctx, cancel := context.WithTimeout(context.Background(), updateDefaultTimeout)
		defer cancel()

		client := deps.GetHttpClient()
		release, err := selfupdate.LatestRelease(ctx, client)
		if err != nil {
			log.Fatal().Err(err).Msg("could not fetch latest release")
		}

		newer, err := selfupdate.IsNewer(internal.BuildVersion, release.TagName)
		if err != nil && !force {
			log.Fatal().Err(err).Msg("could not compare versions, use --force to update anyway")
		}
		if !newer && !force {
			log.Info().Msgf("Already running the latest version %s", internal.BuildVersion)
			return
		}
		log.Info().Msgf("Update available: %s -> %s", internal.BuildVersion, release.TagName)
		if checkOnly {
			return
		}

		if _, err := selfupdate.AssetName(runtime.GOOS, runtime.GOARCH); err != nil {
			log.Fatal().Err(err).Msg("can not update")
		}

		exe, err := selfupdate.Executable()
		if err != nil {
			log.Fatal().Err(err).Msg("could not determine path of current binary")
		}
		if err := selfupdate.CheckWritable(exe); err != nil {
			log.Fatal().Err(err).Msgf("can not replace %s, re-run with sufficient permissions", exe)
		}

		log.Info().Msgf("Downloading %s...", release.TagName)
		if err := selfupdate.Update(ctx, client, release, exe, selfupdate.DefaultVerifiers()); err != nil {
			log.Fatal().Err(err).Msg("update failed")
		}

		log.Info().Msgf("Updated %s to %s", exe, release.TagName)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().BoolP(updateCmdFlagCheck, "c", false, "Only check whether an update is available")
	updateCmd.Flags().BoolP(updateCmdFlagForce, "f", false, "Install the latest release even if it is not newer than the running version")
}
