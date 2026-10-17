package healthcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
)

type PrometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  [2]interface{}    `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func QueryPrometheus(ctx context.Context, query, endpoint string) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("query", query)
	req.URL.RawQuery = q.Encode()

	resp, err := httpClient.Do(req) //#nosec:G704
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bad response: %s - %s", resp.Status, body)
	}

	var promResp PrometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
		return nil, err
	}

	results := make(map[string]float64)
	for _, r := range promResp.Data.Result {
		instance := r.Metric["instance"]
		valueStr, ok := r.Value[1].(string)
		if !ok {
			continue
		}
		var val float64
		_, err := fmt.Sscanf(valueStr, "%f", &val)
		if err != nil {
			return nil, err
		}
		results[instance] = val
	}

	return results, nil
}

func MergePrometheusResults(maps ...map[string]float64) [][]string {
	// A map to store the merged results, where key is the string, value is a slice of float64 values.
	merged := make(map[string][]string)

	// Iterate over all input maps
	for _, m := range maps {
		// For each map, iterate over its key-value pairs
		for key, value := range m {
			merged[key] = append(merged[key], fmt.Sprintf("%.2f", value))
		}
	}

	// Prepare the result in the format of [][]any
	var result [][]string

	// Convert the merged map into a slice of slices of any
	for key, values := range merged {
		// Create a row with the key and corresponding values
		row := []string{key}
		row = append(row, values...)
		result = append(result, row)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i][0] < result[j][0]
	})

	return result
}
