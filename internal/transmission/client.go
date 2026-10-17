package transmission

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type TransmissionClient struct {
	url       string
	client    *http.Client
	sessionId string
	sem       sync.Mutex
	username  string
	password  string
}

func NewClient(addr string) (*TransmissionClient, error) {
	return &TransmissionClient{
		url:       addr,
		client:    &http.Client{Timeout: 3 * time.Second},
		sessionId: "",
		sem:       sync.Mutex{},
	}, nil
}

func (c *TransmissionClient) GetTorrents(ctx context.Context) ([]Torrent, error) {
	reqBody := TransmissionRequest{
		Method: "torrent-get",
		Arguments: map[string]interface{}{
			"fields": []string{"id", "name", "status", "percentDone"},
		},
	}

	resp, err := c.doRequest(ctx, reqBody)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
	}

	var result torrentGetResults
	answer := answerPayload{
		Arguments: &result,
	}
	data, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(data, &answer); err != nil {
		err = fmt.Errorf("can't unmarshal request answer body: %w", err)
		return nil, err
	}

	return result.Torrents, nil

}

func (c *TransmissionClient) RemoveTorrent(ctx context.Context, torrentIDs []int64, deleteData bool) error {
	reqBody := TransmissionRequest{
		Method: "torrent-remove",
		Arguments: map[string]interface{}{
			"ids":               torrentIDs,
			"delete-local-data": deleteData,
		},
	}

	resp, err := c.doRequest(ctx, reqBody)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
	}

	var rpcResp answerPayload
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return err
	}

	if rpcResp.Result != "success" {
		return fmt.Errorf("transmission error: %s", rpcResp.Result)
	}

	return nil
}

func (c *TransmissionClient) AddTorrent(ctx context.Context, magnetLink string) error {
	reqBody := TransmissionRequest{
		Method: "torrent-add",
		Arguments: map[string]interface{}{
			"filename": magnetLink,
		},
	}

	resp, err := c.doRequest(ctx, reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
	}

	var rpcResp answerPayload
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return err
	}

	if rpcResp.Result != "success" {
		return fmt.Errorf("transmission error: %s", rpcResp.Result)
	}

	return nil
}

func (c *TransmissionClient) doRequest(ctx context.Context, payload any) (*http.Response, error) {
	return c.doRequestWithRetry(ctx, payload, 1)
}

func (c *TransmissionClient) doRequestWithRetry(ctx context.Context, payload any, attempt int) (*http.Response, error) {
	if attempt > 2 {
		return nil, fmt.Errorf("too many retries getting Transmission session ID")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.sessionId != "" {
		req.Header.Set("X-Transmission-Session-Id", c.sessionId)
	}
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	// #nosec G704
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusConflict {
		newSessionID := resp.Header.Get("X-Transmission-Session-Id")
		if newSessionID == "" {
			return nil, fmt.Errorf("server returned 409 without X-Transmission-Session-Id")
		}
		c.SetSessionId(newSessionID)
		resp.Body.Close()
		return c.doRequestWithRetry(ctx, payload, attempt+1)
	}

	return resp, nil
}
