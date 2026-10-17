package deps

import (
	"net/http"
	"sync"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog/log"
)

var (
	onceHttpClient = sync.Once{}
	httpClient     *http.Client
)

func GetHttpClient() *http.Client {
	onceHttpClient.Do(
		func() {
			c := retryablehttp.NewClient()
			c.Logger = &zerologAdapter{}
			httpClient = c.StandardClient()
		})

	return httpClient
}

type zerologAdapter struct {
}

// Debug logs a debug-level message
func (z *zerologAdapter) Debug(msg string, keysAndValues ...interface{}) {
	log.Debug().Fields(keysAndValues).Msg(msg)
}

// Info logs an info-level message
func (z *zerologAdapter) Info(msg string, keysAndValues ...interface{}) {
	log.Info().Fields(keysAndValues).Msg(msg)
}

// Warn logs a warning-level message
func (z *zerologAdapter) Warn(msg string, keysAndValues ...interface{}) {
	log.Warn().Fields(keysAndValues).Msg(msg)
}

// Error logs an error-level message
func (z *zerologAdapter) Error(msg string, keysAndValues ...interface{}) {
	log.Error().Fields(keysAndValues).Msg(msg)
}
