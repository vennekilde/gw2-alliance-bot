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
	"github.com/vennekilde/gw2-alliance-bot/internal/nick"
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

	go func() {
		for {
			err := b.service.Synchronize()
			if err != nil {
				log.Printf("unable to synchronize service settings: %v", err)
			}
			time.Sleep(5 * time.Minute)
		}
	}()

	b.worlds.Start()

	b.discord.Identify.Intents = discordgo.IntentDirectMessages | discordgo.IntentGuildMembers | discordgo.IntentsGuilds
	b.discord.StateEnabled = true

	b.discord.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		go b.beginBackendSync()
	})

	b.discord.AddHandler(func(s *discordgo.Session, event *discordgo.GuildMemberUpdate) {
		zap.L().Info("member update", zap.Any("event", event))
		ctx := context.Background()
		resp, err := b.backend.GetPlatformUserWithResponse(ctx, backend.PlatformID, event.Member.User.ID, &api.GetPlatformUserParams{})
		if err != nil {
			zap.L().Error("unable to get verification status for member", zap.Any("member", event.Member), zap.Any("resp", resp), zap.Error(err))
			return
		} else if resp.JSON200 == nil {
			return
		}

		b.guildRoleHandler.CheckRoles(event.GuildID, event.Member, resp.JSON200.Accounts)
		b.guildRoleHandler.CheckGuildTags(event.GuildID, event.Member)
		b.wvw.VerifyWvWWorldRoles(event.GuildID, event.Member, resp.JSON200.Accounts, resp.JSON200.Bans)
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

			if err != nil || resp.JSON500 != nil {
				zap.L().Error("unable to get verification update", zap.Any("resp", resp), zap.Any("err", err))
				time.Sleep(10 * time.Second)
				continue
			}

			if resp.StatusCode() == 408 {
				continue
			}

			if resp.JSON200 == nil {
				zap.L().Error("unexpected response from server", zap.Any("resp", resp))
				time.Sleep(10 * time.Second)
				continue
			}

			zap.L().Info("received verification update", zap.Any("update", resp.JSON200), zap.Any("err", err))
			err = b.RefreshUser(resp.JSON200)
			if err != nil {
				zap.L().Error("unable to refresh user", zap.Any("user", resp.JSON200), zap.Error(err))
			}
		}
	}()

	for {
		for _, guild := range b.discord.State.Guilds {
			var lastMemberID string
			for {
				ctx := context.Background()

				limit := 25
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
						zap.L().Error("unable to get verification status for member", zap.Any("member", member), zap.Any("resp", resp), zap.Error(err))
						continue
					} else if resp.JSON200 == nil {
						continue
					}

					// Cache guildID in member struct, as it is not by default
					member.GuildID = guild.ID
					err = b.RefreshMember(resp.JSON200, member)
					if err != nil {
						zap.L().Error("unable to refresh member", zap.Any("member", member), zap.Error(err))
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

func (b *Bot) RefreshUser(user *api.User) error {
	for _, platformLink := range user.PlatformLinks {
		if platformLink.PlatformID != backend.PlatformID {
			continue
		}

		for _, guild := range b.discord.State.Guilds {
			member, err := b.discord.GuildMember(guild.ID, platformLink.PlatformUserID)
			if err != nil {
				zap.L().Error("unable to get member", zap.String("guild id", guild.ID), zap.String("user id", platformLink.PlatformUserID), zap.Error(err))
				continue
			}

			if member == nil {
				continue
			}

			err = b.RefreshMember(user, member)
			if err != nil {
				zap.L().Error("unable to refresh member", zap.Any("member", member), zap.Error(err))
			}
		}
	}
	return nil
}

func (b *Bot) RefreshMember(user *api.User, member *discordgo.Member) error {
	// Ensure user has correct roles
	b.guildRoleHandler.CheckRoles(member.GuildID, member, user.Accounts)

	err := b.wvw.VerifyWvWWorldRoles(member.GuildID, member, user.Accounts, user.Bans)
	if err != nil {
		zap.L().Error("unable to verify WvW roles", zap.Any("member", member), zap.Error(err))
	}

	b.guildRoleHandler.CheckGuildTags(member.GuildID, member)

	if b.service.GetSetting(member.GuildID, backend.SettingAccRepEnabled) == "true" {
		accNames := make([]string, 0, len(user.Accounts))
		for _, acc := range user.Accounts {
			if acc.Expired == nil || !*acc.Expired {
				accNames = append(accNames, acc.Name)
			}
		}
		if len(accNames) > 0 {
			err := nick.SetAccsAsNick(b.discord, member, accNames)
			if err != nil {
				zap.L().Error("unable to set nick name", zap.Any("member", member), zap.Error(err))
			}
		}
	}

	return nil
}

func (b *Bot) Close() error {
	return b.discord.Close()
}
