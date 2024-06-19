package guild

import (
	"fmt"
	"regexp"
	"time"

	"github.com/MrGunflame/gw2api"
	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"github.com/vennekilde/gw2-alliance-bot/internal/backend"
	"github.com/vennekilde/gw2-alliance-bot/internal/discord"
	"github.com/vennekilde/gw2-alliance-bot/internal/nick"
	"go.uber.org/zap"
)

var (
	RegexRoleNameMatcher = regexp.MustCompile(`^\[\S{0,4}\] ([\S ]*)?`)
	RegexGuildTagMatcher = regexp.MustCompile(`^\[(\S{0,4})\]`)
)

type GuildRoleHandler struct {
	discord *discordgo.Session
	cache   *discord.Cache
	guilds  *Guilds
	service *backend.Service
}

func NewGuildRoleHandler(discord *discordgo.Session, cache *discord.Cache, guilds *Guilds, service *backend.Service) *GuildRoleHandler {
	return &GuildRoleHandler{
		discord: discord,
		cache:   cache,
		guilds:  guilds,
		service: service,
	}
}

// GetMemberGuildFromRoles returns the guild the member is in, based on user's roles or nickname
// Assumes the first role found with a guild tag is the guild the member is in
func (g *GuildRoleHandler) GetMemberGuildFromRoles(member *discordgo.Member) *gw2api.Guild {
	// Check if guild tag is in nickname
	var tag string
	matches := RegexGuildTagMatcher.FindStringSubmatch(member.Nick)
	if len(matches) > 0 {
		tag = matches[1]
	}

	var guild *gw2api.Guild
	// Check each member role
	for _, roleID := range member.Roles {
		role := g.cache.GetRole(member.GuildID, roleID)
		if role == nil {
			continue
		}

		matches := RegexRoleNameMatcher.FindStringSubmatch(role.Name)
		if len(matches) > 1 {
			guild, _ = g.guilds.GetGuildInfoByName(matches[1])
			// Return early, if the role name matches the guild tag in the nickname
			if guild != nil && guild.Tag == tag {
				break
			}
		}
	}

	return guild
}

func (g *GuildRoleHandler) CheckGuildTags(guildID string, member *discordgo.Member) {
	if g.service.GetSetting(guildID, backend.SettingGuildTagRepEnabled) != "true" {
		return
	}

	member.GuildID = guildID
	// Collect list of guild roles from the member
	guildRoleTags := make(map[string]string)
	for _, roleID := range member.Roles {
		role := g.cache.GetRole(guildID, roleID)
		if role == nil {
			// For some reason, there exists roles that do not exist on the discord server, but the member appears to have it...
			continue
		}
		if RegexRoleNameMatcher.MatchString(role.Name) {
			guildRoleTags[role.Name] = RegexGuildTagMatcher.FindStringSubmatch(role.Name)[1]
		}
	}

	// Get existing guild tag from member
	var guildTag string
	matches := RegexGuildTagMatcher.FindStringSubmatch(member.Nick)
	if len(matches) > 1 {
		guildTag = matches[1]
	}

	if len(guildRoleTags) == 0 {
		if guildTag != "" {
			// Remove guild tag from member
			err := nick.RemoveGuildTagFromNick(g.discord, member)
			if err != nil {
				zap.L().Warn("unable to remove guild tag from member", zap.Any("member", member), zap.Error(err))
			}
		}
		// No guild roles, no need to continue
		return
	}

	// Check if guild tag is in guild roles
	if guildTag != "" {
		for _, tag := range guildRoleTags {
			if tag == guildTag {
				// Guild tag is in guild roles, no need to continue
				return
			}
		}
	}

	// Set guild tag as nickname
	for _, tag := range guildRoleTags {
		// Just need to pick one
		if tag != "" {
			err := nick.SetGuildTagAsNick(g.discord, member, tag)
			if err != nil {
				zap.L().Warn("unable to set guild tag as nickname", zap.Any("member", member), zap.Error(err))
				continue
			}
			break
		}
	}
}

func (g *GuildRoleHandler) CheckRoles(guildID string, member *discordgo.Member, accounts []api.Account, favoredGuildRoleID string) {
	verificationRole := g.service.GetSetting(guildID, backend.SettingGuildCommonRole)
	serverCache := g.cache.Servers[guildID]
	serverGuildRoles := map[string]*discordgo.Role{}

	for _, account := range accounts {
		if account.Guilds == nil {
			continue
		}

		gw2Guilds, partial := g.guilds.GetGuildsInfo(account.Guilds)
		if partial {
			zap.L().Warn("partial failure fetching guilds", zap.Any("guilds", account.Guilds))
			return // Partial failure, try again later
		}
		for _, guild := range gw2Guilds {
			role := serverCache.FindRoleByTagAndName(fmt.Sprintf("[%s] %s", guild.Tag, guild.Name))
			if role != nil {
				serverGuildRoles[role.ID] = role
				if favoredGuildRoleID == "" {
					// Just need to pick one as the favored role
					favoredGuildRoleID = role.ID
				}
			}
		}
	}

	isVerified := len(serverGuildRoles) > 0
	hasVerifiedRole := false
	var userGuildRoleID string
	// Check if user is a member of the guild that they have a role for
nextRole:
	for _, roleID := range member.Roles {
		// Flag if user has a the common guild verification role
		if roleID == verificationRole {
			hasVerifiedRole = true
			continue
		}

		// Check if the role the user has, is the one that got added
		if roleID == favoredGuildRoleID {
			if userGuildRoleID != "" {
				// Remove role if user has multiple guild roles
				err := g.discord.GuildMemberRoleRemove(guildID, member.User.ID, userGuildRoleID)
				if err != nil {
					zap.L().Warn("unable to remove role from member", zap.String("roleID", roleID), zap.Any("member", member), zap.Error(err))
				}
			}
			userGuildRoleID = roleID
			continue
		}

		// Check if the role is a guild role
		role := g.cache.GetRole(guildID, roleID)
		if role != nil && RegexRoleNameMatcher.MatchString(role.Name) {
			// Check if user already has a guild role
			if userGuildRoleID != "" {
				// Remove role, as user already has another guild role
				err := g.discord.GuildMemberRoleRemove(guildID, member.User.ID, roleID)
				if err != nil {
					zap.L().Warn("unable to remove role from member", zap.String("roleID", roleID), zap.Any("member", member), zap.Error(err))
				}
				continue
			}

			// Check if user is allowed to have this guild role
			for _, guildRole := range serverGuildRoles {
				if role.ID == guildRole.ID {
					userGuildRoleID = role.ID
					continue nextRole
				}
			}

			// Remove role, as user is not allowed to have it if we reach this point
			err := g.discord.GuildMemberRoleRemove(guildID, member.User.ID, roleID)
			if err != nil {
				zap.L().Warn("unable to remove role from member", zap.String("roleID", roleID), zap.Any("member", member), zap.Error(err))
			}
		}
	}

	// Check if user should have a guild role
	enforceGuildRep := g.service.GetSetting(guildID, backend.SettingEnforceGuildRep) == "true"
	if userGuildRoleID == "" && favoredGuildRoleID != "" && enforceGuildRep {
		// Add guild role, if member only has 1 possible guild role
		err := g.discord.GuildMemberRoleAdd(guildID, member.User.ID, favoredGuildRoleID)
		if err != nil {
			zap.L().Warn("unable to add role to member", zap.Any("role", favoredGuildRoleID), zap.Any("member", member), zap.Error(err))
		}
	}

	if verificationRole != "" {
		if !isVerified && hasVerifiedRole {
			// Remove verified role, if user was not verified above
			err := g.discord.GuildMemberRoleRemove(guildID, member.User.ID, verificationRole)
			if err != nil {
				zap.L().Warn("unable to remove role from member", zap.Any("role", verificationRole), zap.Any("member", member), zap.Error(err))
			}
		} else if isVerified && !hasVerifiedRole {
			// Add verified role, if user is verified, but does not have it
			err := g.discord.GuildMemberRoleAdd(guildID, member.User.ID, verificationRole)
			if err != nil {
				zap.L().Warn("unable to add role to member", zap.Any("role", verificationRole), zap.Any("member", member), zap.Error(err))
			}
		}
	}
}

func (g *GuildRoleHandler) AddVerificationRole(guildID string, userID string) error {
	verificationRole := g.service.GetSetting(guildID, backend.SettingGuildCommonRole)
	if verificationRole != "" {
		err := g.discord.GuildMemberRoleAdd(guildID, userID, verificationRole)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *GuildRoleHandler) SetGuildRole(guildID string, userID string, roleID string) error {
	member, err := g.discord.GuildMember(guildID, userID)
	if err != nil {
		return err
	}

	verificationRole := g.service.GetSetting(guildID, backend.SettingGuildCommonRole)

	// Remove other guild roles
	for _, memberRoleID := range member.Roles {
		if memberRoleID == roleID || memberRoleID == verificationRole {
			// Do not remove role we want to add
			continue
		}

		role := g.cache.GetRole(guildID, memberRoleID)
		if role != nil && RegexRoleNameMatcher.MatchString(role.Name) {
			err := g.discord.GuildMemberRoleRemove(guildID, userID, memberRoleID)
			if err != nil {
				zap.L().Error("unable to remove role from member", zap.String("guildID", guildID), zap.String("userID", userID), zap.Error(err))
			}
		}
	}

	if verificationRole != "" {
		err = g.discord.GuildMemberRoleAdd(guildID, userID, verificationRole)
		if err != nil {
			return err
		}
	}

	err = g.discord.GuildMemberRoleAdd(guildID, userID, roleID)
	if err != nil {
		return err
	}

	return nil
}

type Guilds struct {
	gw2API *gw2api.Session
	cache  map[string]*gw2api.Guild
}

func NewGuilds() *Guilds {
	return &Guilds{
		cache:  make(map[string]*gw2api.Guild),
		gw2API: gw2api.New(),
	}
}

func (g *Guilds) GetGuildsInfo(guildIds *[]string) (guilds []*gw2api.Guild, partial bool) {
	if guildIds == nil {
		return nil, false
	}
	guilds = make([]*gw2api.Guild, 0, len(*guildIds))
	for _, id := range *guildIds {
		guild, guildPartial := g.GetGuildInfo(id)
		if guildPartial {
			partial = true
		}

		if guild != nil {
			guilds = append(guilds, guild)
		}
	}

	return guilds, partial
}

func (g *Guilds) GetGuildInfo(guildId string) (guild *gw2api.Guild, partial bool) {
	if guildId == "" {
		return nil, false
	}

	guild, ok := g.cache[guildId]
	if !ok {
		// Fetch guild from gw2api
		gw2ApiGuild, err := g.gw2API.Guild(guildId, false)
		if err != nil {
			guild = &gw2api.Guild{
				ID: guildId,
			}
			zap.L().Warn("unable to fetch guild", zap.String("guild id", guildId), zap.Error(err))
			if err.Error() == "too many requests" {
				time.Sleep(5 * time.Second)
			}
			return guild, true
		}
		g.cache[guildId] = &gw2ApiGuild
		guild = &gw2ApiGuild
	}

	return guild, partial
}

// GetGuildInfoByName returns the guild info by guild name
// will only return a guild, if the guild has been fetched before
func (g *Guilds) GetGuildInfoByName(guildName string) (guild *gw2api.Guild, partial bool) {
	for _, guild := range g.cache {
		if guild.Name == guildName {
			return guild, false
		}
	}

	return nil, false
}
