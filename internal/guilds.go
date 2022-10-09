package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api/types"
	"github.com/vennekilde/gw2apidb/pkg/gw2api"
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

func (g *GuildRoleHandler) checkRoles(guild *discordgo.Guild, member *discordgo.Member, status *types.VerificationStatus) {
	// Skip if not verified for now
	// TODO remove when enough are verified
	if status.Status == types.EnumVerificationStatusStatusACCESS_DENIED_ACCOUNT_NOT_LINKED ||
		status.Status == types.EnumVerificationStatusStatusACCESS_DENIED_EXPIRED {
		return
	}

	gw2Guilds := g.guilds.GetGuildInfo(status.AccountData.Guilds)
	verificationRole := g.identifyVerificationRole(guild.ID)
	serverCache := g.cache.servers[guild.ID]

	serverGuildRoles := make([]*discordgo.Role, 0, len(gw2Guilds))
	for _, guild := range gw2Guilds {
		var role *discordgo.Role
		for _, role = range serverCache.roles {
			if role.Name == fmt.Sprintf("[%s] %s", guild.Tag, guild.Name) {
				serverGuildRoles = append(serverGuildRoles, role)
				break
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
				zap.L().Warn("wanted to remove role from member", zap.String("role", role.Name), zap.Any("member", member))
				/*err := g.discord.GuildMemberRoleRemove(guild.ID, member.User.ID, roleID)
				if err != nil {
					zap.L().Warn("unable to remove role from member", zap.String("roleID", roleID), zap.Any("member", member), zap.Error(err))
				}*/
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
		zap.L().Warn("wanted to remove role from member", zap.String("role", verificationRole.Name), zap.Any("member", member))
		/*err := g.discord.GuildMemberRoleRemove(guild.ID, member.User.ID, verificationRole.ID)
		if err != nil {
			zap.L().Warn("unable to remove role from member", zap.Any("role", verificationRole), zap.Any("member", member), zap.Error(err))
		}*/
	} else if isVerified && !hasVerifiedRole {
		// Add verified role, if user is verified, but does not have it
		err := g.discord.GuildMemberRoleAdd(guild.ID, member.User.ID, verificationRole.ID)
		if err != nil {
			zap.L().Warn("unable to add role to member", zap.Any("role", verificationRole), zap.Any("member", member), zap.Error(err))
		}
	}
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
		if role.Name == "API Verified" {
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
			g.discord.GuildMemberRoleRemove(guildID, userID, memberRoleID)
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
	gw2API *gw2api.GW2Api
	cache  map[string]*Guild
}

func newGuilds() *Guilds {
	return &Guilds{
		cache:  make(map[string]*Guild),
		gw2API: gw2api.NewGW2Api(),
	}
}

func (g *Guilds) GetGuildInfo(guildIds []string) []*Guild {
	guilds := make([]*Guild, len(guildIds))
	for i, id := range guildIds {
		guild, ok := g.cache[id]
		if !ok {
			guild = &Guild{}
			err := g.fetchGuildInfo(id, guild)
			if err != nil {
				zap.L().Warn("unable to fetch guild", zap.String("guild id", id), zap.Error(err))
				continue
			}
			g.cache[id] = guild
		}
		guilds[i] = guild
	}

	return guilds
}

// Guild
type Guild struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Tag       string      `json:"tag"`
	Level     int         `json:"level"`
	MOTD      string      `json:"motd"`
	Influence int         `json:"influence"`
	Aetherium int         `json:"aetherium"`
	Resonance int         `json:"resonance"`
	Favor     int         `json:"favor"`
	Emblem    GuildEmblem `json:"emblem"`
}

// GuildEmblem
type GuildEmblem struct {
	Background gw2api.EmblemLayers `json:"background"`
	Foreground gw2api.EmblemLayers `json:"foreground"`
	Flags      []string            `json:"flags"`
}

func (g *Guilds) fetchGuildInfo(id string, result *Guild) error {
	resp, err := g.gw2API.Client.Get(fmt.Sprintf("https://api.guildwars2.com/v2/guild/%s", id))
	if err != nil {
		return err
	}

	var data []byte
	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		return err
	}

	defer resp.Body.Close()

	if err = json.Unmarshal(data, &result); err != nil {
		var gwErr gw2api.APIError
		if err = json.Unmarshal(data, &gwErr); err != nil {
			return err
		}
		return fmt.Errorf("endpoint returned error: %v", gwErr)
	}
	return err
}
