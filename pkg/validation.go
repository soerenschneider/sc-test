package pkg

import "errors"

func OtpValidation(s string) error {
	if !IsAsciiNumeric(s) {
		return errors.New("string is not ascii numeric")
	}

	if len(s) == 6 || len(s) == 8 {
		return nil
	}

	return errors.New("otp token length must be either 6 or 8")
}

func IsAsciiNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
