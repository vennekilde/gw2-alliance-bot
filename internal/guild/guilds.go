package guild

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
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

func (g *GuildRoleHandler) CheckRoles(guildID string, member *discordgo.Member, roles []string, accounts []api.Account, addedRole string) {
	verificationRole := g.service.GetSetting(guildID, backend.SettingGuildCommonRole)
	verifiedRoles := g.service.GetSettingSlice(guildID, backend.SettingGuildVerifyRoles)
	isVerified := false
	hasVerifiedRole := false

	var fallbackGuildRole string
	assignAddedRoleIfNeeded := false

	assignedGuildRoles := make(map[string]bool, 8)
	// Ensure at least a role is evaluated, in case multiple role updates are sent that overwrite each other
	if addedRole != "" && !slices.Contains(roles, addedRole) {
		// Add the added role to the list of roles
		roles = append(roles, addedRole)
		assignAddedRoleIfNeeded = true
	}

	// Check if user is a member of the guild that they have a role for¨
	for _, roleID := range roles {
		// Flag if user has a the common guild verification role
		if roleID == verificationRole {
			hasVerifiedRole = true
			continue
		}

		// Check if the role is a guild role
		role := g.cache.GetRole(guildID, roleID)
		if role == nil || !RegexRoleNameMatcher.MatchString(role.Name) {
			continue
		}

		assignedGuildRoles[role.ID] = false
	}

	serverCache := g.cache.Servers[guildID]

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
			guildFQDN := fmt.Sprintf("[%s] %s", guild.Tag, guild.Name)
			role := serverCache.FindRoleByTagAndName(guildFQDN)
			if role != nil {
				if fallbackGuildRole == "" {
					// Keep role as backup
					fallbackGuildRole = role.ID
				}

				if _, ok := assignedGuildRoles[role.ID]; ok {
					// Mark the role as permitted
					assignedGuildRoles[role.ID] = true
				}

				if !isVerified && (len(verifiedRoles) == 0 || slices.Contains(verifiedRoles, role.ID)) {
					// Ensure they are allowed to be verified
					if g.CanHaveGuildVerifiedRole(guildID, account.ApiKeys) {
						isVerified = true
					}
				}
			}
		}
	}

	// Remove guild roles not allowed
	for roleID, allowed := range assignedGuildRoles {
		if !allowed {
			err := g.discord.GuildMemberRoleRemove(guildID, member.User.ID, roleID)
			if err != nil {
				zap.L().Warn("unable to remove role from member", zap.Any("role", roleID), zap.Any("member", member), zap.Error(err))
			}
			delete(assignedGuildRoles, roleID)
		}
	}

	// Check if user should have a guild role
	enforceGuildRep := g.service.GetSetting(guildID, backend.SettingEnforceGuildRep) == "true"
	if enforceGuildRep && len(assignedGuildRoles) == 0 && fallbackGuildRole != "" {
		// Add fallback guild role, if user does not have any guild roles assigned
		err := g.discord.GuildMemberRoleAdd(guildID, member.User.ID, fallbackGuildRole)
		if err != nil {
			zap.L().Warn("unable to add role to member", zap.Any("role", fallbackGuildRole), zap.Any("member", member), zap.Error(err))
		}
	} else if assignAddedRoleIfNeeded && len(assignedGuildRoles) == 1 {
		// Due to multiple role updates, the added role is not in the list of assigned roles
		if assignedGuildRoles[addedRole] {
			// Add the added role to the list of assigned roles
			err := g.discord.GuildMemberRoleAdd(guildID, member.User.ID, addedRole)
			if err != nil {
				zap.L().Warn("unable to add role to member", zap.Any("role", addedRole), zap.Any("member", member), zap.Error(err))
			}
		}
	}

	// Check if user has too many guild roles
	if len(assignedGuildRoles) > 1 {
		var roleToKeep string
		// Check if addedRole is in the list of assigned roles
		if addedRole != "" {
			if allowed := assignedGuildRoles[addedRole]; allowed {
				// Keep the added role
				roleToKeep = addedRole
			}
		}

		for roleID := range assignedGuildRoles {
			if roleToKeep == "" {
				// Keep the first role we find
				roleToKeep = roleID
				continue
			}
			if roleID == roleToKeep {
				// Skip the role we want to keep
				continue
			}

			err := g.discord.GuildMemberRoleRemove(guildID, member.User.ID, roleID)
			if err != nil {
				zap.L().Warn("unable to remove role from member", zap.Any("role", roleID), zap.Any("member", member), zap.Error(err))
			}
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

// CanHaveGuildVerifiedRoleAccounts returns true if just one of the accounts has the required API Key permissions to have the guild verification role on this server
func (g *GuildRoleHandler) CanHaveGuildVerifiedRoleAccounts(guildID string, accounts []api.Account) bool {
	for _, account := range accounts {
		if g.CanHaveGuildVerifiedRole(guildID, account.ApiKeys) {
			return true
		}
	}
	return false
}

// CanHaveGuildVerifiedRole check if the user has the required API Key permissions to have the guild verification role on this server
func (g *GuildRoleHandler) CanHaveGuildVerifiedRole(guildID string, apiKeys []api.TokenInfo) bool {
	requiredPermissions := g.service.GetSettingSlice(guildID, backend.SettingGuildRequiredPermissions)
	if len(requiredPermissions) == 0 {
		return true
	}

	apiPermissions := make(map[string]struct{})
	for _, apiKey := range apiKeys {
		for _, permission := range apiKey.Permissions {
			permission := strings.Split(permission, ",")
			for _, p := range permission {
				apiPermissions[p] = struct{}{}
			}
		}
	}

outer:
	for _, requiredPermission := range requiredPermissions {
		for permission := range apiPermissions {
			if permission == requiredPermission {
				continue outer
			}
		}
		return false
	}
	return true
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

// GetServerGuilds returns a list of guilds that the server has
func (g *Guilds) GetServerGuilds(server *discordgo.Guild) (guilds []*gw2api.Guild) {
	guilds = make([]*gw2api.Guild, 0)

	for _, role := range server.Roles {
		if RegexRoleNameMatcher.MatchString(role.Name) {
			matches := RegexGuildTagMatcher.FindStringSubmatch(role.Name)
			if len(matches) > 1 {
				guild, _ := g.GetGuildInfoByName(matches[1])
				if guild != nil {
					guilds = append(guilds, guild)
				}
			}
		}
	}

	return guilds
}

// GetGuildRoles returns a list of guild roles that the server has
func (g *Guilds) GetGuildRoles(server *discordgo.Guild) (roles []*discordgo.Role) {
	return g.GetGuildRolesFrom(server.Roles)
}

// GetGuildRoleFrom returns a list of guild roles from a list of roles
func (g *Guilds) GetGuildRolesFrom(roles []*discordgo.Role) []*discordgo.Role {
	subset := make([]*discordgo.Role, 0)

	for _, role := range roles {
		if RegexRoleNameMatcher.MatchString(role.Name) {
			subset = append(subset, role)
		}
	}

	return subset
}
