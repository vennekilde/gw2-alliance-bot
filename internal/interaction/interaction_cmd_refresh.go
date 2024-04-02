package interaction

import (
	"context"
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"github.com/vennekilde/gw2-alliance-bot/internal/backend"
	"github.com/vennekilde/gw2-alliance-bot/internal/world"
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
			Name:        "refresh",
			Description: "Force the Discord bot to refresh your verification status with the verification server",
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
	members := resolveMembersFromApplicationCommandData(event)
	for memberID, member := range members {
		ctx := context.Background()
		resp, err := c.backend.PostPlatformUserRefreshWithResponse(ctx, backend.PlatformID, memberID)
		if err != nil {
			onError(s, event, err)
			return
		} else if resp.JSON200 == nil {
			onError(s, event, errors.New("unexpected response from the server"))
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
