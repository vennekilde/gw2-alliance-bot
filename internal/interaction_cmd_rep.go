package internal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"go.uber.org/zap"
)

func (c *Interactions) registerInteractionRep() {
	// Represent
	c.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:        "rep",
			Description: "Pick guild to represent",
		},
		handler: c.onCommandRep,
	})

	c.interactions[InteractionIDSetRole] = c.onSetRole
}

func (c *Interactions) onCommandRep(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	if !c.activeForUser(user.ID) {
		return
	}

	ctx := context.Background()
	if event.Member == nil {
		c.onError(s, event, errors.New("this command can only be used inside a discord server channel"))
		return
	}

	zap.L().Info("fetching status")
	status, err := c.backend.GetPlatformUserWithResponse(ctx, platformID, user.ID, &api.GetPlatformUserParams{})
	if err != nil {
		c.onError(s, event, err)
		return
	}

	c.handleRepFromStatus(s, event, user, status.JSON200.Accounts)
}

func (c *Interactions) buildOverviewGuildComponents(guildID string, accounts []api.Account) (components []discordgo.MessageComponent, lastRole *discordgo.Role) {
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

func (c *Interactions) buildGuildComponents(guildID string, account *api.Account) ([]discordgo.MessageComponent, *discordgo.Role) {
	if account.Guilds == nil {
		return nil, nil
	}

	roles := c.cache.servers[guildID].roles

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
			CustomID: fmt.Sprintf("%s:%s:%s:%s", InteractionIDSetRole, guild.ID, role.ID, role.Name),
		})
	}

	return components, lastRole
}

func (c *Interactions) handleRepFromStatus(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User, accounts []api.Account) {
	var err error
	components, lastRole := c.buildOverviewGuildComponents(event.GuildID, accounts)

	zap.L().Info("sending reply")
	if len(components) == 1 {
		c.setRole(s, event, user, lastRole.ID, lastRole.Name)
	} else if len(components) == 0 {
		_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
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
	} else {
		err = c.guildRoleHandler.AddVerificationRole(event.GuildID, user.ID)
		if err != nil {
			c.onError(s, event, err)
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
	}
	if err != nil {
		c.onError(s, event, err)
	}
}

func (c *Interactions) onSetRole(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
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
	resp, err := c.backend.GetPlatformUserWithResponse(ctx, platformID, user.ID, &api.GetPlatformUserParams{})
	if err != nil {
		c.onError(s, event, err)
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
	c.onError(s, event, fmt.Errorf("unable to verify you are still elligble to represent %s", roleName))
}

func (c *Interactions) setRole(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User, roleID string, roleName string) {
	err := c.guildRoleHandler.SetGuildRole(event.GuildID, user.ID, roleID)
	if err != nil {
		c.onError(s, event, err)
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
		c.onError(s, event, err)
		return
	}
}
