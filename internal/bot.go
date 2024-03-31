package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/MrGunflame/gw2api"
	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"github.com/vennekilde/gw2-alliance-bot/internal/backend"
	discord_internal "github.com/vennekilde/gw2-alliance-bot/internal/discord"
	"github.com/vennekilde/gw2-alliance-bot/internal/guild"
	"github.com/vennekilde/gw2-alliance-bot/internal/interaction"
	"github.com/vennekilde/gw2-alliance-bot/internal/world"
	"go.uber.org/zap"
)

type Bot struct {
	cache        *discord_internal.Cache
	interactions *interaction.Interactions
	backend      *api.ClientWithResponses
	service      *backend.Service

	worlds           *world.Worlds
	wvw              *world.WvW
	token            string
	guildRoleHandler *guild.GuildRoleHandler
	discord          *discordgo.Session

	// Debug
	debugUser string
}

func NewBot(discordToken string, backendURL string, serviceUUID string, backendToken string, debugUser string) *Bot {
	client, _ := api.NewClientWithResponses(
		backendURL,
		api.WithBaseURL(backendURL),
		api.WithRequestEditorFn(
			func(ctx context.Context, req *http.Request) error {
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", backendToken))
				return nil
			},
		),
	)
	discord, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		panic(fmt.Errorf("error creating Discord session: %w", err))
	}

	cache := discord_internal.NewCache(discord)
	service := backend.NewService(client, serviceUUID)
	worlds := world.NewWorlds(gw2api.New())
	wvw := world.NewWvW(discord, service, worlds)
	guilds := guild.NewGuilds()
	guildRoleHandler := guild.NewGuildRoleHandler(discord, cache, guilds, service)

	b := &Bot{
		discord:          discord,
		cache:            cache,
		backend:          client,
		token:            discordToken,
		debugUser:        debugUser,
		service:          service,
		worlds:           worlds,
		wvw:              wvw,
		guildRoleHandler: guildRoleHandler,
	}
	b.interactions = interaction.NewInteractions(b.discord, b.cache, b.service, b.backend, guilds, guildRoleHandler, wvw, b.ActiveForUser)

	return b
}

func (b *Bot) Start() {
	for {
		err := b.service.Synchronize()
		if err == nil {
			break
		}
		log.Printf("unable to synchronize service settings: %v", err)
		time.Sleep(5 * time.Second)
	}

	b.worlds.Start()

	b.discord.Identify.Intents = discordgo.IntentDirectMessages | discordgo.IntentGuildMembers | discordgo.IntentsGuilds
	b.discord.StateEnabled = true

	b.discord.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		b.cache.CacheAll()

		go b.beginBackendSync()
	})

	b.discord.AddHandler(func(s *discordgo.Session, event *discordgo.GuildMemberUpdate) {
		zap.L().Info("member update", zap.Any("event", event))
	})
	b.discord.AddHandler(func(s *discordgo.Session, event *discordgo.GuildCreate) {
		zap.L().Info("guild joined", zap.Any("event", event))
		b.cache.CacheAll()
	})

	b.discord.AddHandler(func(s *discordgo.Session, event *discordgo.GuildDelete) {
		zap.L().Info("guild left", zap.Any("event", event))
		b.cache.CacheAll()
	})

	err := b.discord.Open()
	if err != nil {
		panic(fmt.Errorf("error opening connection: %w", err))
	}
}

func (b *Bot) ActiveForUser(userID string) bool {
	return b.debugUser == "" || b.debugUser == userID
}

func (b *Bot) beginBackendSync() {
	go func() {
		for {
			ctx := context.Background()
			resp, err := b.backend.GetPlatformUserUpdatesWithResponse(ctx, 2, &api.GetPlatformUserUpdatesParams{})

			var update *api.User
			if resp != nil && resp.JSON200 != nil {
				update = resp.JSON200
			} else {
				time.Sleep(10 * time.Second)
			}
			zap.L().Info("received verification update", zap.Any("update", update), zap.Any("resp", resp), zap.Any("err", err))
		}
	}()

	for {
		for _, guild := range b.discord.State.Guilds {
			var lastMemberID string
			for {
				ctx := context.Background()

				limit := 100
				zap.L().Info("fetching guild members scheduled for refresh", zap.String("guild id", guild.ID), zap.String("guild name", guild.Name), zap.Int("limit", limit))
				members, err := b.discord.GuildMembers(guild.ID, lastMemberID, limit)
				if err != nil {
					zap.L().Error("unable to fetch guild members from server", zap.String("guild id", guild.ID), zap.String("guild name", guild.Name), zap.Error(err))
				}
				for _, member := range members {
					lastMemberID = member.User.ID
					if !b.ActiveForUser(member.User.ID) {
						continue
					}

					resp, err := b.backend.GetPlatformUserWithResponse(ctx, backend.PlatformID, member.User.ID, &api.GetPlatformUserParams{})
					if err != nil {
						zap.L().Error("unable to get verification status for member", zap.Any("member", member), zap.Error(err))
						continue
					}

					if resp.JSON200 == nil {
						continue
					}
					// Ensure user has correct roles
					_ = b.guildRoleHandler.CheckRoles(guild, member, resp.JSON200.Accounts)

					err = b.wvw.VerifyWvWWorldRoles(guild.ID, member, resp.JSON200.Accounts)
					if err != nil {
						zap.L().Error("unable to verify WvW roles", zap.Any("member", member), zap.Error(err))
					}

					if b.service.GetSetting(guild.ID, backend.SettingAccRepEnabled) == "true" {
						accNames := make([]string, 0, len(resp.JSON200.Accounts))
						for _, acc := range resp.JSON200.Accounts {
							if acc.Expired == nil || !*acc.Expired {
								accNames = append(accNames, acc.Name)
							}
						}
						if len(accNames) > 0 {
							// Cache guildID in member struct, as it is not by default
							member.GuildID = guild.ID
							err := interaction.SetAccsAsNick(b.discord, member, accNames)
							if err != nil {
								zap.L().Error("unable to set nick name", zap.Any("member", member), zap.Error(err))
							}
						}
					}
				}
				// Check if we should fetch more members
				if len(members) == 0 || len(members) < limit {
					break
				}
				time.Sleep(time.Second * 5)
			}
		}
	}
}

func (b *Bot) Close() error {
	return b.discord.Close()
}
