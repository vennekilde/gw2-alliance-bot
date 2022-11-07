package internal

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"go.uber.org/zap"
)

const serviceID = "2"

type Bot struct {
	cache        *Cache
	interactions *Interactions
	backend      *api.GuildWars2VerificationAPI

	token   string
	discord *discordgo.Session
}

func NewBot(discordToken string, backendURL string, backendToken string) *Bot {
	b := &Bot{
		backend: api.NewGuildWars2VerificationAPI(),
		token:   discordToken,
	}
	b.backend.BaseURI = backendURL
	b.backend.AuthHeader = backendToken

	return b
}

func (b *Bot) Start() {
	var err error
	b.discord, err = discordgo.New("Bot " + b.token)
	if err != nil {
		panic(fmt.Errorf("error creating Discord session: %w", err))
	}
	b.cache = newCache(b.discord)
	b.interactions = newInteractions(b.discord, b.cache, b.backend)

	b.discord.Identify.Intents = discordgo.IntentDirectMessages | discordgo.IntentGuildMembers | discordgo.IntentsGuilds
	b.discord.StateEnabled = true

	b.discord.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		b.cache.cacheAll()
		b.interactions.register(s)

		go b.beginBackendSync()
	})

	b.discord.AddHandler(func(s *discordgo.Session, event *discordgo.GuildMemberUpdate) {
		zap.L().Info("member update", zap.Any("event", event))
	})

	// Command handler
	b.discord.AddHandler(b.interactions.onInteraction)

	err = b.discord.Open()
	if err != nil {
		panic(fmt.Errorf("error opening connection: %w", err))
	}
}

func (b *Bot) beginBackendSync() {
	go func() {
		for {
			update, resp, err := b.backend.V1.V1UpdatesService_idSubscribeGet("2", nil, nil)
			zap.L().Info("received verification update", zap.Any("update", update), zap.Any("resp", resp), zap.Any("err", err))
		}
	}()

	for {
		for _, guild := range b.discord.State.Guilds {
			var lastMemberID string
			for {
				limit := 100
				zap.L().Info("fetching guild members scheduled for refresh", zap.String("guild id", guild.ID), zap.String("guild name", guild.Name), zap.Int("limit", limit))
				members, err := b.discord.GuildMembers(guild.ID, lastMemberID, limit)
				if err != nil {
					zap.L().Error("unable to fetch guild members from server", zap.String("guild id", guild.ID), zap.String("guild name", guild.Name), zap.Error(err))
				}
				for _, member := range members {
					lastMemberID = member.User.ID
					status, _, err := b.backend.V1.V1UsersService_idService_user_idVerificationStatusGet(member.User.ID, serviceID, map[string]interface{}{}, map[string]interface{}{})
					if err != nil {
						zap.L().Error("unable to get verification status for member", zap.Any("member", member), zap.Error(err))
					}
					b.interactions.guildRoleHandler.checkRoles(guild, member, &status)
					time.Sleep(time.Second * 5)
				}
				// Check if we should fetch more members
				if len(members) == 0 || len(members) < limit {
					break
				}
			}
		}
	}
}

func (b *Bot) Close() error {
	return b.discord.Close()
}
