package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
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

func authorFromInteraction(event *discordgo.InteractionCreate, member *discordgo.Member, memberID string) *discordgo.MessageEmbedAuthor {
	var author discordgo.MessageEmbedAuthor
	if member.Nick != "" {
		author.Name = member.Nick
		author.IconURL = member.AvatarURL("")
	} else if member.User != nil {
		author.Name = member.User.Username
		author.IconURL = member.User.AvatarURL("")
	} else {
		user := event.ApplicationCommandData().Resolved.Users[memberID]
		author.Name = user.Username
		author.IconURL = user.AvatarURL("")
	}

	return &author
}

func (c *Interactions) onCommandStatus(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	if !c.activeForUser(user.ID) {
		return
	}

	members := c.resolveMembersFromApplicationCommandData(event)
	for memberID, member := range members {
		ctx := context.Background()
		resp, err := c.backend.GetPlatformUserWithResponse(ctx, platformID, memberID, &api.GetPlatformUserParams{})
		if err != nil {
			c.onError(s, event, err)
			return
		}
		user := resp.JSON200
		c.sendFollowupStatusMessage(s, event, memberID, member, user)
	}
}

func (c *Interactions) sendFollowupStatusMessage(s *discordgo.Session, event *discordgo.InteractionCreate, memberID string, member *discordgo.Member, user *api.User) {
	author := authorFromInteraction(event, member, memberID)

	accountTableFields := c.ui.buildStatusFields(user)

	activeBan := api.ActiveBan(user.Bans)
	expired := len(user.Accounts) == 0
	embeds := []*discordgo.MessageEmbed{
		{
			Color:       linkStatusColor(&expired, nil, activeBan != nil, 0x57F287, 0xED4245, 0x95A5A6), // green, red, lightgrey
			Title:       "Overview",
			Description: "Accounts linked with your discord user",
			Fields:      accountTableFields,
			Author:      author,
		},
	}

	_, err := s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
		Flags:  discordgo.MessageFlagsEphemeral,
		Embeds: embeds,
	})
	if err != nil {
		c.onError(s, event, err)
	}
}

func (c *Interactions) buildAccountEmbeds(accounts []api.Account, bans []api.Ban) []*discordgo.MessageEmbed {
	activeBan := api.ActiveBan(bans)

	embeds := []*discordgo.MessageEmbed{}
	for _, account := range accounts {
		embed := c.buildAccountEmbed(account, activeBan != nil)
		embeds = append(embeds, embed)
	}

	return embeds
}

func (c *Interactions) buildAccountEmbed(account api.Account, banned bool) *discordgo.MessageEmbed {
	fields := c.buildStatusFields(&account)

	embed := discordgo.MessageEmbed{
		Color:       linkStatusColor(account.Expired, nil, banned, 0x3498DB, 0xED4245, 0x95A5A6), // blue, red, lightgrey
		Description: linkStatus(account.Expired, nil, banned),
		Fields:      fields,
	}

	if account.Name != "" {
		embed.Title = account.Name
	} else {
		embed.Title = embed.Description
		embed.Description = ""
	}

	/*if status.World != nil {
		embed.Author = &discordgo.MessageEmbedAuthor{
			Name: WorldNames[*status.World].Name,
		}
	}*/

	return &embed
}

func (c *Interactions) buildOverviewStatusFields(accounts []api.Account) []*discordgo.MessageEmbedField {
	fields := []*discordgo.MessageEmbedField{}

	for _, account := range accounts {
		fields = append(fields, c.buildStatusFields(&account)...)
	}

	return fields
}

func (c *Interactions) buildStatusFields(account *api.Account) []*discordgo.MessageEmbedField {
	fields := []*discordgo.MessageEmbedField{}
	if account != nil {
		guilds := c.guilds.GetGuildInfo(account.Guilds)
		guildNames := make([]string, len(guilds))
		for i, guild := range guilds {
			guildNames[i] = fmt.Sprintf("[%s] %s", guild.Tag, guild.Name)
		}

		if account.Id != "" {
			/*fields = append(fields,
				&discordgo.MessageEmbedField{
					Name:  "Account Name",
					Value: status.Account.Name,
				},
			)*/

			if len(guildNames) > 0 {
				fields = append(fields,
					&discordgo.MessageEmbedField{
						Name:   "Guilds",
						Value:  strings.Join(guildNames, "\n"),
						Inline: true,
					},
				)
			}
		}
	}

	if account.World != 0 {
		fields = append(fields,
			&discordgo.MessageEmbedField{
				Name:  "World",
				Value: WorldNames[account.World].Name,
			},
		)
	}

	/*if status.Expires != nil {
		var untilStr string
		until := time.Until(*status.Expires)
		if until.Hours() > 24 {
			untilStr = fmt.Sprintf("%d days", int(until.Hours()/24))
		} else {
			untilStr = until.String()
		}
		fields = append(fields,
			&discordgo.MessageEmbedField{
				Name:  "Expires",
				Value: untilStr,
			},
		)
	}*/

	return fields
}
