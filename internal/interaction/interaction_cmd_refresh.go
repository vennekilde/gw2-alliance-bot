package interaction

import (
	"context"
	"errors"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"github.com/vennekilde/gw2-alliance-bot/internal/backend"
	"github.com/vennekilde/gw2-alliance-bot/internal/world"
	"github.com/vennekilde/gw2-alliance-bot/resources"
)

type RefreshCmd struct {
	backend   *api.ClientWithResponses
	statusCmd *StatusCmd
	wvw       *world.WvW
}

func NewRefreshCmd(backend *api.ClientWithResponses, statusCmd *StatusCmd, wvw *world.WvW) *RefreshCmd {
	return &RefreshCmd{
		backend:   backend,
		statusCmd: statusCmd,
		wvw:       wvw,
	}
}

func (c *RefreshCmd) Register(i *Interactions) {
	// refresh cmd
	i.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:                     resources.T("cmd.refresh.name"),
			Description:              resources.T("cmd.refresh.description"),
			NameLocalizations:        resources.GetLocalizations("cmd.refresh.name"),
			DescriptionLocalizations: resources.GetLocalizations("cmd.refresh.description"),
		},
		handler: c.onRefresh,
	})

	var statsPermission int64 = discordgo.PermissionAdministrator
	var statsPermissionDM bool = false

	// refresh menu
	i.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:                     "Refresh",
			Type:                     discordgo.UserApplicationCommand,
			DefaultMemberPermissions: &statsPermission,
			DMPermission:             &statsPermissionDM,
		},
		handler: c.onRefresh,
	})
}

func (c *RefreshCmd) onRefresh(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	locale := GetInteractionLocale(event)
	members := resolveMembersFromApplicationCommandData(event)
	for memberID, member := range members {
		ctx := context.Background()
		resp, err := c.backend.PostPlatformUserRefreshWithResponse(ctx, backend.PlatformID, memberID)
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

		err = c.wvw.VerifyWvWWorldRoles(event.GuildID, member, resp.JSON200.Accounts, resp.JSON200.Bans)
		if err != nil {
			onError(s, event, err)
			return
		}

		c.statusCmd.sendFollowupStatusMessage(s, event, user.ID, member, resp.JSON200)
	}
}
