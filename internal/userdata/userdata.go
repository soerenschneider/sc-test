package userdata

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/soerenschneider/sc/internal"
	"github.com/soerenschneider/sc/pkg"
)

const defaultHistoryFile = "~/.config/sc.json"

var (
	ErrCommandNotFound = errors.New("command not found")
)

type ProfileData = map[string]map[string]map[string]any

func Upsert[T any](profile, command string, values T) error {
	data, err := loadData()
	if err != nil {
		return err
	}

	raw, err := json.Marshal(values)
	if err != nil {
		return err
	}

	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return err
	}

	if data[profile] == nil {
		data[profile] = make(map[string]map[string]any)
	}
	data[profile][command] = generic

	return saveData(data)
}

func LoadCommandData[T any](profile, command string) (T, error) {
	var result T

	data, err := loadData()
	if err != nil {
		return result, err
	}

	profileData, ok := data[profile]
	if !ok {
		return result, internal.ErrProfileNotFound
	}

	cmdData, ok := profileData[command]
	if !ok {
		return result, ErrCommandNotFound
	}

	// Marshal map[string]any back to JSON before marshaling into struct
	raw, err := json.Marshal(cmdData)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(raw, &result); err != nil {
		return result, err
	}

	return result, nil
}

func getFilePath() (string, error) {
	path := pkg.GetExpandedFile(defaultHistoryFile)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return path, nil
}

func loadData() (ProfileData, error) {
	path, err := getFilePath()
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty map
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return make(ProfileData), nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	var data ProfileData
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}

func saveData(data ProfileData) error {
	path, err := getFilePath()
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ") // for prettier files
	return encoder.Encode(data)
}
