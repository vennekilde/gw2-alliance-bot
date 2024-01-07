package internal

import (
	"fmt"
	"regexp"

	"github.com/MrGunflame/gw2api"
	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"go.uber.org/zap"
)

var roleNameMatcher = regexp.MustCompile(`\[\S{0,4}\] ([\S ]*)?\S`)

type GuildRoleHandler struct {
	discord *discordgo.Session
	cache   *Cache
	guilds  *Guilds
}

func newGuildRoleHandler(discord *discordgo.Session, cache *Cache, guilds *Guilds) *GuildRoleHandler {
	return &GuildRoleHandler{
		discord: discord,
		cache:   cache,
		guilds:  guilds,
	}
}

func (g *GuildRoleHandler) checkRoles(guild *discordgo.Guild, member *discordgo.Member, accounts []api.Account) (serverGuildRoles []*discordgo.Role) {
	verificationRole := g.identifyVerificationRole(guild.ID)
	serverCache := g.cache.servers[guild.ID]

	for _, account := range accounts {
		if account.Guilds == nil {
			continue
		}

		gw2Guilds := g.guilds.GetGuildInfo(account.Guilds)
		for _, guild := range gw2Guilds {
			var role *discordgo.Role
			for _, role = range serverCache.roles {
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
		if roleID == verificationRole.ID {
			hasVerifiedRole = true
			continue
		}

		role := serverCache.roles[roleID]
		if role != nil && roleNameMatcher.MatchString(role.Name) {
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
				err := g.discord.GuildMemberRoleRemove(guild.ID, member.User.ID, roleID)
				if err != nil {
					zap.L().Warn("unable to remove role from member", zap.String("roleID", roleID), zap.Any("member", member), zap.Error(err))
				}
			}
		}
	}

	if !hasAGuildRole && len(serverGuildRoles) == 1 {
		// Add guild role, if member only has 1 possible guild role
		err := g.discord.GuildMemberRoleAdd(guild.ID, member.User.ID, serverGuildRoles[0].ID)
		if err != nil {
			zap.L().Warn("unable to add role to member", zap.Any("role", verificationRole), zap.Any("member", member), zap.Error(err))
		}
	}

	if !isVerified && hasVerifiedRole {
		// Remove verified role, if user was not verified above
		err := g.discord.GuildMemberRoleRemove(guild.ID, member.User.ID, verificationRole.ID)
		if err != nil {
			zap.L().Warn("unable to remove role from member", zap.Any("role", verificationRole), zap.Any("member", member), zap.Error(err))
		}
	} else if isVerified && !hasVerifiedRole {
		// Add verified role, if user is verified, but does not have it
		err := g.discord.GuildMemberRoleAdd(guild.ID, member.User.ID, verificationRole.ID)
		if err != nil {
			zap.L().Warn("unable to add role to member", zap.Any("role", verificationRole), zap.Any("member", member), zap.Error(err))
		}
	}

	return serverGuildRoles
}

func (g *GuildRoleHandler) AddVerificationRole(guildID string, userID string) error {
	verificationRole := g.identifyVerificationRole(guildID)
	if verificationRole != nil {
		err := g.discord.GuildMemberRoleAdd(guildID, userID, verificationRole.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *GuildRoleHandler) identifyVerificationRole(guildID string) *discordgo.Role {
	serverCache := g.cache.servers[guildID]
	for _, role := range serverCache.roles {
		if role.Name == "Verified" {
			return role
		}
	}
	return nil
}

func (g *GuildRoleHandler) SetGuildRole(guildID string, userID string, roleID string) error {
	member, err := g.discord.GuildMember(guildID, userID)
	if err != nil {
		return err
	}

	verificationRole := g.identifyVerificationRole(guildID)
	serverCache := g.cache.servers[guildID]

	// Remove other guild roles
	for _, memberRoleID := range member.Roles {
		if memberRoleID == roleID || memberRoleID == verificationRole.ID {
			// Do not remove role we want to add
			continue
		}

		role := serverCache.roles[memberRoleID]
		if role != nil && roleNameMatcher.MatchString(role.Name) {
			err := g.discord.GuildMemberRoleRemove(guildID, userID, memberRoleID)
			if err != nil {
				zap.L().Error("unable to remove role from member", zap.String("guildID", guildID), zap.String("userID", userID), zap.Error(err))
			}
		}
	}

	if verificationRole != nil {
		err = g.discord.GuildMemberRoleAdd(guildID, userID, verificationRole.ID)
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

func newGuilds() *Guilds {
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
