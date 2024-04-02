package interaction

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"unicode"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"github.com/vennekilde/gw2-alliance-bot/internal/backend"
	"github.com/vennekilde/gw2-alliance-bot/internal/discord"
	"github.com/vennekilde/gw2-alliance-bot/internal/guild"
	"github.com/vennekilde/gw2-alliance-bot/internal/world"
	"go.uber.org/zap"
)

// #nosec G101 - not passwords
const (
	InteractionIDModalAPIKey = "modal-api-key"
	InteractionIDSetAPIKey   = "set-api-key"
)

type InteractionHandler func(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User)

type Command struct {
	command *discordgo.ApplicationCommand
	handler InteractionHandler
}

type Interactions struct {
	discord          *discordgo.Session
	cache            *discord.Cache
	backend          *api.ClientWithResponses
	service          *backend.Service
	guilds           *guild.Guilds
	guildRoleHandler *guild.GuildRoleHandler
	commands         map[string]*Command
	interactions     map[string]InteractionHandler
	ui               *UIBuilder

	activeForUser func(userID string) bool
}

func NewInteractions(discord *discordgo.Session, cache *discord.Cache, service *backend.Service, backend *api.ClientWithResponses, guilds *guild.Guilds, guildRoleHandler *guild.GuildRoleHandler, wvw *world.WvW, activeForUser func(userID string) bool) *Interactions {
	c := &Interactions{
		discord:          discord,
		cache:            cache,
		commands:         make(map[string]*Command),
		interactions:     make(map[string]InteractionHandler),
		backend:          backend,
		service:          service,
		guilds:           guilds,
		guildRoleHandler: guildRoleHandler,
		activeForUser:    activeForUser,
		ui: &UIBuilder{
			guilds: guilds,
		},
	}

	statusHandler := NewStatusCmd(backend, c.ui)
	statusHandler.Register(c)

	refreshHandler := NewRefreshCmd(backend, statusHandler, wvw)
	refreshHandler.Register(c)

	repHandler := NewRepCmd(backend, cache, c.guilds, c.guildRoleHandler, service, wvw)
	repHandler.Register(c)

	verifyHandler := NewVerifyCmd(backend, c.ui, repHandler)
	verifyHandler.Register(c)

	settingsHandler := NewSettingsCmd(service)
	settingsHandler.Register(c)

	discord.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		c.register(s)
	})

	// Command handler
	discord.AddHandler(c.onInteraction)

	return c
}

func (c *Interactions) onInteraction(s *discordgo.Session, event *discordgo.InteractionCreate) {
	user := determineUser(event)
	zap.L().Info("received interaction", zap.Any("event", event), zap.Any("user", user))

	if !c.activeForUser(user.ID) {
		return
	}

	if event.Member != nil && event.GuildID != "" {
		event.Member.GuildID = event.GuildID
	}

	switch event.Type {
	case discordgo.InteractionPing:
	case discordgo.InteractionApplicationCommand:
		c.onCommand(s, event, user)
	case discordgo.InteractionMessageComponent:
		c.onMessageComponent(s, event, user)
	case discordgo.InteractionApplicationCommandAutocomplete:
	case discordgo.InteractionModalSubmit:
		c.onModalSubmit(s, event, user)
	}

}

func (c *Interactions) onCommand(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	// Handle panics
	defer func() {
		r := recover()
		if r != nil {
			zap.L().Error("panicked while handling command",
				zap.String("command", event.ApplicationCommandData().Name),
				zap.Any("options", event.ApplicationCommandData().Options),
				zap.Any("user", user.String()),
				zap.Any("recover", r),
			)
			_, err := s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Content: "Error while executing command",
			})
			if err != nil {
				zap.L().Error("unable to send error interaction response", zap.Error(err))
			}
		}
	}()

	err := s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		onError(s, event, err)
		return
	}

	zap.L().Info("received command",
		zap.String("command", event.ApplicationCommandData().Name),
		zap.Any("options", event.ApplicationCommandData().Options),
		zap.Any("user", user.String()),
	)

	var commandKey string
	if event.ApplicationCommandData().TargetID != "" {
		commandKey = fmt.Sprintf("2:%s", event.ApplicationCommandData().Name)
	} else {
		commandKey = fmt.Sprintf("0:%s", event.ApplicationCommandData().Name)
	}

	// Handle command
	if command, ok := c.commands[commandKey]; ok {
		command.handler(s, event, user)
	} else {
		onError(s, event, fmt.Errorf("unknown command name: %s", commandKey))
	}
}

func (c *Interactions) onMessageComponent(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	id := event.MessageComponentData().CustomID
	// Handle panics
	defer func() {
		r := recover()
		if r != nil {
			zap.L().Error("panicked while handling command",
				zap.String("id", id),
				zap.Any("user", user.String()),
				zap.Any("recover", r),
			)
			err := s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Error while handling interaction",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				zap.L().Error("unable to send error interaction response", zap.Error(err))
			}
		}
	}()

	zap.L().Info("received message component interaction",
		zap.String("id", id),
		zap.Any("user", user.String()),
	)
	// Handle handler
	if handler, ok := c.interactions[id]; ok {
		handler(s, event, user)
	} else {
		// ID might have data in the suffix, so check if it matches as a prefix
		for interactionID, handler := range c.interactions {
			if strings.HasPrefix(id, interactionID) {
				handler(s, event, user)
				return
			}
		}
		onError(s, event, fmt.Errorf("unknown interaction id: %s", id))
	}
}

func (c *Interactions) onModalSubmit(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	err := s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		onError(s, event, err)
		return
	}

	id := event.ModalSubmitData().CustomID
	// Handle panics
	defer func() {
		r := recover()
		if r != nil {
			zap.L().Error("panicked while handling command",
				zap.String("id", id),
				zap.Any("user", user.String()),
				zap.Any("recover", r),
			)
			_, err := s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Error while processing data",
			})
			if err != nil {
				zap.L().Error("unable to send error interaction response", zap.Error(err))
			}
		}
	}()

	zap.L().Info("received modal submit interaction",
		zap.String("id", id),
		zap.Any("user", user.String()),
	)
	// Handle handler
	if handler, ok := c.interactions[id]; ok {
		handler(s, event, user)
	} else {
		onError(s, event, errors.New("unknown interaction type"))
	}
}

func (c *Interactions) addCommand(command *Command) {
	c.commands[fmt.Sprintf("%d:%s", command.command.Type, command.command.Name)] = command
}

func (c *Interactions) register(s *discordgo.Session) {
	zap.L().Info("registering command handlers")

	appCommands := make([]*discordgo.ApplicationCommand, 0, len(c.commands))
	for _, command := range c.commands {
		appCommands = append(appCommands, command.command)
	}

	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", appCommands)
	if err != nil {
		log.Panicf("Cannot create commands: %v", err)
	}
	zap.L().Info("registered command handlers")

	//c.discord.ApplicationCommandDelete(s.State.User.ID, "", "1028039574452178985")

	// Print registered commands
	registersCommands, _ := c.discord.ApplicationCommands(s.State.User.ID, "")
	for _, cmd := range registersCommands {
		zap.L().Info("currently registered command", zap.String("id", cmd.ID), zap.String("name", cmd.Name), zap.Uint8("type", uint8(cmd.Type)))
	}
}

func determineUser(event *discordgo.InteractionCreate) *discordgo.User {
	if event.User != nil {
		return event.User
	}
	if event.Member != nil {
		return event.Member.User
	}
	if event.Message != nil {
		return event.Message.Author
	}
	return nil
}

func onError(s *discordgo.Session, event *discordgo.InteractionCreate, err error) {
	zap.L().Error("error while executing command", zap.Error(err))

	errStr := err.Error()
	/*if apiErr, ok := err.(goraml.APIError); ok {
		if typedErr, ok := apiErr.Message.(*api.Error); ok {
			errStr = typedErr.SafeDisplayError
		}
	}*/

	_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
		Flags: discordgo.MessageFlagsEphemeral,
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "Error!",
				Description: "There was a problem while processing your request",
				Color:       0xED4245, // Red
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Message",
						Value:  string(unicode.ToUpper(rune(errStr[0]))) + errStr[1:],
						Inline: false,
					},
				},
			},
		},
	})
	if err != nil {
		zap.L().Error("unable to send error interaction response", zap.Error(err))
	}
}

func resolveMembersFromApplicationCommandData(event *discordgo.InteractionCreate) map[string]*discordgo.Member {
	var members map[string]*discordgo.Member
	appComData := event.ApplicationCommandData()
	if appComData.Resolved != nil {
		members = appComData.Resolved.Members
		for id, user := range appComData.Resolved.Users {
			member := members[id]
			if member != nil {
				if member.User == nil {
					member.User = user
				}
			} else {
				members[id] = &discordgo.Member{
					User: user,
				}
			}
		}
	} else {
		user := determineUser(event)
		member := event.Member
		if member == nil {
			member = &discordgo.Member{
				User: user,
			}
		}
		members = map[string]*discordgo.Member{
			user.ID: member,
		}
	}
	return members
}
