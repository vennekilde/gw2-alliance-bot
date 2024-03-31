package world

import (
	"slices"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"github.com/vennekilde/gw2-alliance-bot/internal/backend"
)

type WvW struct {
	discord *discordgo.Session
	service *backend.Service
	worlds  *Worlds
}

func NewWvW(discord *discordgo.Session, service *backend.Service, worlds *Worlds) *WvW {
	return &WvW{
		discord: discord,
		service: service,
		worlds:  worlds,
	}
}

// Check if a platform user is in the correct role for their world
func (w *WvW) VerifyWvWWorldRoles(guildID string, member *discordgo.Member, accounts []api.Account) error {
	primaryWorld := w.service.GetSetting(guildID, backend.SettingWvWWorld)
	if primaryWorld == "disabled" || primaryWorld == "" {
		return nil
	}
	primaryWorldID, err := strconv.Atoi(primaryWorld)
	if err != nil {
		return err
	}
	LinkedWorlds, err := w.worlds.GetWorldLinks(primaryWorldID)
	if err != nil {
		return err
	}

	primaryRoleID := w.service.GetSetting(guildID, backend.SettingPrimaryRole)
	linkedRoleID := w.service.GetSetting(guildID, backend.SettingLinkedRole)

	hasPrimaryRole := slices.Contains(member.Roles, primaryRoleID)
	hasLinkedRole := slices.Contains(member.Roles, linkedRoleID)

	shouldHavePrimaryRole := false
	shouldHaveLinkedRole := false

	for _, account := range accounts {
		if account.World == primaryWorldID {
			shouldHavePrimaryRole = true
			// Check if user has the primary role
			if !hasPrimaryRole {
				// Add primary role
				err = w.discord.GuildMemberRoleAdd(guildID, member.User.ID, primaryRoleID)
				if err != nil {
					return err
				}
			}
		}
		if slices.Contains(LinkedWorlds, account.World) {
			shouldHaveLinkedRole = true
			// Check if user has the linked role
			if !hasLinkedRole {
				// Add linked role
				err = w.discord.GuildMemberRoleAdd(guildID, member.User.ID, linkedRoleID)
				if err != nil {
					return err
				}
			}
		}
	}

	// Check if user should have primary role
	if !shouldHavePrimaryRole && hasPrimaryRole {
		// Remove primary role
		err = w.discord.GuildMemberRoleRemove(guildID, member.User.ID, primaryRoleID)
		if err != nil {
			return err
		}
	}

	// Check if user should have linked role
	if !shouldHaveLinkedRole && hasLinkedRole {
		// Remove linked role
		err = w.discord.GuildMemberRoleRemove(guildID, member.User.ID, linkedRoleID)
		if err != nil {
			return err
		}
	}

	// Check if user should have associated roles
	if !shouldHavePrimaryRole && !shouldHaveLinkedRole {
		associatedRoles := w.service.GetSetting(guildID, backend.SettingAssociatedRoles)
		if associatedRoles != "" {
			associatedRoleIDs := strings.Split(associatedRoles, ",")
			for _, roleID := range associatedRoleIDs {
				hasRole := slices.Contains(member.Roles, roleID)
				if hasRole {
					// Remove role
					err = w.discord.GuildMemberRoleRemove(guildID, member.User.ID, roleID)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
