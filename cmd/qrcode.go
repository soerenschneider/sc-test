package cmd

import (
	"github.com/spf13/cobra"
)

const (
	qrcodeEncode            = "encode"
	qrcodeEncodeInputFile   = "encode-file"
	qrcodeEncodeOutputFile  = "output-file"
	qrcodeDecodeFile        = "file"
	qrcodeDecodePrintBanner = "print-banner"
)

var qrcodeCmd = &cobra.Command{
	Use:     "qrcode",
	Aliases: []string{"qr"},
	Short:   "Encodes or decodes a string to/from a QR code.",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(qrcodeCmd)
}
