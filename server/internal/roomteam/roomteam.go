package roomteam

import (
	"crypto/rand"
	"encoding/base64"
	"slices"
	"strings"
	"time"
)

const TokenTTL = 10 * time.Minute

const tokenPrefix = "rmt_"

var defaultRoomNumbers = []string{
	"103",
	"118",
	"119",
	"123",
	"124",
	"125",
	"128",
	"201",
	"202",
	"203",
	"204",
	"205",
	"206",
	"207",
	"208",
	"209",
	"210",
	"212",
	"213",
	"214",
	"215",
	"216",
	"217",
	"218",
}

func DefaultRoomNumbers() []string {
	return slices.Clone(defaultRoomNumbers)
}

func NormalizeRoomNumber(roomNumber string) string {
	return strings.TrimSpace(roomNumber)
}

func ValidRoomNumber(roomNumber string) bool {
	return slices.Contains(defaultRoomNumbers, NormalizeRoomNumber(roomNumber))
}

func RoomID(roomNumber string) string {
	return "room-" + NormalizeRoomNumber(roomNumber)
}

func NewQRToken() (string, error) {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return tokenPrefix + base64.RawURLEncoding.EncodeToString(buf), nil
}
