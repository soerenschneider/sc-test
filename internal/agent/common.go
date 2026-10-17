package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	AgentCmdFlagsServer   = "server"
	AgentCmdFlagsCertFile = "cert-file"
	AgentCmdFlagsKeyFile  = "key-file"
	AgentCmdFlagsCaFile   = "ca-file"
	AgentCmdFlagsVerbose  = "verbose"

	defaultPort uint16 = 9999
)

func MustBuildApp(cmd *cobra.Command) *ScAgentClient {
	server, err := cmd.Flags().GetString(AgentCmdFlagsServer)
	if err != nil {
		log.Fatal().Err(err).Msg("could not get flag")
	}

	server = AddDefaultProtoAndPort(server, true, defaultPort)

	certFile, _ := cmd.Flags().GetString(AgentCmdFlagsCertFile)
	keyFile, _ := cmd.Flags().GetString(AgentCmdFlagsKeyFile)
	caFile, _ := cmd.Flags().GetString(AgentCmdFlagsCaFile)

	var opts []HttpClientOption
	if len(certFile) > 0 && len(keyFile) > 0 {
		certFile = GetExpandedFile(certFile)
		keyFile = GetExpandedFile(keyFile)
		caFile = GetExpandedFile(caFile)

		opts = append(opts, WithTlsClientCert(certFile, keyFile, caFile))
	}

	app, err := BuildApp(server, opts...)
	if err != nil {
		log.Fatal().Err(err).Msg("could not build client for sc-agent")
	}
	return app
}

func MustResponse[T any](resp *http.Response) *T {
	if resp == nil {
		log.Fatal().Msg("empty response")
		return nil // unnecessary statement but makes the linter happy
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Fatal().Int("status", resp.StatusCode).Msg("call unsuccessful")
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal().Err(err).Msg("could not read from body")
	}

	if len(data) == 0 {
		return nil
	}

	var ret *T
	if err = json.Unmarshal(data, &ret); err != nil {
		log.Fatal().Err(err).Msg("could not unmarshal data")
	}

	return ret
}

func AddDefaultProtoAndPort(server string, useHttps bool, defaultPort uint16) string {
	defaultProtocol := "https"
	if !useHttps {
		defaultProtocol = "http"
	}

	if !strings.HasPrefix(server, "https://") && !strings.HasPrefix(server, "http://") {
		server = fmt.Sprintf("%s://%s", defaultProtocol, server)
	}

	parsedURL, err := url.Parse(server)
	if err != nil {
		return server
	}

	port := parsedURL.Port()
	if len(port) == 0 {
		return fmt.Sprintf("%s:%d", server, defaultPort)
	}

	return server
}

func GetExpandedFile(filename string) string {
	usr, _ := user.Current()
	dir := usr.HomeDir

	if strings.HasPrefix(filename, "~/") {
		return filepath.Join(dir, filename[2:])
	}

	if strings.HasPrefix(filename, "$HOME/") {
		return filepath.Join(dir, filename[6:])
	}

	return filename
}
