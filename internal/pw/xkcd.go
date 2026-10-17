package pw

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

const (
	wordlistsDir = "wordlists"
)

var (
	//go:embed wordlists/*.txt
	wordlists embed.FS

	specialCharacters = []string{"@", "#", "$", "%", "^", "&", "*", "(", ")", "_", "+", "-", "=", ",", ".", "/", "?"}
	validSeparators   = []string{" ", "-", "_", ".", ",", ":", ";", "/"}
)

type SpecialCharacterPosition int8

const (
	Append SpecialCharacterPosition = iota
	Prepend
	ReplaceSeparator
)

type PasswordOptions struct {
	Language   string
	NumWords   int
	Separator  string
	Lowercase  bool
	Special    bool
	SpecialPos SpecialCharacterPosition
}

func (s SpecialCharacterPosition) String() string {
	switch s {
	case Prepend:
		return "prepend"
	case Append:
		return "append"
	case ReplaceSeparator:
		return "replace-separator"
	default:
		return "unknown"
	}
}

func (s *SpecialCharacterPosition) Set(value string) error {
	switch value {
	case "prepend":
		*s = Prepend
	case "append":
		*s = Append
	case "replace-separator":
		*s = ReplaceSeparator
	default:
		return fmt.Errorf(
			"invalid special character position %q (allowed: prepend, append, replace-separator)",
			value,
		)
	}
	return nil
}

func (s *SpecialCharacterPosition) Type() string {
	return "SpecialCharacterPosition"
}

func (p *PasswordOptions) Validate() error {
	entries, err := getWordlists()
	if err != nil {
		// this is weird and should panic, likely a problem with go:embed
		panic(err)
	}

	found := false
	available := make([]string, 0, len(entries))
	for _, entry := range entries {
		langCode := filepath.Base(entry.Name())[:len(entry.Name())-4]
		available = append(available, langCode)
		if langCode == p.Language {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("language %q not found, available languages: %v", p.Language, available)
	}

	// check NumWords
	if p.NumWords < 4 {
		return errors.New("num words must be at least 5")
	}

	// check separator
	if len(p.Separator) != 1 {
		return errors.New("separator must be a single character")
	}
	if !slices.Contains(validSeparators, p.Separator) {
		return errors.New("separator must not be one of: " + strings.Join(validSeparators, " "))
	}

	return nil
}

func GenerateXkcd(opt PasswordOptions) (string, error) {
	if err := opt.Validate(); err != nil {
		return "", err
	}

	randomPassword, err := opt.generate()
	if err != nil {
		return "", err
	}

	if !opt.Special || opt.Separator != " " && isSeparatorSpecialChar(opt.Separator) {
		return randomPassword, nil
	}

	return insertSpecial(randomPassword, opt.Separator, opt.SpecialPos)
}

func (p *PasswordOptions) generate() (string, error) {
	wordlist, err := getWordlist(p.Language)
	if err != nil {
		return "", err
	}

	if len(wordlist) < 7776 {
		return "", errors.New("wordlist too small, need 7776 words")
	}

	picks := make([]string, p.NumWords)
	for i := 0; i < p.NumWords; i++ {
		word, err := randomPick(wordlist)
		if err != nil {
			return "", err
		}
		if p.Lowercase {
			word = strings.ToLower(word)
		}
		picks[i] = word
	}

	return strings.Join(picks, p.Separator), nil
}

func randomPick(choices []string) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(choices))))
	if err != nil {
		return "", err
	}
	return choices[n.Int64()], nil
}

func insertSpecial(password string, sep string, pos SpecialCharacterPosition) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	special, err := randomPick(specialCharacters)
	if err != nil {
		return "", err
	}

	switch pos {
	case Prepend:
		return special + password, nil
	case Append:
		return password + special, nil
	case ReplaceSeparator:
		if strings.Contains(password, sep) {
			upperBound := strings.Count(password, sep)
			nth, err := rand.Int(rand.Reader, big.NewInt(int64(upperBound)))
			if err != nil {
				return "", err
			}
			return replaceNthOccurrence(password, sep, special, 1+int(nth.Int64()))
		} else {
			// If no separator, just append
			return password + special, nil
		}
	default:
		// fallback (should never happen)
		return password + special, nil
	}
}

func getWordlists() ([]fs.DirEntry, error) {
	return wordlists.ReadDir(wordlistsDir)
}

func getWordlist(lang string) ([]string, error) {
	filename := path.Join(wordlistsDir, lang+".txt")
	data, err := wordlists.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var wordlist []string
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		wordlist = append(wordlist, scanner.Text())
	}

	if err != nil {
		return nil, err
	}

	return wordlist, nil
}

func isSeparatorSpecialChar(sep string) bool {
	return slices.Contains(validSeparators, sep)
}

// replaceNthOccurrence replaces the nth occurrence of old in s with new
func replaceNthOccurrence(s, old, new string, n int) (string, error) {
	if n <= 0 {
		return s, errors.New("invalid index") // invalid occurrence
	}

	index := -1
	start := 0
	for i := 0; i < n; i++ {
		pos := strings.Index(s[start:], old)
		if pos == -1 {
			return s, nil // less than n occurrences
		}
		index = start + pos
		start = index + len(old)
	}

	// Replace the nth occurrence
	return s[:index] + new + s[index+len(old):], nil
}
