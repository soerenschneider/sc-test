package linkding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"
)

type LinkdingClient struct {
	client *http.Client
	url    string
	token  string
}

func NewLinkdingClient(url string, token string) (*LinkdingClient, error) {
	return &LinkdingClient{
		client: &http.Client{Timeout: 3 * time.Second},
		url:    url,
		token:  token,
	}, nil
}

func buildURL(baseURL string, q string, limit int) (string, error) {
	u, err := url.Parse(baseURL + "/api/bookmarks/")
	if err != nil {
		return "", err
	}

	query := url.Values{}
	if q != "" {
		query.Set("q", q)
	}

	if limit < 50 || limit > 1000 {
		limit = 50
	}
	query.Set("limit", strconv.Itoa(limit))

	u.RawQuery = query.Encode()
	return u.String(), nil
}

func (c *LinkdingClient) AddBookmark(ctx context.Context, bookmark Bookmark) (int64, error) {
	u, err := url.Parse(c.url + "/api/bookmarks/")
	if err != nil {
		return -1, err
	}

	endpoint := u.String()

	data, _ := json.Marshal(bookmark)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return -1, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+c.token)

	// #nosec G704
	resp, err := c.client.Do(req)
	if err != nil {
		return -1, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 201 {
		return -1, fmt.Errorf("expected http code 201, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}

	var created Bookmark
	err = json.Unmarshal(body, &created)
	if err != nil {
		return -1, err
	}

	return created.ID, nil
}

func (c *LinkdingClient) ListBookmarks(ctx context.Context, query string, limit int) ([]Bookmark, error) {
	var allBookmarks []Bookmark
	startURL, err := buildURL(c.url, query, limit)
	if err != nil {
		return nil, err
	}
	url := startURL

	fetchPage := func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Token "+c.token)

		// #nosec G704
		resp, err := c.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to fetch bookmarks: %s", resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		var data BookmarksResponse
		err = json.Unmarshal(body, &data)
		if err != nil {
			return err
		}

		allBookmarks = append(allBookmarks, data.Results...)
		url = data.Next // continue to next page if available
		return nil
	}

	for url != "" {
		if err := fetchPage(); err != nil {
			return nil, err
		}
	}

	return allBookmarks, nil
}

func (c *LinkdingClient) GetAllTags(ctx context.Context) ([]Tag, error) {
	var allTags []Tag

	u, err := url.Parse(c.url + "/api/tags/")
	if err != nil {
		return nil, err
	}

	url := u.String()

	fetchPage := func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Token "+c.token)

		// #nosec G704
		resp, err := c.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to fetch tags: %s", resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		var data TagsResponse
		if err := json.Unmarshal(body, &data); err != nil {
			return err
		}

		allTags = append(allTags, data.Results...)
		url = data.Next // Follow pagination
		return nil
	}

	for url != "" {
		if err := fetchPage(); err != nil {
			return nil, err
		}

	}

	sort.Slice(allTags, func(i, j int) bool {
		return allTags[i].Name < allTags[j].Name
	})

	return allTags, nil
}
