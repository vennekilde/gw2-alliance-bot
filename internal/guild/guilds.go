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
	RegexRoleNameMatcher = regexp.MustCompile(`^\[\S{0,4}\] ([\S ]*)?\S`)
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

func (g *GuildRoleHandler) CheckGuildTags(guildID string, member *discordgo.Member) {
	member.GuildID = guildID
	// Collect list of guild roles from the member
	guildRoleTags := make(map[string]string)
	for _, roleID := range member.Roles {
		role := g.cache.Servers[guildID].Roles[roleID]
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

func (g *GuildRoleHandler) CheckRoles(guildID string, member *discordgo.Member, accounts []api.Account) (serverGuildRoles []*discordgo.Role) {
	verificationRole := g.service.GetSetting(guildID, backend.SettingGuildCommonRole)
	serverCache := g.cache.Servers[guildID]

	for _, account := range accounts {
		if account.Guilds == nil {
			continue
		}

		gw2Guilds := g.guilds.GetGuildInfo(account.Guilds)
		for _, guild := range gw2Guilds {
			var role *discordgo.Role
			for _, role = range serverCache.Roles {
				if role.Name == fmt.Sprintf("[%s] %s", guild.Tag, guild.Name) {
					serverGuildRoles = append(serverGuildRoles, role)
					break
				}
			}
		}
	}

	isVerified := len(serverGuildRoles) > 0
	hasVerifiedRole := false
	hasAGuildRole := false
	for _, roleID := range member.Roles {
		if roleID == verificationRole {
			hasVerifiedRole = true
			continue
		}

		role := serverCache.Roles[roleID]
		if role != nil && RegexRoleNameMatcher.MatchString(role.Name) {
			// Check if user is allowed to have this guild role
			isAllowedRole := false
			for _, guildRole := range serverGuildRoles {
				if role.ID == guildRole.ID {
					isAllowedRole = true
					hasAGuildRole = true
					break
				}
			}
			if !isAllowedRole {
				err := g.discord.GuildMemberRoleRemove(guildID, member.User.ID, roleID)
				if err != nil {
					zap.L().Warn("unable to remove role from member", zap.String("roleID", roleID), zap.Any("member", member), zap.Error(err))
				}
			}
		}
	}

	if !hasAGuildRole && len(serverGuildRoles) == 1 {
		// Add guild role, if member only has 1 possible guild role
		err := g.discord.GuildMemberRoleAdd(guildID, member.User.ID, serverGuildRoles[0].ID)
		if err != nil {
			zap.L().Warn("unable to add role to member", zap.Any("role", serverGuildRoles[0].ID), zap.Any("member", member), zap.Error(err))
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

	return serverGuildRoles
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
	serverCache := g.cache.Servers[guildID]

	// Remove other guild roles
	for _, memberRoleID := range member.Roles {
		if memberRoleID == roleID || memberRoleID == verificationRole {
			// Do not remove role we want to add
			continue
		}

		role := serverCache.Roles[memberRoleID]
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

func (g *Guilds) GetGuildInfo(guildIds *[]string) []*gw2api.Guild {
	if guildIds == nil {
		return nil
	}
	guilds := make([]*gw2api.Guild, 0, len(*guildIds))
	for _, id := range *guildIds {
		guild, ok := g.cache[id]
		if !ok {
			// Fetch guild from gw2api
			gw2ApiGuild, err := g.gw2API.Guild(id, false)
			if err != nil {
				zap.L().Warn("unable to fetch guild", zap.String("guild id", id), zap.Error(err))
				if err.Error() == "too many requests" {
					time.Sleep(5 * time.Second)
				}
				continue
			}
			g.cache[id] = &gw2ApiGuild
			guild = &gw2ApiGuild
		}

		if guild != nil {
			guilds = append(guilds, guild)
		}
	}

	return guilds
}
