package cmd

import (
	"fmt"
	"image"
	"os"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	"github.com/liyue201/goqr"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
)

var qrcodeDecodeCmd = &cobra.Command{
	Use:     "decode",
	Aliases: []string{"dec"},
	Short:   "Decodes an image containing QR code to text",
	Run: func(cmd *cobra.Command, args []string) {
		file, _ := os.Open(pkg.GetString(cmd, qrcodeDecodeFile))
		defer func() {
			_ = file.Close()
		}()

		img, _, err := image.Decode(file)
		if err != nil {
			log.Fatal().Err(err).Msg("could not decode image")
		}

		qrcodes, err := goqr.Recognize(img)
		if err != nil {
			log.Fatal().Err(err).Msg("could not recognize QR data")
		}

		printBanner, _ := pkg.GetBool(cmd, qrcodeDecodePrintBanner)
		for _, qr := range qrcodes {
			if printBanner {
				fmt.Println("--- Decoded text start ---")
			}

			fmt.Println(strings.TrimSpace(string(qr.Payload)))

			if printBanner {
				fmt.Println("--- Decoded text end ---")
			}
		}
	},
}

func init() {
	qrcodeCmd.AddCommand(qrcodeDecodeCmd)

	qrcodeDecodeCmd.Flags().BoolP(qrcodeDecodePrintBanner, "p", true, "Print pre- and postfix banners")
	qrcodeDecodeCmd.Flags().StringP(qrcodeDecodeFile, "f", "", "Decodes the QR code of the given file")
	_ = qrcodeDecodeCmd.MarkFlagRequired(qrcodeDecodeFile)
}
