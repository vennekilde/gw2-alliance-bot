package gw2api

import (
	"time"

	"github.com/lib/pq"
	"github.com/vennekilde/gw2apidb/pkg/orm"
)

// Permission abstracts the bitmask containing permission information
type Permission uint

// Perm* represent the specific permission as required by the API
const (
	_                      = iota
	PermAccount Permission = iota
	PermCharacter
	PermInventory
	PermTradingpost
	PermWallet
	PermUnlocks
	PermPvP
	PermBuilds
	PermProgression
	PermGuilds
	PermSize
)

//
var (
	permissionsMapping = map[string]Permission{
		"account":     PermAccount,
		"characters":  PermCharacter,
		"inventories": PermInventory,
		"tradingpost": PermTradingpost,
		"wallet":      PermWallet,
		"unlocks":     PermUnlocks,
		"pvp":         PermPvP,
		"builds":      PermBuilds,
		"progression": PermProgression,
		"guilds":      PermGuilds,
	}
)

var TableNameTokenInfo = "token_infos"

// TokenInfo contains information about the provided API Key of the user.
// Including the name of the key as set by the user and the permissions
// associated with it
type TokenInfo struct {
	Gw2Model
	LastSuccess time.Time      `gorm:"default:CURRENT_TIMESTAMP"`
	ID          string         `json:"id" gorm:"type:varchar(64)"`
	Name        string         `json:"name" gorm:"type:varchar(128);NOT NULL"`
	APIKey      string         `json:"apikey" gorm:"type:varchar(128);NOT NULL"`
	AccountID   string         `json:"accountid" gorm:"type:varchar(64);NOT NULL"`
	Permissions pq.StringArray `json:"permissions" gorm:"type:varchar(255)[]"`
}

// TokenInfo requests the token information from the authenticated API
// Requires authentication
func (gw2 *GW2Api) TokenInfo() (token TokenInfo, err error) {
	ver := "v2"
	tag := "tokeninfo"
	err = gw2.fetchAuthenticatedEndpoint(ver, tag, 0, nil, &token)
	return
}

func (ent *TokenInfo) Persist(apikey string, accountID string) (err error) {
	ent.APIKey = apikey
	ent.AccountID = accountID
	ent.LastSuccess = time.Now().UTC()
	return orm.DB().Omit("db_created").Save(ent).Error
}

func (ent *TokenInfo) UpdateLastAttemptedUpdate() (err error) {
	return orm.DB().Model(ent).Where("id = ?", ent.ID).Update("db_updated", time.Now()).Error
}
func (ent *TokenInfo) UpdateLastSuccessfulUpdate() (err error) {
	return orm.DB().Model(ent).Where("id = ?", ent.ID).Updates(map[string]interface{}{
		"db_updated":   time.Now().UTC(),
		"last_success": time.Now().UTC(),
	}).Error
}
