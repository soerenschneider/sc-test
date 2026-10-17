package pkg

import (
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func MustGetString(cmd *cobra.Command, name string) string {
	val, err := cmd.Flags().GetString(name)
	if err != nil {
		log.Fatal().Err(err).Msg("could not get flag")
	}
	return val
}

func GetStringArray(cmd *cobra.Command, name string) []string {
	val, err := cmd.Flags().GetStringArray(name)
	if err != nil {
		log.Fatal().Err(err).Msg("could not get flag")
	}
	return val
}

func GetInt64Array(cmd *cobra.Command, name string) []int64 {
	val, _ := cmd.Flags().GetInt64Slice(name)
	return val
}

func GetBool(cmd *cobra.Command, name string) (bool, bool) {
	val, _ := cmd.Flags().GetBool(name)
	isChanged := cmd.Flags().Changed(name)
	return val, isChanged
}

func MustGetBool(cmd *cobra.Command, name string) bool {
	val, err := cmd.Flags().GetBool(name)
	if err != nil {
		log.Fatal().Err(err).Msg("could not get flag")
	}
	return val
}

func GetString(cmd *cobra.Command, name string) string {
	val, _ := cmd.Flags().GetString(name)
	return val
}

func GetDuration(cmd *cobra.Command, name string) time.Duration {
	val, _ := cmd.Flags().GetDuration(name)
	return val
}

func GetInt(cmd *cobra.Command, name string) int {
	val, _ := cmd.Flags().GetInt(name)
	return val
}

func GetFloat32(cmd *cobra.Command, name string) float32 {
	val, _ := cmd.Flags().GetFloat32(name)
	return val
}

func GetExpandedFile(filename string) string {
	if strings.HasPrefix(filename, "~/") {
		usr, err := user.Current()
		if err != nil {
			return ""
		}
		return filepath.Join(usr.HomeDir, filename[2:])
	}

	if strings.HasPrefix(filename, "$HOME/") {
		usr, err := user.Current()
		if err != nil {
			return ""
		}
		return filepath.Join(usr.HomeDir, filename[6:])
	}

	return filename
}

func DieOnErr(err error, msg string) {
	if err == nil {
		return
	}

	log.Fatal().Err(err).Msg(msg)
}
