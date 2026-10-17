package formatter

import (
	"encoding/json"
)

type JsonFormatter struct {
}

func (y *JsonFormatter) Format(data map[string]string) ([]byte, error) {
	marshalled, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}

	return marshalled, nil
}
