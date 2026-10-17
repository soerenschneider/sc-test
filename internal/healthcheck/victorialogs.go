package healthcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"time"
)

type LogEntry struct {
	unix      int64
	Timestamp string `json:"_time"`
	Message   string `json:"_msg"`
}

func TransformLogs(data []LogEntry) ([]string, [][]string) {
	headers := []string{
		"Timestamp",
		"Message",
	}

	ret := make([][]string, 0, len(data))
	for _, logline := range data {
		ret = append(ret, []string{logline.Timestamp, logline.Message})
	}

	return headers, ret
}

type VictorialogsQuery struct {
	Address string
	Query   string
	Limit   int
}

func (q *VictorialogsQuery) GetLimit() int {
	if q.Limit <= 0 || q.Limit > 500 {
		return 25
	}

	return q.Limit
}

func buildURL(baseAddr, endpointPath string) (string, error) {
	u, err := url.Parse(baseAddr)
	if err != nil {
		return "", err
	}

	u.Path = path.Join(u.Path, endpointPath)

	return u.String(), nil
}

func QueryVictorialogs(ctx context.Context, args VictorialogsQuery) ([]LogEntry, error) {
	endpoint, err := buildURL(args.Address, "select/logsql/query")
	if err != nil {
		return nil, fmt.Errorf("could not build url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("query", args.Query)
	q.Set("limit", strconv.Itoa(args.GetLimit()))
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "application/json")

	// #nosec:G704
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bad response: %s - %s", resp.Status, body)
	}

	body, _ := io.ReadAll(resp.Body)
	lines := bytes.Split(body, []byte("\n"))
	logs := make([]LogEntry, 0, len(lines))

	for _, line := range lines {
		var entry LogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		t, err := time.Parse(time.RFC3339, entry.Timestamp)
		if err == nil {
			entry.unix = t.Unix()
			entry.Timestamp = t.Format("01-02 15:04:05")
		}

		logs = append(logs, entry)
	}

	sort.Slice(logs, func(i, j int) bool {
		return logs[i].unix < logs[j].unix
	})

	return logs, nil
}
