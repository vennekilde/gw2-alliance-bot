package discord

import (
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type Cache struct {
	discord *discordgo.Session
	Servers map[string]*ServerCache
}

func NewCache(discord *discordgo.Session) *Cache {
	cache := &Cache{
		discord: discord,
		Servers: make(map[string]*ServerCache),
	}

	discord.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		cache.CacheAll()
	})
	discord.AddHandler(func(s *discordgo.Session, event *discordgo.GuildCreate) {
		zap.L().Info("guild joined", zap.Any("event", event))
		cache.CacheAll()
	})
	discord.AddHandler(func(s *discordgo.Session, event *discordgo.GuildDelete) {
		zap.L().Info("guild left", zap.Any("event", event))
		cache.CacheAll()
	})
	discord.AddHandler(func(s *discordgo.Session, event *discordgo.GuildRoleCreate) {
		zap.L().Info("role created", zap.Any("event", event))
		server := cache.Servers[event.GuildID]
		if server != nil {
			server.UpdateRole(event.Role)
		}
	})
	discord.AddHandler(func(s *discordgo.Session, event *discordgo.GuildRoleUpdate) {
		zap.L().Info("role updated", zap.Any("event", event))
		server := cache.Servers[event.GuildID]
		if server != nil {
			server.UpdateRole(event.Role)
		}
	})
	discord.AddHandler(func(s *discordgo.Session, event *discordgo.GuildRoleDelete) {
		zap.L().Info("role deleted", zap.Any("event", event))
		server := cache.Servers[event.GuildID]
		if server != nil {
			server.DeleteRole(event.RoleID)
		}
	})

	return cache
}

func (r *Cache) CacheAll() {
	zap.L().Info("caching servers")
	for _, guild := range r.discord.State.Guilds {
		s := r.Servers[guild.ID]
		if s == nil {
			s = &ServerCache{
				Roles: make(map[string]*discordgo.Role),
			}
			r.Servers[guild.ID] = s
		}
		err := r.Cache(guild.ID, s)
		if err != nil {
			zap.L().Error("unable to cache server roles", zap.String("server", guild.ID), zap.String("server name", guild.Name), zap.Error(err))
		}
	}
	zap.L().Info("cached servers")
}

func (r *Cache) Cache(serverID string, server *ServerCache) error {
	roles, err := r.discord.GuildRoles(serverID)
	if err != nil {
		return err
	}

	for _, role := range roles {
		server.UpdateRole(role)
	}

	return nil
}

type ServerCache struct {
	Roles map[string]*discordgo.Role
}

func (c *ServerCache) UpdateRole(role *discordgo.Role) {
	c.Roles[role.ID] = role
}

func (c *ServerCache) DeleteRole(roleID string) {
	delete(c.Roles, roleID)
}
