package api

import (
	"strings"
	"time"
)

const (
	ACCESS_DENIED_ACCOUNT_NOT_LINKED      = ACCESSDENIEDACCOUNTNOTLINKED
	ACCESS_DENIED_BANNED                  = ACCESSDENIEDBANNED
	ACCESS_DENIED_EXPIRED                 = ACCESSDENIEDEXPIRED
	ACCESS_DENIED_INVALID_WORLD           = ACCESSDENIEDINVALIDWORLD
	ACCESS_DENIED_REQUIREMENT_NOT_MET     = ACCESSDENIEDREQUIREMENTNOTMET
	ACCESS_DENIED_UNKNOWN                 = ACCESSDENIEDUNKNOWN
	ACCESS_GRANTED_HOME_WORLD             = ACCESSGRANTEDHOMEWORLD
	ACCESS_GRANTED_HOME_WORLD_TEMPORARY   = ACCESSGRANTEDHOMEWORLDTEMPORARY
	ACCESS_GRANTED_LINKED_WORLD           = ACCESSGRANTEDLINKEDWORLD
	ACCESS_GRANTED_LINKED_WORLD_TEMPORARY = ACCESSGRANTEDLINKEDWORLDTEMPORARY
)

func (s Status) ID() int {
	switch s {
	case ACCESS_DENIED_UNKNOWN:
		return 0
	case ACCESS_GRANTED_HOME_WORLD:
		return 1
	case ACCESS_GRANTED_LINKED_WORLD:
		return 2
	case ACCESS_GRANTED_HOME_WORLD_TEMPORARY:
		return 3
	case ACCESS_GRANTED_LINKED_WORLD_TEMPORARY:
		return 4
	case ACCESS_DENIED_ACCOUNT_NOT_LINKED:
		return 5
	case ACCESS_DENIED_EXPIRED:
		return 6
	case ACCESS_DENIED_INVALID_WORLD:
		return 7
	case ACCESS_DENIED_BANNED:
		return 8
	case ACCESS_DENIED_REQUIREMENT_NOT_MET:
		return 9
	default:
		return -1
	}
}

func (s Status) Priority() int {
	switch s {
	case ACCESSDENIEDBANNED:
		return 100
	case ACCESSGRANTEDHOMEWORLD:
		return 90
	case ACCESSGRANTEDLINKEDWORLD:
		return 80
	case ACCESSGRANTEDHOMEWORLDTEMPORARY:
		return 70
	case ACCESSGRANTEDLINKEDWORLDTEMPORARY:
		return 60
	case ACCESSDENIEDINVALIDWORLD:
		return 50
	case ACCESSDENIEDEXPIRED:
		return 40
	case ACCESSDENIEDACCOUNTNOTLINKED:
		return 30
	case ACCESSDENIEDREQUIREMENTNOTMET:
		return 20
	case ACCESSDENIEDUNKNOWN:
		return 10
	default:
		return 0
	}
}

func (s Status) AccessGranted() bool {
	return strings.HasPrefix(string(s), "ACCESS_GRANTED")
}

func (s Status) AccessDenied() bool {
	return !s.AccessGranted()
}

func ActiveBan(bans []Ban) *Ban {
	var activeBan *Ban
	for _, ban := range bans {
		if ban.Until.After(time.Now()) {
			if activeBan == nil || ban.Until.After(activeBan.Until) {
				activeBan = &ban
			}
		}
	}
	return activeBan
}
