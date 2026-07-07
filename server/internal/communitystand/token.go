package communitystand

import (
	"crypto/rand"
	"encoding/base64"
)

func NewQRToken() (string, error) {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "cst_" + base64.RawURLEncoding.EncodeToString(buf), nil
}
