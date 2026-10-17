package oidc

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

type TokenStorage interface {
	SaveToken(token *oauth2.Token) error
	LoadToken() (*oauth2.Token, error)
	DeleteToken() error
}

func GetAuthToken(ctx context.Context, storage TokenStorage, config *oauth2.Config) (*oauth2.Token, error) {
	token, err := storage.LoadToken()
	if err == nil && token != nil {
		if token.Valid() {
			return token, nil
		}

		log.Warn().Msg("Token not valid anymore")
		tokenSource := config.TokenSource(ctx, token)

		// Refresh the token
		newToken, err := tokenSource.Token()
		if err != nil {
			log.Error().Err(err).Msg("Failed to refresh token")
		} else {
			log.Info().Msg("Token refreshed successfully")
			err = storage.SaveToken(newToken)
			if err != nil {
				log.Warn().Msg("could not store new token")
			}
			return newToken, err
		}
	}
	log.Error().Err(err).Msg("could not load token from storage, requesting new one")

	token, err = loginOidc(ctx, config)
	if err != nil {
		return nil, err
	}

	return token, storage.SaveToken(token)
}

func loginOidc(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	state := randomString(16)

	// Start local HTTP server
	address, err := extractHost(config.RedirectURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redirect url: %w", err)
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	authCodeURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	fmt.Println("Press Enter to open browser or CTRL+C to abort...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	if err := openBrowser(authCodeURL); err != nil {
		return nil, err
	}

	codeCh := make(chan string)
	server := &http.Server{
		ReadTimeout:       3 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		IdleTimeout:       3 * time.Second,
		WriteTimeout:      3 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" || r.URL.Query().Get("state") != state {
				http.Error(w, "Invalid state", http.StatusBadRequest)
				return
			}
			code := r.URL.Query().Get("code")
			fmt.Fprintln(w, "Login successful! You can close this window.")
			codeCh <- code
		}),
	}
	log.Info().Msg("Waiting for server to start...")

	go func() {
		log.Info().Msg("Starting server")
		if err := server.Serve(listener); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.Fatal().Err(err).Msg("could not start server")
			}
		}
	}()

	select {
	case <-time.After(30 * time.Second):
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)

		return nil, errors.New("timed out")
	case code := <-codeCh:
		log.Info().Msg("Received authorization code")

		token, err := config.Exchange(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("could not convert auth code to token: %w", err)
		}
		return token, nil
	}
}

func randomString(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default: // linux, etc.
		cmd = "xdg-open"
	}

	if cmd == "rundll32" {
		return exec.Command(cmd, args...).Start()
	}

	return exec.Command(cmd, url).Start()
}

func extractHost(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return parsedURL.Host, nil
}
