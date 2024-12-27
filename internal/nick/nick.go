package nick

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"go.uber.org/zap"
)

const maxDiscordNickLen = 32
const minNameLen = 4

var (
	RegexAccNickName      = regexp.MustCompile(`^(.*) \| (.*\.\d{4})`)
	RegexGuildTagNickName = regexp.MustCompile(`^!?(\[\S{2,4}\])? ?(.*)`)
)

func SetAccAsNick(discord *discordgo.Session, member *discordgo.Member, accName string) error {
	origNick, err := GetNickname(discord, member)
	if err != nil {
		return err
	}

	newNick := AppendAccName(origNick, accName)
	if newNick != member.Nick {
		zap.L().Info("set nickname", zap.String("guildID", member.GuildID), zap.String("nick", newNick), zap.String("old nick", origNick), zap.Int("length", utf8.RuneCountInString(newNick)))
		return discord.GuildMemberNickname(member.GuildID, member.User.ID, newNick)
	}
	return nil
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

func HasAccountAsName(name string, accounts []api.Account) bool {
	// Check if account name is already appended and discord it, if so remove it
	matches := RegexAccNickName.FindStringSubmatch(name)
	if len(matches) <= 1 {
		return false
	}

	accName := matches[2]
	for _, acc := range accounts {
		// Due to how the account name may be truncated, we need to check if the account name is a suffix of the name
		if strings.HasSuffix(acc.Name, accName) {
			return true
		}
	}
	return false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func RemoveGuildTagFromNick(discord *discordgo.Session, member *discordgo.Member) (err error) {
	origNick, err := GetNickname(discord, member)
	if err != nil {
		return err
	}

	newNick := RemoveGuildTag(origNick)
	if newNick != member.Nick {
		zap.L().Info("set nickname", zap.String("guildID", member.GuildID), zap.String("nick", newNick), zap.String("old nick", origNick), zap.Int("length", utf8.RuneCountInString(newNick)))
		return discord.GuildMemberNickname(member.GuildID, member.User.ID, newNick)
	}
	return nil
}

func SetGuildTagAsNick(discord *discordgo.Session, member *discordgo.Member, guildTag string) (err error) {
	origNick, err := GetNickname(discord, member)
	if err != nil {
		return err
	}

	newNick := PrependGuildTag(origNick, guildTag)
	if newNick != member.Nick {
		zap.L().Info("set nickname", zap.String("guildID", member.GuildID), zap.String("nick", newNick), zap.String("old nick", origNick), zap.Int("length", utf8.RuneCountInString(newNick)))
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
		origName = origName[1:]
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

func GetNickname(discord *discordgo.Session, member *discordgo.Member) (nick string, err error) {
	if member.Nick == "" {
		// Attempt to ensure nickname is actually fetched
		member, err = discord.GuildMember(member.GuildID, member.User.ID)
		if err != nil {
			return "", err
		}
	}

	if member.Nick != "" {
		nick = member.Nick
	} else {
		nick = member.User.Username
	}
	return nick, nil
}
