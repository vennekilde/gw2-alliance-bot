package interaction

import (
	"context"
	"errors"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"github.com/vennekilde/gw2-alliance-bot/internal/backend"
	"github.com/vennekilde/gw2-alliance-bot/resources"
)

type APIKeysCmd struct {
	backend *api.ClientWithResponses
	ui      *UIBuilder
}

func NewAPIKeysCmd(backend *api.ClientWithResponses, ui *UIBuilder) *APIKeysCmd {
	return &APIKeysCmd{
		backend: backend,
		ui:      ui,
	}
}

func (c *APIKeysCmd) Register(i *Interactions) {
	// Status cmd
	i.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:                     resources.T("cmd.apikeys.name"),
			Description:              resources.T("cmd.apikeys.description"),
			NameLocalizations:        resources.GetLocalizations("cmd.apikeys.name"),
			DescriptionLocalizations: resources.GetLocalizations("cmd.apikeys.description"),
		},
		handler: c.onCommandAPIKeys,
	})

	var statsPermission int64 = discordgo.PermissionAdministrator
	var statsPermissionDM bool = false

	// Status menu
	i.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:                     "APIKeys",
			Type:                     discordgo.UserApplicationCommand,
			DefaultMemberPermissions: &statsPermission,
			DMPermission:             &statsPermissionDM,
		},
		handler: c.onCommandAPIKeys,
	})

}

func (c *APIKeysCmd) onCommandAPIKeys(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	locale := GetInteractionLocale(event)
	members := resolveMembersFromApplicationCommandData(event)
	for memberID, member := range members {
		ctx := context.Background()
		resp, err := c.backend.GetPlatformUserWithResponse(ctx, backend.PlatformID, memberID, &api.GetPlatformUserParams{})
		if err != nil {
			onError(s, event, err)
			return
		} else if resp.StatusCode() == http.StatusNotFound {
			onError(s, event, errors.New(resources.TL(locale, "errors.not_verified")))
			return
		} else if resp.JSON200 == nil {
			onError(s, event, errors.New(resources.TL(locale, "errors.unexpected_response")))
			return
		}

		user := resp.JSON200
		c.sendFollowupAPIKeysMessage(s, event, memberID, member, user)
	}
}

func (c *APIKeysCmd) sendFollowupAPIKeysMessage(s *discordgo.Session, event *discordgo.InteractionCreate, memberID string, member *discordgo.Member, user *api.User) {
	locale := GetInteractionLocale(event)
	embeds := c.ui.buildTokensTableEmbeds(user, locale)
	_, err := s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
		Flags:  discordgo.MessageFlagsEphemeral,
		Embeds: embeds,
	})
	if err != nil {
		onError(s, event, err)
	}
}
