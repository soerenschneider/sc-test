package healthcheck

import (
	"cmp"
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/multierr"
)

type ProbeResult struct {
	Err      error
	Duration time.Duration
}

func CheckDnsResolution(ctx context.Context, host string) (time.Duration, error) {
	start := time.Now()
	_, err := net.DefaultResolver.LookupHost(ctx, host)
	return time.Since(start), err
}

func CheckHTTP(ctx context.Context, url string) (time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return 0, err
	}

	start := time.Now()
	resp, err := httpClient.Do(req) //#nosec:G704
	duration := time.Since(start)
	if err != nil {
		return duration, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	return duration, nil
}

func CheckDnsServer(endpoints []string) (time.Duration, error) {
	probeResult := ProbeTcp(endpoints, time.Second*2)

	var sum int64
	var errs error
	for _, val := range probeResult {
		sum += val.Duration.Milliseconds()
		if val.Err != nil {
			errs = multierr.Append(errs, val.Err)
		}
	}

	avg := float64(sum / int64(len(endpoints)))
	return time.Millisecond * time.Duration(avg), errs
}

type InternetConnectivityOpts struct {
	DnsRecord           string
	HttpCheckUrl        string
	HttpCheckStatusCode int
	DnsServers          []string
}

func CheckInternetConnection(ctx context.Context, opts InternetConnectivityOpts) map[string]ProbeResult {

	ret := make(map[string]ProbeResult, 3)

	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(3)

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	go func() {
		dur, err := CheckDnsResolution(ctx, opts.DnsRecord)
		mu.Lock()
		ret["DNS Check"] = ProbeResult{
			Err:      err,
			Duration: dur,
		}
		mu.Unlock()
		wg.Done()
	}()

	go func() {
		dur, err := CheckHTTP(ctx, opts.HttpCheckUrl)
		mu.Lock()
		ret["HTTP Check"] = ProbeResult{
			Err:      err,
			Duration: dur,
		}
		mu.Unlock()
		wg.Done()
	}()

	go func() {
		dur, err := CheckDnsServer(opts.DnsServers)
		mu.Lock()
		ret["TCP Check"] = ProbeResult{
			Err:      err,
			Duration: dur,
		}
		mu.Unlock()
		wg.Done()
	}()

	wg.Wait()
	return ret
}

func ProbeTcp(endpoints []string, timeout time.Duration) map[string]ProbeResult {
	ret := make(map[string]ProbeResult, len(endpoints))

	wg := &sync.WaitGroup{}
	wg.Add(len(endpoints))
	mu := &sync.Mutex{}

	for _, endpoint := range endpoints {
		go func() {
			start := time.Now()
			conn, err := net.DialTimeout("tcp", endpoint, cmp.Or(timeout, 2*time.Second))
			dur := time.Since(start)

			mu.Lock()
			ret[endpoint] = ProbeResult{
				Err:      err,
				Duration: dur,
			}
			mu.Unlock()

			if err == nil {
				defer func() {
					_ = conn.Close()
				}()
			}

			wg.Done()
		}()
	}

	wg.Wait()
	return ret
}

func CheckDnsServers(servers []string, domain string, timeout time.Duration) map[string]ProbeResult {
	results := make(map[string]ProbeResult)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	for _, server := range servers {
		wg.Add(1)
		go func(server string) {
			defer wg.Done()

			resolver := &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{}
					return d.DialContext(ctx, "udp", net.JoinHostPort(server, "53"))
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			start := time.Now()
			_, err := resolver.LookupHost(ctx, domain)
			duration := time.Since(start)

			mu.Lock()
			results[server] = ProbeResult{
				Err:      err,
				Duration: duration,
			}
			mu.Unlock()
		}(server)
	}

	wg.Wait()
	return results
}
