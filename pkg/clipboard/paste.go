package clipboard

import (
	"context"

	"golang.design/x/clipboard"
)

func PasteClipboard(ctx context.Context) (string, error) {
	err := clipboard.Init()
	if err != nil {
		return "", err
	}

	data, err := clipboard.Read(ctx, clipboard.FmtText)
	return string(data), err
}

func CopyClipboard(ctx context.Context, text string) error {
	err := clipboard.Init()
	if err != nil {
		return err
	}

	_, err = clipboard.Write(ctx, clipboard.FmtText, []byte(text))
	return err
}
