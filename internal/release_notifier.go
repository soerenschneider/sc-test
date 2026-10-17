package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

const (
	componentName = "release_watcher"
	owner         = "soerenschneider"
	repo          = "sc"
)

type Client interface {
	Do(r *http.Request) (*http.Response, error)
}

type ReleaseNotifier struct {
	client     Client
	tag        string
	updateSeen bool
}

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

func NewReleaseNotifier(client Client, tag string) (*ReleaseNotifier, error) {
	if client == nil {
		return nil, errors.New("nil client passed")
	}
	if len(tag) == 0 {
		return nil, errors.New("empty release tag passed")
	}
	return &ReleaseNotifier{
		client: client,
		tag:    tag,
	}, nil
}

func (r *ReleaseNotifier) CheckRelease(ctx context.Context) {
	release, err := GetLatestRelease(ctx, r.client, owner, repo)
	if err != nil {
		log.Error().Str("component", componentName).Err(err).Msg("check for latest release failed")
		return
	}

	if release != r.tag {
		if !r.updateSeen {
			log.Info().Str("component", componentName).Str("remote_version", release).Str("local_version", r.tag).Msg("noticed update")
		}
		r.updateSeen = true
	}
}

// GetLatestRelease fetches the latest release tag from a GitHub repository
func GetLatestRelease(ctx context.Context, client Client, owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch latest release: %s", resp.Status)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}
