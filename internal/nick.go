package internal

import (
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const maxDiscordNickLen = 32
const minNameLen = 4

var accNickNameRegex = regexp.MustCompile(`(.*) \| .*\.\d{4}`)

func setAccAsNick(discord *discordgo.Session, member *discordgo.Member, accName string) error {
	var origName string
	if member.Nick != "" {
		origName = member.Nick
	} else {
		origName = member.User.Username
	}

	newNick := appendAccName(origName, accName)
	if newNick != member.Nick {
		zap.L().Info("set nickname", zap.String("guildID", member.GuildID), zap.String("nick", newNick), zap.String("old nick", origName), zap.Int("length", utf8.RuneCountInString(newNick)))
		return discord.GuildMemberNickname(member.GuildID, member.User.ID, newNick)
	}
	return nil
}

func appendAccName(origName string, accName string) string {
	// Check if account name is already appended and discord it, if so
	matches := accNickNameRegex.FindStringSubmatch(origName)
	if len(matches) > 1 {
		origName = matches[1]
	}

	// Calc length and overhead
	origNameLen := utf8.RuneCountInString(origName)
	accNameLen := utf8.RuneCountInString(accName)
	overhead := origNameLen + accNameLen + 3 - maxDiscordNickLen

	// Check if we need to cut the nick name
	if overhead > 0 {
		// Cut name if possible, but max down to 4 chars
		if origNameLen > minNameLen {
			reduction := minInt(overhead, origNameLen-minNameLen)
			origName = origName[:origNameLen-reduction]
			overhead -= reduction
		}

		// Check if we need to cut acc name
		if overhead > 0 {
			// Take remaining length from account name
			accName = accName[overhead:]
		}
	}
	return fmt.Sprintf("%s | %s", origName, accName)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
