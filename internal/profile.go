package internal

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type Profile map[string]map[string]map[string]any // profile -> command -> flags

var profileData Profile

func Load(path string) error {
	data, err := os.ReadFile(pkg.GetExpandedFile(path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// file does not exist, ignore it
			return nil
		}
		return err
	}

	if err := yaml.Unmarshal(data, &profileData); err != nil {
		return err
	}

	return nil
}

func getCommandHierarchy(cmd string) []string {
	elements := strings.Split(cmd, "-")
	ret := make([]string, len(elements))

	// Build from longest to shortest by removing suffixes
	current := cmd
	for i := 0; i < len(elements); i++ {
		ret[i] = current

		// Remove the last element and its separator for next iteration
		if i < len(elements)-1 {
			lastIdx := strings.LastIndex(current, "-")
			if lastIdx != -1 {
				current = current[:lastIdx]
			}
		}
	}

	return ret
}

func Get(cmdName, profileName, key string) (string, bool) {
	if cmd, ok := profileData[profileName]; ok {
		commandHierarchy := getCommandHierarchy(cmdName)
		for _, c := range commandHierarchy {
			if section, ok := cmd[c]; ok {
				switch v := section[key].(type) {
				case string:
					return v, true
				case bool:
					return strconv.FormatBool(v), true
				case int:
					return strconv.Itoa(v), true
				case float64:
					return strconv.FormatFloat(v, 'f', -1, 64), true
				case []interface{}:
					parts := []string{}
					for _, item := range v {
						parts = append(parts, fmt.Sprintf("%v", item))
					}
					return strings.Join(parts, ","), true
				default:
					return "", false
				}
			}
		}
	}
	return "", false
}

var ErrProfileNotFound = errors.New("profile not found")

func ApplyFlags(cmdName string, cmd *cobra.Command, profileName string) error {
	_, found := profileData[profileName]
	if !found {
		return ErrProfileNotFound
	}

	flagNames := getAllFlagNames(cmd)

	commandHierarchy := getCommandHierarchy(cmdName)
	var cmdNameFound string
	for _, c := range commandHierarchy {
		if cmd, ok := profileData[profileName]; ok {
			if _, ok := cmd[c]; ok {
				cmdNameFound = c
				log.Info().Msgf("Applying settings from parent command %q instead of %q", c, cmdName)
			}
		}
	}

	for _, key := range flagNames {
		val, ok := Get(cmdNameFound, profileName, key)
		if !ok {
			continue
		}

		flag := cmd.Flags().Lookup(key)
		if flag != nil && !cmd.Flags().Changed(key) && flag.Value.String() == flag.DefValue {
			if err := cmd.Flags().Set(key, val); err != nil {
				log.Warn().Err(err).Msg("could not set flag")
			}
			continue
		}

		pFlag := cmd.PersistentFlags().Lookup(key)
		if pFlag != nil && !cmd.PersistentFlags().Changed(key) && pFlag.Value.String() == pFlag.DefValue {
			if err := cmd.PersistentFlags().Set(key, val); err != nil {
				log.Warn().Err(err).Msg("could not set persistent flag")
			}
			continue
		}
	}

	return nil
}

// getAllFlagNames returns all local + persistent flag names for a command
func getAllFlagNames(cmd *cobra.Command) []string {
	flagNames := []string{}

	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flagNames = append(flagNames, flag.Name)
	})

	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		// Avoid duplicates if persistent flag also in local flags
		found := false
		for _, name := range flagNames {
			if name == f.Name {
				found = true
				break
			}
		}
		if !found {
			flagNames = append(flagNames, f.Name)
		}
	})

	return flagNames
}
