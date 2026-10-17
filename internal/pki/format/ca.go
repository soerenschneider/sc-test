package format

import (
	"fmt"
)

// CaStorage accepts CA data to write to the configured storage implementation.
type CaStorage struct {
	storage StorageImplementation
}

func NewCaStorage(storage StorageImplementation) (*CaStorage, error) {
	return &CaStorage{
		storage: storage,
	}, nil
}

func (out *CaStorage) WriteCa(certData []byte) error {
	if out.storage == nil {
		fmt.Println(string(certData))
		return nil
	}

	return out.storage.Write(certData)
}
