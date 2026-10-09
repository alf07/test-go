package services

import (
	"errors"
	"regexp"
)

var ErrInvalidUUID = errors.New("invalid wallet UUID")

var walletUUIDPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

func validWalletUUID(value string) bool {
	return walletUUIDPattern.MatchString(value)
}
