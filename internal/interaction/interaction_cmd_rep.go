package interaction

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"github.com/vennekilde/gw2-alliance-bot/internal/backend"
	"github.com/vennekilde/gw2-alliance-bot/internal/discord"
	"github.com/vennekilde/gw2-alliance-bot/internal/guild"
	"github.com/vennekilde/gw2-alliance-bot/internal/world"
	"go.uber.org/zap"
)

const (
	InteractionIDRepGuild = "rep-guild"
	InteractionIDSRepAcc  = "rep-acc"
)

type RepCmd struct {
	backend          *api.ClientWithResponses
	cache            *discord.Cache
	guilds           *guild.Guilds
	guildRoleHandler *guild.GuildRoleHandler
	service          *backend.Service
	wvw              *world.WvW
}

func NewRepCmd(backend *api.ClientWithResponses, cache *discord.Cache, guilds *guild.Guilds, guildRoleHandler *guild.GuildRoleHandler, service *backend.Service, wvw *world.WvW) *RepCmd {
	return &RepCmd{
		backend:          backend,
		cache:            cache,
		guilds:           guilds,
		guildRoleHandler: guildRoleHandler,
		service:          service,
		wvw:              wvw,
	}
}

func (c *RepCmd) Register(i *Interactions) {
	i.interactions[InteractionIDRepGuild] = c.onSetRole
	i.interactions[InteractionIDSRepAcc] = c.InteractSetNickByAccount

	// Represent
	i.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:        "rep",
			Description: "Pick guild to represent",
		},
		handler: c.onCommandRep,
	})

}
func (c *RepCmd) onCommandRep(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	ctx := context.Background()
	if event.Member == nil {
		onError(s, event, errors.New("this command can only be used inside a discord server channel"))
		return
	}

	zap.L().Info("fetching status")
	status, err := c.backend.GetPlatformUserWithResponse(ctx, backend.PlatformID, user.ID, &api.GetPlatformUserParams{})
	if err != nil {
		onError(s, event, err)
		return
	}

	// We have the data, so might as well verify the roles, but ignore the error atm.
	_ = c.wvw.VerifyWvWWorldRoles(event.GuildID, event.Member, status.JSON200.Accounts, status.JSON200.Bans)

	c.handleRepFromStatus(s, event, user, status.JSON200.Accounts)
}

func (c *RepCmd) buildOverviewGuildComponents(guildID string, accounts []api.Account) (components []discordgo.MessageComponent, lastRole *discordgo.Role) {
	if len(accounts) == 0 {
		return components, lastRole
	}

	for _, account := range accounts {
		comps, role := c.buildGuildComponents(guildID, &account)

		if role != nil {
			lastRole = role
		}
		components = append(components, comps...)
	}

	return components, lastRole
}

func (c *RepCmd) buildGuildComponents(guildID string, account *api.Account) ([]discordgo.MessageComponent, *discordgo.Role) {
	if account.Guilds == nil {
		return nil, nil
	}

	roles := c.cache.Servers[guildID].Roles

	zap.L().Info("fetching guilds")
	guilds := c.guilds.GetGuildInfo(account.Guilds)
	components := make([]discordgo.MessageComponent, 0, len(guilds))
	var lastRole *discordgo.Role
	for _, guild := range guilds {
		var role *discordgo.Role
		for _, role = range roles {
			if role.Name == fmt.Sprintf("[%s] %s", guild.Tag, guild.Name) {
				goto guildIDFound
			}
		}
		// Guild not found
		continue

	guildIDFound:
		lastRole = role
		components = append(components, discordgo.Button{
			// Label is what the user will see on the button.
			Label: fmt.Sprintf("[%s] %s", guild.Tag, guild.Name),
			// Style provides coloring of the button. There are not so many styles tho.
			Style: discordgo.PrimaryButton,
			// CustomID is a thing telling Discord which data to send when this button will be pressed.
			CustomID: fmt.Sprintf("%s:%s:%s:%s", InteractionIDRepGuild, guild.ID, role.ID, role.Name),
		})
	}

	return components, lastRole
}

func (c *RepCmd) buildAccRepSelectMenu(accounts []api.Account) []discordgo.MessageComponent {
	components := make([]discordgo.MessageComponent, 0, len(accounts))
	for _, acc := range accounts {
		components = append(components, discordgo.Button{
			// Label is what the user will see on the button.
			Label: fmt.Sprintf("%s (%s)", acc.Name, world.WorldNames[acc.World].Name),
			// Style provides coloring of the button. There are not so many styles tho.
			Style: discordgo.PrimaryButton,
			// CustomID is a thing telling Discord which data to send when this button will be pressed.
			CustomID: fmt.Sprintf("%s:%s", InteractionIDSRepAcc, acc.Name),
		})
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: components,
		},
	}
}

func (c *RepCmd) handleRepFromStatus(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User, accounts []api.Account) {
	components, lastRole := c.buildOverviewGuildComponents(event.GuildID, accounts)
	if len(components) == 1 {
		c.setRole(s, event, user, lastRole.ID, lastRole.Name)
	} else if len(components) == 0 {
		// Only show if /rep was called directly
		if event.Type == discordgo.InteractionApplicationCommand || event.Type == discordgo.InteractionApplicationCommandAutocomplete {
			_, err := s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Flags: discordgo.MessageFlagsEphemeral,
				Embeds: []*discordgo.MessageEmbed{
					{
						Title: "Guild / Alliance Roles",
						Description: `Found no guild or alliance role on this server applicable for your Guild Wars 2 account
						
									Contact the server management if you have any questions`,
						//Color:       0x3498DB, // blue
					},
				},
			})
			if err != nil {
				onError(s, event, err)
			}
		}
	} else {
		err := c.guildRoleHandler.AddVerificationRole(event.GuildID, user.ID)
		if err != nil {
			onError(s, event, err)
		}

		_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: "Pick guild to represent",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: components,
				},
			},
		})
		if err != nil {
			onError(s, event, err)
		}
	}

	accRepEnabled := c.service.GetSetting(event.GuildID, backend.SettingAccRepEnabled)
	if accRepEnabled == "true" {
		if len(accounts) == 1 {
			err := SetAccAsNick(s, event.Member, accounts[0].Name)
			if err != nil {
				onError(s, event, err)
			}
		} else if len(accounts) > 1 {
			_, err := s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Flags:      discordgo.MessageFlagsEphemeral,
				Content:    "Pick an account to represent on this server",
				Components: c.buildAccRepSelectMenu(accounts),
			})
			if err != nil {
				onError(s, event, err)
			}
		}
	}
}

func (c *RepCmd) onSetRole(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	ctx := context.Background()

	parts := strings.Split(event.MessageComponentData().CustomID, ":")
	guildID := parts[1]
	roleID := parts[2]
	roleName := parts[3]

	err := s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		zap.L().Error("unable to respond to interaction", zap.Any("session", s), zap.Any("event", event), zap.Error(err))
	}

	// Ensure user still has the guild
	resp, err := c.backend.GetPlatformUserWithResponse(ctx, backend.PlatformID, user.ID, &api.GetPlatformUserParams{})
	if err != nil {
		onError(s, event, err)
		return
	}

	if resp.JSON200 != nil && resp.JSON200.Accounts != nil {
		for _, account := range resp.JSON200.Accounts {
			if account.Guilds == nil {
				continue
			}
			for _, accGuildID := range *account.Guilds {
				if accGuildID == guildID {
					c.setRole(s, event, user, roleID, roleName)
					return
				}
			}
		}
	}

	// If we reach this, then we failed
	onError(s, event, fmt.Errorf("unable to verify you are still elligble to represent %s", roleName))
}

func (c *RepCmd) setRole(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User, roleID string, roleName string) {
	err := c.guildRoleHandler.SetGuildRole(event.GuildID, user.ID, roleID)
	if err != nil {
		onError(s, event, err)
		return
	}

	_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
		Flags: discordgo.MessageFlagsEphemeral,
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       fmt.Sprintf("Representing %s!", roleName),
				Description: fmt.Sprintf("You have been granted the role: %s", roleName),
				Color:       0x57F287, // green
			},
		},
	})
	if err != nil {
		onError(s, event, err)
		return
	}
}

func (c *RepCmd) InteractSetNickByAccount(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	if event.GuildID == "" {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: "This command can only be used in a server",
		})
		return
	}

	parts := strings.Split(event.MessageComponentData().CustomID, ":")
	if len(parts) != 2 {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: "Invalid command",
		})
		return
	}
	accName := parts[1]

	err := SetAccAsNick(s, event.Member, accName)
	if err != nil {
		onError(s, event, err)
		return
	}

	err = s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Components: event.Message.Components,
			Content:    event.Message.Content,
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Account name updated",
					Description: fmt.Sprintf("You nickname has been updated with account name: %s", accName),
					Color:       0x57F287, // green
				},
			},
		},
	})
	if err != nil {
		onError(s, event, err)
		return
	}
}
