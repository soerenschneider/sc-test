package formatter

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

type EnvVarFormatter struct {
	// uppercaseKeys prints all keys in uppercase
	uppercaseKeys bool
	// valueOnly omits the key and only prints the value.
	// Only works for secrets that contain a single key/value pair.
	valueOnly bool
}

func NewEnvVarFormatter(uppercaseKeys, valueOnly bool) *EnvVarFormatter {
	return &EnvVarFormatter{
		uppercaseKeys: uppercaseKeys,
		valueOnly:     valueOnly,
	}
}

func (y *EnvVarFormatter) Format(data map[string]string) ([]byte, error) {
	var buffer bytes.Buffer

	if y.valueOnly && len(data) > 1 {
		return nil, errors.New("can not use option 'valueOnly' with secrets containing multiple secret pairs")
	}

	for key, value := range data {
		if y.uppercaseKeys {
			key = strings.ToUpper(key)
		}
		if y.valueOnly {
			_, _ = fmt.Fprintf(&buffer, "%s\n", value)
		} else {
			_, _ = fmt.Fprintf(&buffer, "%s=%s\n", key, value)
		}
	}

	return buffer.Bytes(), nil
}
