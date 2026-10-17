package internal

import (
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/terminal"
)

type TerminalQrEncoder struct {
}

func (e *TerminalQrEncoder) Encode(data string) error {
	qrc, _ := qrcode.New(data)
	w := terminal.New()
	if err := qrc.Save(w); err != nil {
		return err
	}
	return nil
}
