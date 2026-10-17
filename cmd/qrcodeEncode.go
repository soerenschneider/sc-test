package cmd

import (
	"os"

	"charm.land/huh/v2"
	"github.com/mdp/qrterminal/v3"
	"github.com/rs/zerolog/log"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

var qrcodeEncodeCmd = &cobra.Command{
	Use:     "encode",
	Aliases: []string{"enc"},
	Short:   "Encode a string or file to a QR code",
	Run: func(cmd *cobra.Command, args []string) {
		encodeString := pkg.GetString(cmd, qrcodeEncode)
		encodeFile := pkg.GetString(cmd, qrcodeEncodeInputFile)

		if encodeString != "" && encodeFile != "" {
			log.Fatal().Msg("only one of --encode or --encode-file can be specified")
		}

		if encodeFile != "" {
			data, err := os.ReadFile(encodeFile)
			if err != nil {
				log.Fatal().Err(err).Msg("could not read file")
			}
			encodeString = string(data)
		}

		var fields []huh.Field
		if encodeString == "" {
			fields = append(fields, huh.NewInput().Title("Text to encode").Value(&encodeString).Validate(huh.ValidateNotEmpty()))
		}

		if len(fields) > 0 {
			if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
				log.Fatal().Err(err).Msg("could not get info from user")
			}
		}

		qrcodeOutputFile := pkg.GetString(cmd, qrcodeEncodeOutputFile)
		if qrcodeOutputFile != "" {
			if err := qrcode.WriteFile(encodeString, qrcode.Medium, 256, qrcodeOutputFile); err != nil {
				log.Fatal().Err(err).Msg("could not write qr code to file")
			}
		} else {
			qrterminal.Generate(encodeString, qrterminal.L, os.Stdout)
		}
	},
}

func init() {
	qrcodeCmd.AddCommand(qrcodeEncodeCmd)

	qrcodeEncodeCmd.Flags().StringP(qrcodeEncode, "e", "", "Encode string")
	qrcodeEncodeCmd.Flags().StringP(qrcodeEncodeInputFile, "f", "", "Encode the content of the given file")
	qrcodeEncodeCmd.Flags().StringP(qrcodeEncodeOutputFile, "o", "", "Write the QR code to the given file")
}
