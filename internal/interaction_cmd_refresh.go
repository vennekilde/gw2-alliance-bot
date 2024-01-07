package internal

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

func (c *Interactions) registerInteractionRefresh() {
	// refresh cmd
	c.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:        "refresh",
			Description: "Force the Discord bot to refresh your verification status with the verification server",
		},
		handler: c.onRefresh,
	})

	var statsPermission int64 = discordgo.PermissionAdministrator
	var statsPermissionDM bool = false

	// refresh menu
	c.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:                     "Refresh",
			Type:                     discordgo.UserApplicationCommand,
			DefaultMemberPermissions: &statsPermission,
			DMPermission:             &statsPermissionDM,
		},
		handler: c.onRefresh,
	})
}

func (c *Interactions) onRefresh(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	if !c.activeForUser(user.ID) {
		return
	}

	members := c.resolveMembersFromApplicationCommandData(event)
	for memberID, member := range members {
		ctx := context.Background()
		resp, err := c.backend.PostPlatformUserRefreshWithResponse(ctx, platformID, memberID)
		if err != nil {
			c.onError(s, event, err)
			return
		}
		c.sendFollowupStatusMessage(s, event, user.ID, member, resp.JSON200)
	}
}
