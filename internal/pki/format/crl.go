package format

import (
	"fmt"
)

type CrlStorage struct {
	storage StorageImplementation
}

func NewCrlStorage(storage StorageImplementation) (*CrlStorage, error) {
	return &CrlStorage{
		storage: storage,
	}, nil
}

func (out *CrlStorage) WriteCrl(crlData []byte) error {
	if out.storage == nil {
		fmt.Println(string(crlData))
		return nil
	}

	return out.storage.Write(crlData)
}
