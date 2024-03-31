package interaction

import (
	"crypto/md5" // #nosec G501 - md5 not used for cryptography
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/vennekilde/gw2-alliance-bot/internal/world"
)

// Honestly, it doesn't matter what salt is used, or if it is public knowledge. The point isn't for it to be secure
var salt = "2Qztw0zRJ0F5ThRGet7161VhcHpcPHG0cwYAT2ziS9DrX0pO0iLHL104vJUs"

// GetAPIKeyName creates a 16 character MD5 hash based on the platformUserId
// The hash doesn't need to be secure, so don't worry about it being MD5
// Additionally it prefixes the apikey prefix, along with the service id, if it is above 0
func GetAPIKeyName(worldPerspective int, platformID int, platformUserId string) string {
	name := GetAPIKeyCode(platformID, platformUserId)
	if platformID > 0 {
		name = strconv.Itoa(platformID) + "-" + name
	}
	name = world.NormalizedWorldName(worldPerspective) + name
	return name
}

// GetAPIKeyCode creates a 16 character MD5 hash based on the platformUserId
// The hash doesn't need to be secure, so don't worry about it being MD5
func GetAPIKeyCode(platformID int, platformUserId string) string {
	// #nosec G401 - md5 not used for cryptography
	md5Hasher := md5.New()
	md5Hasher.Write([]byte(salt + platformUserId))
	name := strings.ToUpper(hex.EncodeToString(md5Hasher.Sum(nil))[0:16])
	return name
}
