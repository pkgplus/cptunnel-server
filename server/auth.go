package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"
)

func Authorized(username, password string, key []byte) bool {
	h := hmac.New(sha256.New, key)
	io.WriteString(h, username)
	return fmt.Sprintf("%x", h.Sum(nil)) == password
}
