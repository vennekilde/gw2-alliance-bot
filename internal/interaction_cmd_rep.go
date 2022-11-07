package internal

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api/types"
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
	if event.Member == nil {
		c.onError(s, event, errors.New("this command can only be used inside a discord server channel"))
		return
	}

	zap.L().Info("fetching status")
	status, _, err := c.backend.V1.V1UsersService_idService_user_idVerificationStatusGet(user.ID, serviceID, map[string]interface{}{}, map[string]interface{}{})
	if err != nil {
		c.onError(s, event, err)
		return
	}

	c.handleRepFromStatus(s, event, user, &status)
}

func (c *Interactions) buildGuildComponents(guildID string, status *types.VerificationStatus) ([]discordgo.MessageComponent, *discordgo.Role) {
	roles := c.cache.servers[guildID].roles

	zap.L().Info("fetching guilds")
	guilds := c.guilds.GetGuildInfo(status.AccountData.Guilds)
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
			CustomID: fmt.Sprintf("%s:%s:%s", InteractionIDSetRole, role.ID, role.Name),
		})
	}

	return components, lastRole
}

func (c *Interactions) handleRepFromStatus(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User, status *types.VerificationStatus) {
	var err error
	components, lastRole := c.buildGuildComponents(event.GuildID, status)

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
	parts := strings.Split(event.MessageComponentData().CustomID, ":")
	roleID := parts[1]
	roleName := parts[2]

	err := s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		zap.L().Error("unable to respond to interaction", zap.Any("session", s), zap.Any("event", event), zap.Error(err))
	}

	c.setRole(s, event, user, roleID, roleName)
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
