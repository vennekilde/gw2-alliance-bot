package nick

import (
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const maxDiscordNickLen = 32
const minNameLen = 4

var (
	RegexAccNickName      = regexp.MustCompile(`^(.*) \| .*\.\d{4}`)
	RegexGuildTagNickName = regexp.MustCompile(`^!?(\[\S{2,4}\])? ?(.*)`)
)

func SetAccAsNick(discord *discordgo.Session, member *discordgo.Member, accName string) error {
	var origName string
	if member.Nick != "" {
		origName = member.Nick
	} else {
		origName = member.User.Username
	}

	newNick := AppendAccName(origName, accName)
	if newNick != member.Nick {
		zap.L().Info("set nickname", zap.String("guildID", member.GuildID), zap.String("nick", newNick), zap.String("old nick", origName), zap.Int("length", utf8.RuneCountInString(newNick)))
		return discord.GuildMemberNickname(member.GuildID, member.User.ID, newNick)
	}
	return nil
}

func SetAccsAsNick(discord *discordgo.Session, member *discordgo.Member, accNames []string) error {
	if len(accNames) == 0 {
		return nil
	}

	var newNick string
	var origName string
	if member.Nick != "" {
		origName = member.Nick
	} else {
		origName = member.User.Username
	}
	for _, accName := range accNames {
		newNick = AppendAccName(origName, accName)
		if newNick == member.Nick {
			return nil
		}
	}

	zap.L().Info("set nickname", zap.String("guildID", member.GuildID), zap.String("nick", newNick), zap.String("old nick", origName), zap.Int("length", utf8.RuneCountInString(newNick)))
	return discord.GuildMemberNickname(member.GuildID, member.User.ID, newNick)
}

func AppendAccName(origName string, accName string) string {
	// Check if account name is already appended and discord it, if so remove it
	matches := RegexAccNickName.FindStringSubmatch(origName)
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

func RemoveGuildTagFromNick(discord *discordgo.Session, member *discordgo.Member) error {
	var origName string
	if member.Nick != "" {
		origName = member.Nick
	} else {
		origName = member.User.Username
	}

	newNick := RemoveGuildTag(origName)
	if newNick != member.Nick {
		zap.L().Info("set nickname", zap.String("guildID", member.GuildID), zap.String("nick", newNick), zap.String("old nick", origName), zap.Int("length", utf8.RuneCountInString(newNick)))
		return discord.GuildMemberNickname(member.GuildID, member.User.ID, newNick)
	}
	return nil
}

func SetGuildTagAsNick(discord *discordgo.Session, member *discordgo.Member, guildTag string) error {
	var origName string
	if member.Nick != "" {
		origName = member.Nick
	} else {
		origName = member.User.Username
	}

	newNick := PrependGuildTag(origName, guildTag)
	if newNick != member.Nick {
		zap.L().Info("set nickname", zap.String("guildID", member.GuildID), zap.String("nick", newNick), zap.String("old nick", origName), zap.Int("length", utf8.RuneCountInString(newNick)))
		return discord.GuildMemberNickname(member.GuildID, member.User.ID, newNick)
	}
	return nil
}

func PrependGuildTag(origName string, guildTag string) string {
	addExclamation := origName[0] == '!'
	origName = RemoveGuildTag(origName)

	fmtStr := "[%s] %s"
	if addExclamation {
		fmtStr = "![%s] %s"
	}
	newName := fmt.Sprintf(fmtStr, guildTag, origName)
	if utf8.RuneCountInString(newName) > maxDiscordNickLen {
		newName = newName[:maxDiscordNickLen]
	}

	return newName
}

func RemoveGuildTag(origName string) string {
	addExclamation := origName[0] == '!'
	// Check if guild tag is already appended and discord it, if so remove it
	matches := RegexGuildTagNickName.FindStringSubmatch(origName)
	if len(matches) > 1 {
		origName = matches[len(matches)-1]
	}

	if addExclamation {
		origName = "!" + origName
	}
	return origName
}
