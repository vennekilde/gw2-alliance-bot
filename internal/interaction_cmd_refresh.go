package internal

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (c *Interactions) registerInteractionRefresh() {
	// refresh cmd
	c.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:        "refresh",
			Description: "Force the Discord bot to refresh your verification status with the verification server",
		},
		handler: func(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
			status, _, err := c.backend.V1.V1UsersService_idService_user_idVerificationRefreshPost(user.ID, serviceID, map[string]interface{}{}, map[string]interface{}{})
			if err != nil {
				c.onError(s, event, err)
				return
			}
			_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: fmt.Sprintf("%s\n\nPick guild to represent", status.Status),
			})
			if err != nil {
				c.onError(s, event, err)
			}
		},
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
		handler: func(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
			for memberID := range event.ApplicationCommandData().Resolved.Members {
				status, _, err := c.backend.V1.V1UsersService_idService_user_idVerificationRefreshPost(memberID, serviceID, map[string]interface{}{}, map[string]interface{}{})
				if err != nil {
					c.onError(s, event, err)
					return
				}
				_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
					Flags:   discordgo.MessageFlagsEphemeral,
					Content: fmt.Sprintf("%s\n\nPick guild to represent", status.Status),
				})
				if err != nil {
					c.onError(s, event, err)
				}
			}
		},
	})
}
