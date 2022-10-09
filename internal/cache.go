package internal

import (
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type Cache struct {
	discord *discordgo.Session
	servers map[string]*ServerCache
}

func newCache(discord *discordgo.Session) *Cache {
	r := &Cache{
		discord: discord,
		servers: make(map[string]*ServerCache),
	}

	discord.AddHandler(func(s *discordgo.Session, event *discordgo.GuildRoleCreate) {
		zap.L().Info("role created", zap.Any("event", event))
		server := r.servers[event.GuildID]
		if server != nil {
			server.updateRole(event.Role)
		}
	})
	discord.AddHandler(func(s *discordgo.Session, event *discordgo.GuildRoleUpdate) {
		zap.L().Info("role updated", zap.Any("event", event))
		server := r.servers[event.GuildID]
		if server != nil {
			server.updateRole(event.Role)
		}
	})
	discord.AddHandler(func(s *discordgo.Session, event *discordgo.GuildRoleDelete) {
		zap.L().Info("role deleted", zap.Any("event", event))
		server := r.servers[event.GuildID]
		if server != nil {
			server.deleteRole(event.RoleID)
		}
	})

	return r
}

func (r *Cache) cacheAll() {
	zap.L().Info("caching servers")
	for _, guild := range r.discord.State.Guilds {
		s := r.servers[guild.ID]
		if s == nil {
			s = &ServerCache{
				roles: make(map[string]*discordgo.Role),
			}
			r.servers[guild.ID] = s
		}
		err := r.cache(guild.ID, s)
		if err != nil {
			zap.L().Error("unable to cache server roles", zap.String("server", guild.ID), zap.String("server name", guild.Name), zap.Error(err))
		}
	}
	zap.L().Info("cached servers")
}

func (r *Cache) cache(serverID string, server *ServerCache) error {
	roles, err := r.discord.GuildRoles(serverID)
	if err != nil {
		return err
	}

	for _, role := range roles {
		server.updateRole(role)
	}

	return nil
}

type ServerCache struct {
	roles map[string]*discordgo.Role
}

func (c *ServerCache) updateRole(role *discordgo.Role) {
	c.roles[role.ID] = role
}

func (c *ServerCache) deleteRole(roleID string) {
	delete(c.roles, roleID)
}
