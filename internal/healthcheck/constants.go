package healthcheck

import "github.com/hashicorp/go-retryablehttp"

var httpClient = retryablehttp.NewClient().HTTPClient
