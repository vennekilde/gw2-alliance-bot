package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"go.uber.org/zap"
)

const platformID = 2

type Bot struct {
	cache        *Cache
	interactions *Interactions
	backend      *api.ClientWithResponses

	token   string
	discord *discordgo.Session

	// Debug
	debugUser string
}

func NewBot(discordToken string, backendURL string, backendToken string, debugUser string) *Bot {
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

	b := &Bot{
		backend:   client,
		token:     discordToken,
		debugUser: debugUser,
	}

	return b
}

func (b *Bot) Start() {
	var err error
	b.discord, err = discordgo.New("Bot " + b.token)
	if err != nil {
		panic(fmt.Errorf("error creating Discord session: %w", err))
	}
	b.cache = newCache(b.discord)
	b.interactions = newInteractions(b.discord, b.cache, b.backend, b.ActiveForUser)

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

					resp, err := b.backend.GetPlatformUserWithResponse(ctx, platformID, member.User.ID, &api.GetPlatformUserParams{})
					if err != nil {
						zap.L().Error("unable to get verification status for member", zap.Any("member", member), zap.Error(err))
						continue
					}

					if resp.JSON200 == nil {
						continue
					}
					// Ensure user has correct roles
					_ = b.interactions.guildRoleHandler.checkRoles(guild, member, resp.JSON200.Accounts)

					// @TODO disabled until account pick can be used
					/*accName := resp.JSON200.Name
					if accName != "" {
						// Cache guildID in member struct, as it is not by default
						member.GuildID = guild.ID
						err := setAccAsNick(b.discord, member, accName)
						if err != nil {
							zap.L().Error("unable to set nick name", zap.Any("member", member), zap.Error(err))
						}
					}*/
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
