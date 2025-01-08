package interaction

import (
	"context"
	"errors"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"github.com/vennekilde/gw2-alliance-bot/internal/backend"
)

type StatusCmd struct {
	backend *api.ClientWithResponses
	ui      *UIBuilder
}

func NewStatusCmd(backend *api.ClientWithResponses, ui *UIBuilder) *StatusCmd {
	return &StatusCmd{
		backend: backend,
		ui:      ui,
	}
}

func (c *StatusCmd) Register(i *Interactions) {
	// Status cmd
	i.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:        "status",
			Description: "Display your current verification status",
		},
		handler: c.onCommandStatus,
	})

	var statsPermission int64 = discordgo.PermissionAdministrator
	var statsPermissionDM bool = false

	// Status menu
	i.addCommand(&Command{
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

func (c *StatusCmd) onCommandStatus(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	members := resolveMembersFromApplicationCommandData(event)
	for memberID, member := range members {
		ctx := context.Background()
		resp, err := c.backend.GetPlatformUserWithResponse(ctx, backend.PlatformID, memberID, &api.GetPlatformUserParams{})
		if err != nil {
			onError(s, event, err)
			return
		} else if resp.StatusCode() == http.StatusNotFound {
			onError(s, event, errors.New("you are not verified"))
			return
		} else if resp.JSON200 == nil {
			onError(s, event, errors.New("unexpected response from the server"))
			return
		}

		user := resp.JSON200
		c.sendFollowupStatusMessage(s, event, memberID, member, user)
	}
}

func (c *StatusCmd) sendFollowupStatusMessage(s *discordgo.Session, event *discordgo.InteractionCreate, memberID string, member *discordgo.Member, user *api.User) {
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
		onError(s, event, err)
	}
}
