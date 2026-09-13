package validation

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	NameRegex     = regexp.MustCompile(`^[A-Za-z0-9_]{3,80}$`)
	EmailRegex    = regexp.MustCompile(`^[A-Za-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+$`)
	LoginRegex    = regexp.MustCompile(`^[A-Za-z0-9_.@+-]{3,90}$`)
	PasswordRegex = regexp.MustCompile(`^[\x20-\x7E]{8,128}$`)
)

func Name(value string) bool {
	return NameRegex.MatchString(value)
}

func Email(value string) bool {
	return EmailRegex.MatchString(value) && utf8.RuneCountInString(value) <= 90
}

func Login(value string) bool {
	return LoginRegex.MatchString(value)
}

func Password(value string) bool {
	return PasswordRegex.MatchString(value)
}

func NormalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func Required(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("value is required")
	}
	return nil
}
