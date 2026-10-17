package cmd

import (
	"errors"
	"fmt"
	"os"

	"charm.land/huh/v2"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/pkg"
	"github.com/soerenschneider/sc/pkg/pass"
	"github.com/spf13/cobra"
)

var passShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Interacts with the pass unix password store",
	Run: func(cmd *cobra.Command, args []string) {
		useOtp, _ := pkg.GetBool(cmd, passOtp)
		otpPrefix := pkg.GetString(cmd, passOtpPrefix)
		showData, _ := pkg.GetBool(cmd, passShow)

		// Check if pass is installed
		if err := pass.CheckPassInstalled(); err != nil {
			log.Fatal().Err(err).Msg("pass is available")
		}

		// Get all password entries
		entries, err := pass.GetPassEntries(useOtp, otpPrefix, "")
		if err != nil {
			log.Fatal().Err(err).Msg("could not get pass entries")
		}

		var options []huh.Option[string]
		for _, value := range entries {
			options = append(options, huh.NewOption(value, value))
		}

		var passEntry string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select a secret").
					Options(options...).
					Filtering(true).
					Value(&passEntry),
			),
		)

		if err := form.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				os.Exit(0)
			}
			log.Fatal().Err(err).Msg("could not display picker")
		}

		data, err := pass.GetPassEntry(passEntry, showData, otpPrefix)
		if err != nil {
			log.Fatal().Err(err).Msg("could not get pass entry")
		}

		if showData {
			if err := huh.NewNote().Title(passEntry).Description(data).Run(); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					os.Exit(0)
				}
				log.Fatal().Err(err).Msg("could not display note")
			}
		} else {
			// this will just print the text that the content has been copied to clipboard
			fmt.Println(data)
		}
	},
}

func init() {
	passCmd.AddCommand(passShowCmd)

	passShowCmd.Flags().BoolP(passShow, "s", false, "Show secret data on the output. If set to false (default), the first line is copied to the clipboard.")
	passShowCmd.Flags().BoolP(passOtp, "o", false, "Get an OTP code. The entry picker will only show OTP entries.")
	passShowCmd.Flags().String(passOtpPrefix, "otp/", "Sets the otp path prefix.")
}
