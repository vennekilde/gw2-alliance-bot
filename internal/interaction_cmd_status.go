package internal

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api/types"
)

func (c *Interactions) registerInteractionStatus() {
	// Status cmd
	c.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:        "status",
			Description: "Display your current verification status",
		},
		handler: c.onCommandStatus,
	})

	var statsPermission int64 = discordgo.PermissionAdministrator
	var statsPermissionDM bool = false

	// Status menu
	c.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:                     "Status",
			Type:                     discordgo.UserApplicationCommand,
			DefaultMemberPermissions: &statsPermission,
			DMPermission:             &statsPermissionDM,
		},
		handler: c.onCommandStatus,
	})

}

func (c *Interactions) onCommandStatus(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	var members map[string]*discordgo.Member
	if event.ApplicationCommandData().Resolved != nil && event.ApplicationCommandData().Resolved.Members != nil {
		members = event.ApplicationCommandData().Resolved.Members
	} else {
		members = map[string]*discordgo.Member{
			user.ID: event.Member,
		}
	}

	for memberID, member := range members {
		status, _, err := c.backend.V1.V1UsersService_idService_user_idVerificationStatusGet(memberID, serviceID, map[string]interface{}{}, map[string]interface{}{})
		if err != nil {
			c.onError(s, event, err)
			return
		}

		var memberName string
		if member.Nick != "" {
			memberName = member.Nick
		} else if member.User != nil {
			memberName = member.User.Username
		} else {
			memberName = event.ApplicationCommandData().Resolved.Users[memberID].Username
		}
		fields := c.buildStatusFields(memberName, &status)

		var statusDesc string
		switch status.Status {
		case types.EnumVerificationStatusStatusACCESS_DENIED_UNKNOWN:
			statusDesc = "Unable to determine status"
		case types.EnumVerificationStatusStatusACCESS_GRANTED_HOME_WORLD:
			statusDesc = "Linked with Guild Wars 2 account"
		case types.EnumVerificationStatusStatusACCESS_GRANTED_LINKED_WORLD:
			statusDesc = "Linked with Guild Wars 2 account"
		case types.EnumVerificationStatusStatusACCESS_GRANTED_HOME_WORLD_TEMPORARY:
			statusDesc = "Linked with Guild Wars 2 account"
		case types.EnumVerificationStatusStatusACCESS_GRANTED_LINKED_WORLD_TEMPORARY:
			statusDesc = "Linked with Guild Wars 2 account"
		case types.EnumVerificationStatusStatusACCESS_DENIED_INVALID_WORLD:
			statusDesc = "Linked with Guild Wars 2 account"
		case types.EnumVerificationStatusStatusACCESS_DENIED_REQUIREMENT_NOT_MET:
			statusDesc = "Linked with Guild Wars 2 account"
		case types.EnumVerificationStatusStatusACCESS_DENIED_ACCOUNT_NOT_LINKED:
			statusDesc = "Not linked with Guild Wars 2 account!\nType /verify to link with your Guild Wars 2 account"
		case types.EnumVerificationStatusStatusACCESS_DENIED_EXPIRED:
			statusDesc = "Not linked with Guild Wars 2 account!\nType /verify to link with your Guild Wars 2 account"
		case types.EnumVerificationStatusStatusACCESS_DENIED_BANNED:
			statusDesc = "Banned!\nYour Guild Wars 2 account has been blacklisted from being used with this bot"
		}

		_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Flags: discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{
				{
					Color:       0x3498DB, // blue
					Title:       "Verification Status",
					Description: statusDesc,
					Fields:      fields,
				},
			},
		})
		if err != nil {
			c.onError(s, event, err)
		}
	}
}

func (c *Interactions) buildStatusFields(memberName string, status *types.VerificationStatus) []*discordgo.MessageEmbedField {
	guilds := c.guilds.GetGuildInfo(status.AccountData.Guilds)
	guildNames := make([]string, len(guilds))
	for i, guild := range guilds {
		guildNames[i] = fmt.Sprintf("[%s] %s", guild.Tag, guild.Name)
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:  "Discord",
			Value: memberName,
		},
	}
	if status.AccountData.ID != "" {
		fields = append(fields,
			&discordgo.MessageEmbedField{
				Name:  "Account Name",
				Value: status.AccountData.Name,
			},
			&discordgo.MessageEmbedField{
				Name:  "World",
				Value: WorldNames[status.AccountData.World].Name,
			},
		)
		if len(status.AccountData.Guilds) > 0 {
			fields = append(fields,
				&discordgo.MessageEmbedField{
					Name:  "Guilds",
					Value: strings.Join(guildNames, "\n"),
				},
			)
		}
	}
	return fields
}
