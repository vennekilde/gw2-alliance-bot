package interaction

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/backend"
	"github.com/vennekilde/gw2-alliance-bot/internal/guild"
	"github.com/vennekilde/gw2-alliance-bot/internal/world"
	"github.com/vennekilde/gw2-alliance-bot/resources"
	"go.uber.org/zap"
)

const (
	PermissionAccount      = "account"
	PermissionInventiories = "inventories"
	PermissionCharacters   = "characters"
	PermissionTradingPost  = "tradingpost"
	PermissionWallet       = "wallet"
	PermissionUnlocks      = "unlocks"
	PermissionPvP          = "pvp"
	PermissionWvW          = "wvw"
	PermissionBuilds       = "builds"
	PermissionProgression  = "progression"
	PermissionGuilds       = "guilds"
)

var Permissions = []string{
	PermissionAccount,
	PermissionInventiories,
	PermissionCharacters,
	PermissionTradingPost,
	PermissionWallet,
	PermissionUnlocks,
	PermissionPvP,
	PermissionWvW,
	PermissionBuilds,
	PermissionProgression,
	PermissionGuilds,
}

const (
	InteractionIDSettingsSetWvWWorldDisable             = "setting-set-wvw-world-disable"
	InteractionIDSettingsSetWvWWorldEU                  = "setting-set-wvw-world-eu"
	InteractionIDSettingsSetWvWWorldEUNational          = "setting-set-wvw-world-eu-national"
	InteractionIDSettingsSetWvWWorldNA                  = "setting-set-wvw-world-na"
	InteractionIDSettingsSetPrimaryWorldRole            = "setting-set-prmary-world-role"
	InteractionIDSettingsSetLinkedWorldRole             = "setting-set-linked-world-role"
	InteractionIDSettingsSetWvWAssociatedRoles          = "setting-set-wvw-associated-roles"
	InteractionIDSettingsSetAccRepEnable                = "setting-set-acc-rep-enable"
	InteractionIDSettingsSetAccRepDisable               = "setting-set-acc-rep-disable"
	InteractionIDSettingsSetGuildTagRepEnable           = "setting-set-guild-tag-rep-enable"
	InteractionIDSettingsSetGuildTagRepDisable          = "setting-set-guild-tag-rep-disable"
	InteractionIDSettingsSetEnforceGuildTagRepEnable    = "setting-set-enforce-guild-tag-rep-enable"
	InteractionIDSettingsSetEnforceGuildTagRepDisable   = "setting-set-enforce-guild-tag-rep-disable"
	InteractionIDSettingsSetGuildCommonRole             = "setting-set-guild-common-role"
	InteractionIDSettingsSetGuildVerifyRoles            = "setting-set-guild-verify-roles"
	InteractionIDSettingsSetRolesToRemoveWhenNotInGuild = "setting-set-roles-to-remove-when-not-in-guild"
	InteractionIDSettingsSetAPIKeyPermissions           = "setting-set-api-key-permissions"
)

type SettingsCmd struct {
	service *backend.Service
	guilds  *guild.Guilds
}

func NewSettingsCmd(service *backend.Service, guilds *guild.Guilds) *SettingsCmd {
	return &SettingsCmd{
		service: service,
		guilds:  guilds,
	}
}

func (c *SettingsCmd) Register(i *Interactions) {
	i.interactions[InteractionIDSettingsSetWvWWorldDisable] = c.InteractSetWvWWorld
	i.interactions[InteractionIDSettingsSetWvWWorldEU] = c.InteractSetWvWWorld
	i.interactions[InteractionIDSettingsSetWvWWorldEUNational] = c.InteractSetWvWWorld
	i.interactions[InteractionIDSettingsSetWvWWorldNA] = c.InteractSetWvWWorld
	i.interactions[InteractionIDSettingsSetPrimaryWorldRole] = c.InteractSetWorldRole
	i.interactions[InteractionIDSettingsSetLinkedWorldRole] = c.InteractSetWorldRole
	i.interactions[InteractionIDSettingsSetWvWAssociatedRoles] = c.InteractSetAssociatedRoles
	i.interactions[InteractionIDSettingsSetAccRepEnable] = c.InteractSetAccRep
	i.interactions[InteractionIDSettingsSetAccRepDisable] = c.InteractSetAccRep
	i.interactions[InteractionIDSettingsSetGuildTagRepEnable] = c.InteractSetGuildTagRep
	i.interactions[InteractionIDSettingsSetGuildTagRepDisable] = c.InteractSetGuildTagRep
	i.interactions[InteractionIDSettingsSetEnforceGuildTagRepEnable] = c.InteractSetEnforceGuildTagRep
	i.interactions[InteractionIDSettingsSetEnforceGuildTagRepDisable] = c.InteractSetEnforceGuildTagRep
	i.interactions[InteractionIDSettingsSetGuildCommonRole] = c.InteractSetGuildCommonRole
	i.interactions[InteractionIDSettingsSetGuildVerifyRoles] = c.InteractSetGuildVerifyRoles
	i.interactions[InteractionIDSettingsSetRolesToRemoveWhenNotInGuild] = c.InteractSetRolesToRemoveWhenNotInGuild
	i.interactions[InteractionIDSettingsSetAPIKeyPermissions] = c.InteractSetRequiredAPIKeyPermissions

	var permission int64 = discordgo.PermissionAdministrator
	var permissionDM bool = false

	// Settings command
	i.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:                     resources.T("cmd.settings.name"),
			Description:              resources.T("cmd.settings.description"),
			NameLocalizations:        resources.GetLocalizations("cmd.settings.name"),
			DescriptionLocalizations: resources.GetLocalizations("cmd.settings.description"),
			DefaultMemberPermissions: &permission,
			DMPermission:             &permissionDM,
		},
		handler: func(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
			//ctx := context.Background()
			locale := GetInteractionLocale(event)
			currentWorld := c.service.GetSetting(event.GuildID, backend.SettingWvWWorld)
			currentWorldID, _ := strconv.Atoi(currentWorld)
			wvwWorldSelectComponents := buildWvWWorldSelectMenu(currentWorldID)

			_, err := s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Content:    resources.TL(locale, "settings.wvw_world.title"),
				Flags:      discordgo.MessageFlagsEphemeral,
				Components: wvwWorldSelectComponents,
			})
			if err != nil {
				onError(s, event, err)
			}

			currentPrimaryRole := c.service.GetSetting(event.GuildID, backend.SettingPrimaryRole)
			currentLinkedRole := c.service.GetSetting(event.GuildID, backend.SettingLinkedRole)
			worldRoleSelectComponents := buildWorldRoleSelectMenu(currentPrimaryRole, currentLinkedRole)

			_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Content:    resources.TL(locale, "settings.wvw_roles.title"),
				Flags:      discordgo.MessageFlagsEphemeral,
				Components: worldRoleSelectComponents,
			})
			if err != nil {
				onError(s, event, err)
			}

			accRepComponents := c.buildAccountRepToggle(event.GuildID)
			_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Content:    resources.TL(locale, "settings.account_rep.title"),
				Flags:      discordgo.MessageFlagsEphemeral,
				Components: accRepComponents,
			})
			if err != nil {
				onError(s, event, err)
			}

			guildTagRepComponents := c.buildGuildTagRepToggle(event.GuildID)
			_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Content:    resources.TL(locale, "settings.guild_rep.title"),
				Flags:      discordgo.MessageFlagsEphemeral,
				Components: guildTagRepComponents,
			})
			if err != nil {
				onError(s, event, err)
			}

			currentCommonGuildRole := c.service.GetSetting(event.GuildID, backend.SettingGuildCommonRole)
			currentRequiredPermissions := c.service.GetSettingSlice(event.GuildID, backend.SettingGuildRequiredPermissions)
			currentGuildVerifyRoles := c.service.GetSettingSlice(event.GuildID, backend.SettingGuildVerifyRoles)
			currentGuildRolesToRemove := c.service.GetSettingSlice(event.GuildID, backend.SettingRolesToRemoveWhenNotInGuild)
			roles, err := s.GuildRoles(event.GuildID)
			if err != nil {
				onError(s, event, err)
			}

			guildCommonRoleComponents := c.buildGuildVerificationMenu(roles, currentCommonGuildRole, currentGuildVerifyRoles, currentGuildRolesToRemove, currentRequiredPermissions)
			_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Content:    resources.TL(locale, "settings.guild_verification.title"),
				Flags:      discordgo.MessageFlagsEphemeral,
				Components: guildCommonRoleComponents,
			})
			if err != nil {
				onError(s, event, err)
			}
		},
	})
}

func buildWvWWorldSelectMenu(currentWorldID int) []discordgo.MessageComponent {
	worlds := world.WorldsSorted()

	// Can only return 25 options, so we need to split them up somehow
	euWorldOptions := make([]discordgo.SelectMenuOption, 0, len(worlds))
	euNationalWorldOptions := make([]discordgo.SelectMenuOption, 0, len(worlds))
	naWorldOptions := make([]discordgo.SelectMenuOption, 0, len(worlds))
	disable := discordgo.Button{
		Label:    resources.T("settings.wvw_world.button_disable"),
		Style:    discordgo.DangerButton,
		CustomID: InteractionIDSettingsSetWvWWorldDisable,
	}
	for _, world := range worlds {
		option := discordgo.SelectMenuOption{
			Label: world.Name,
			Value: strconv.Itoa(world.ID),
		}
		if currentWorldID == world.ID {
			option.Default = true
		}
		if world.ID >= 2000 {
			if strings.Contains(world.Name, "[") {
				euNationalWorldOptions = append(euNationalWorldOptions, option)
			} else {
				euWorldOptions = append(euWorldOptions, option)
			}
		} else {
			naWorldOptions = append(naWorldOptions, option)
		}
	}

	eu := discordgo.SelectMenu{
		CustomID:    InteractionIDSettingsSetWvWWorldEU,
		Placeholder: resources.T("settings.wvw_world.placeholder_eu"),
		Options:     euWorldOptions,
	}
	euNational := discordgo.SelectMenu{
		CustomID:    InteractionIDSettingsSetWvWWorldEUNational,
		Placeholder: resources.T("settings.wvw_world.placeholder_eu_national"),
		Options:     euNationalWorldOptions,
	}
	na := discordgo.SelectMenu{
		CustomID:    InteractionIDSettingsSetWvWWorldNA,
		Placeholder: resources.T("settings.wvw_world.placeholder_na"),
		Options:     naWorldOptions,
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{disable},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{eu},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{euNational},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{na},
		},
	}
}

func buildWorldRoleSelectMenu(currentPrimaryRoleID string, currentLinkedRoleID string) []discordgo.MessageComponent {
	minValues := 0
	primary := discordgo.SelectMenu{
		MenuType:    discordgo.RoleSelectMenu,
		CustomID:    InteractionIDSettingsSetPrimaryWorldRole,
		Placeholder: resources.T("settings.wvw_roles.primary_placeholder"),
		MinValues:   &minValues,
	}
	if currentPrimaryRoleID != "" {
		primary.DefaultValues = []discordgo.SelectMenuDefaultValue{
			{
				Type: discordgo.SelectMenuDefaultValueRole,
				ID:   currentPrimaryRoleID,
			},
		}
	}

	linked := discordgo.SelectMenu{
		MenuType:    discordgo.RoleSelectMenu,
		CustomID:    InteractionIDSettingsSetLinkedWorldRole,
		Placeholder: resources.T("settings.wvw_roles.linked_placeholder"),
		MinValues:   &minValues,
	}
	if currentLinkedRoleID != "" {
		linked.DefaultValues = []discordgo.SelectMenuDefaultValue{
			{
				Type: discordgo.SelectMenuDefaultValueRole,
				ID:   currentLinkedRoleID,
			},
		}
	}

	associated := discordgo.SelectMenu{
		MenuType:    discordgo.RoleSelectMenu,
		CustomID:    InteractionIDSettingsSetWvWAssociatedRoles,
		Placeholder: resources.T("settings.wvw_roles.associated_placeholder"),
		MinValues:   &minValues,
		MaxValues:   25,
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{primary},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{linked},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{associated},
		},
	}
}

func (c *SettingsCmd) buildAccountRepToggle(guildID string) []discordgo.MessageComponent {
	accRepEnabled := c.service.GetSetting(guildID, backend.SettingAccRepEnabled)
	label := resources.T("settings.account_rep.button_enable")
	customID := InteractionIDSettingsSetAccRepEnable
	style := discordgo.SuccessButton
	if accRepEnabled == "true" {
		label = resources.T("settings.account_rep.button_disable")
		customID = InteractionIDSettingsSetAccRepDisable
		style = discordgo.DangerButton
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				&discordgo.Button{
					Label:    label,
					Style:    style,
					CustomID: customID,
				},
			},
		},
	}
}

func (c *SettingsCmd) buildGuildTagRepToggle(guildID string) []discordgo.MessageComponent {
	guildRepEnabled := c.service.GetSetting(guildID, backend.SettingGuildTagRepEnabled)
	label := resources.T("settings.guild_rep.button_enable")
	customID := InteractionIDSettingsSetGuildTagRepEnable
	style := discordgo.SuccessButton
	if guildRepEnabled == "true" {
		label = resources.T("settings.guild_rep.button_disable")
		customID = InteractionIDSettingsSetGuildTagRepDisable
		style = discordgo.DangerButton
	}

	guildRepEnforcement := c.service.GetSetting(guildID, backend.SettingEnforceGuildRep)
	labelEnforcement := resources.T("settings.guild_rep.button_enforce_enable")
	customIDEnforcement := InteractionIDSettingsSetEnforceGuildTagRepEnable
	styleEnforcement := discordgo.SuccessButton
	if guildRepEnforcement == "true" {
		labelEnforcement = resources.T("settings.guild_rep.button_enforce_disable")
		customIDEnforcement = InteractionIDSettingsSetEnforceGuildTagRepDisable
		styleEnforcement = discordgo.DangerButton
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				&discordgo.Button{
					Label:    label,
					Style:    style,
					CustomID: customID,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				&discordgo.Button{
					Label:    labelEnforcement,
					Style:    styleEnforcement,
					CustomID: customIDEnforcement,
				},
			},
		},
	}
}

func (c *SettingsCmd) buildGuildVerificationMenu(roles []*discordgo.Role, currentCommonGuildRole string, currentGuildVerifyRoles []string, currentGuildRolesToRemove []string, currentAPIKeyPermissions []string) []discordgo.MessageComponent {
	zero := 0
	rolesSelect := discordgo.SelectMenu{
		MenuType:    discordgo.RoleSelectMenu,
		CustomID:    InteractionIDSettingsSetGuildCommonRole,
		Placeholder: resources.T("settings.guild_verification.common_role_placeholder"),
		MinValues:   &zero,
	}
	if currentCommonGuildRole != "" {
		rolesSelect.DefaultValues = []discordgo.SelectMenuDefaultValue{
			{
				Type: discordgo.SelectMenuDefaultValueRole,
				ID:   currentCommonGuildRole,
			},
		}
	}

	guildsRoles := c.guilds.GetGuildRolesFrom(roles)
	guildRolesOptions := make([]discordgo.SelectMenuOption, 0, len(guildsRoles))
	for _, guild := range guildsRoles {
		option := discordgo.SelectMenuOption{
			Label: guild.Name,
			Value: guild.ID,
		}
		for _, roleID := range currentGuildVerifyRoles {
			if roleID == guild.ID {
				option.Default = true
				break
			}
		}
		guildRolesOptions = append(guildRolesOptions, option)
	}

	guildRolesSelect := discordgo.SelectMenu{
		MenuType:    discordgo.StringSelectMenu,
		CustomID:    InteractionIDSettingsSetGuildVerifyRoles,
		Placeholder: resources.T("settings.guild_verification.verify_roles_placeholder"),
		MinValues:   &zero,
		MaxValues:   len(guildRolesOptions),
		Options:     guildRolesOptions,
	}

	if len(currentGuildVerifyRoles) > 0 {
		guildRolesSelect.DefaultValues = make([]discordgo.SelectMenuDefaultValue, len(currentGuildVerifyRoles))
		for i, roleID := range currentGuildVerifyRoles {
			guildRolesSelect.DefaultValues[i] = discordgo.SelectMenuDefaultValue{
				ID: roleID,
			}
		}
	}

	permissionsOptions := make([]discordgo.SelectMenuOption, 0, len(Permissions))
	for _, permission := range Permissions {
		option := discordgo.SelectMenuOption{
			Label: permission,
			Value: permission,
		}

		for _, currentPermission := range currentAPIKeyPermissions {
			if currentPermission == permission {
				option.Default = true
				break
			}
		}
		permissionsOptions = append(permissionsOptions, option)
	}

	requiredAPIKeyPermissionsSelect := discordgo.SelectMenu{
		MenuType:    discordgo.StringSelectMenu,
		CustomID:    InteractionIDSettingsSetAPIKeyPermissions,
		Placeholder: resources.T("settings.guild_verification.permissions_placeholder"),
		MinValues:   &zero,
		MaxValues:   len(permissionsOptions),
		Options:     permissionsOptions,
	}

	// Roles to remove when not in in the verified guild
	guildRolesToRemoveOptions := make([]discordgo.SelectMenuOption, 0, len(roles))
	for _, role := range roles {
		option := discordgo.SelectMenuOption{
			Label: role.Name,
			Value: role.ID,
		}
		for _, roleID := range currentGuildRolesToRemove {
			if roleID == role.ID {
				option.Default = true
				break
			}
		}
		guildRolesToRemoveOptions = append(guildRolesToRemoveOptions, option)
	}

	guildRolesToRemoveSelect := discordgo.SelectMenu{
		MenuType:    discordgo.StringSelectMenu,
		CustomID:    InteractionIDSettingsSetRolesToRemoveWhenNotInGuild,
		Placeholder: resources.T("settings.guild_verification.roles_to_remove_placeholder"),
		MinValues:   &zero,
		MaxValues:   len(guildRolesToRemoveOptions),
		Options:     guildRolesToRemoveOptions,
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{rolesSelect},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{guildRolesSelect},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{requiredAPIKeyPermissionsSelect},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{guildRolesToRemoveSelect},
		},
	}
}

func (c *SettingsCmd) InteractSetWvWWorld(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	err := s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		onError(s, event, err)
		return
	}

	if event.GuildID == "" {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: resources.T("settings.errors.server_only"),
		})
		return
	}
	response := resources.T("settings.wvw_world.disabled")
	ctx := context.Background()
	if len(event.MessageComponentData().Values) == 0 {
		// Disable
		zap.L().Info("Disabling WvW world mapping")
		err := c.service.SetSetting(ctx, event.GuildID, backend.SettingWvWWorld, "disabled")
		if err != nil {
			onError(s, event, err)
			return
		}
	} else {
		worldIndexStr := event.MessageComponentData().Values[0]
		worldIndex, err := strconv.Atoi(worldIndexStr)
		if err != nil {
			s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Content: resources.T("settings.errors.invalid_world_index"),
			})
			return
		}

		world := world.WorldNames[worldIndex]
		zap.L().Info("Setting WvW world mapping", zap.String("server_id", event.GuildID), zap.String("world", world.Name))
		err = c.service.SetSetting(ctx, event.GuildID, backend.SettingWvWWorld, worldIndexStr)
		if err != nil {
			onError(s, event, err)
			return
		}
		response = world.Name
	}

	_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
		Content: resources.T("settings.wvw_world.updated", resources.TData("world", response)),
	})
	if err != nil {
		onError(s, event, err)
		return
	}
}

func (c *SettingsCmd) InteractSetWorldRole(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	err := s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		onError(s, event, err)
		return
	}

	if event.GuildID == "" {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: "This command can only be used in a server",
		})
		return
	}

	var property string
	var label string
	switch event.MessageComponentData().CustomID {
	case InteractionIDSettingsSetPrimaryWorldRole:
		property = backend.SettingPrimaryRole
		label = "Primary"
	case InteractionIDSettingsSetLinkedWorldRole:
		property = backend.SettingLinkedRole
		label = "Linked"
	default:
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: resources.T("settings.errors.invalid_role_setting"),
		})
		return
	}

	ctx := context.Background()
	var roleID string
	if len(event.MessageComponentData().Values) == 0 {
		// Disable
		zap.L().Info("Disabling world role", zap.String("server_id", event.GuildID), zap.String("property", property))
		err := c.service.SetSetting(ctx, event.GuildID, property, "")
		if err != nil {
			onError(s, event, err)
			return
		}
	} else {
		roleID = event.MessageComponentData().Values[0]
		zap.L().Info("Setting world role", zap.String("server_id", event.GuildID), zap.String("property", property), zap.String("role_id", roleID))
	}

	err = c.service.SetSetting(ctx, event.GuildID, property, roleID)
	if err != nil {
		onError(s, event, err)
		return
	}

	_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
		Content: resources.T(fmt.Sprintf("settings.wvw_roles.%s_updated", strings.ToLower(label)), resources.TData("roleId", roleID)),
	})
	if err != nil {
		onError(s, event, err)
		return
	}
}

func (c *SettingsCmd) InteractSetAssociatedRoles(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	err := s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		onError(s, event, err)
		return
	}

	if event.GuildID == "" {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: "This command can only be used in a server",
		})
		return
	}

	ctx := context.Background()
	roleIDs := make([]string, len(event.MessageComponentData().Values))

	for i, roleID := range event.MessageComponentData().Values {
		roleIDs[i] = roleID
	}

	err = c.service.SetSetting(ctx, event.GuildID, backend.SettingAssociatedRoles, strings.Join(roleIDs, ","))
	if err != nil {
		onError(s, event, err)
		return
	}

	_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
		Content: resources.T("settings.wvw_roles.associated_updated", resources.TData("roleIds", strings.Join(roleIDs, ">, <@&"))),
	})
	if err != nil {
		onError(s, event, err)
		return
	}
}

func (c *SettingsCmd) InteractSetAccRep(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	if event.GuildID == "" {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: resources.T("settings.errors.server_only"),
		})
		return
	}

	value := "false"
	if event.MessageComponentData().CustomID == InteractionIDSettingsSetAccRepEnable {
		value = "true"
	}

	ctx := context.Background()
	err := c.service.SetSetting(ctx, event.GuildID, backend.SettingAccRepEnabled, value)
	if err != nil {
		onError(s, event, err)
		return
	}

	err = s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    event.Message.Content,
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: c.buildAccountRepToggle(event.GuildID),
		},
	})
	if err != nil {
		onError(s, event, err)
		return
	}
}

func (c *SettingsCmd) InteractSetGuildTagRep(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	if event.GuildID == "" {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: resources.T("settings.errors.server_only"),
		})
		return
	}

	value := "false"
	if event.MessageComponentData().CustomID == InteractionIDSettingsSetGuildTagRepEnable {
		value = "true"
	}

	ctx := context.Background()
	err := c.service.SetSetting(ctx, event.GuildID, backend.SettingGuildTagRepEnabled, value)
	if err != nil {
		onError(s, event, err)
		return
	}

	err = s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    event.Message.Content,
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: c.buildGuildTagRepToggle(event.GuildID),
		},
	})
	if err != nil {
		onError(s, event, err)
		return
	}
}

func (c *SettingsCmd) InteractSetEnforceGuildTagRep(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	if event.GuildID == "" {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: resources.T("settings.errors.server_only"),
		})
		return
	}

	value := "false"
	if event.MessageComponentData().CustomID == InteractionIDSettingsSetEnforceGuildTagRepEnable {
		value = "true"
	}

	ctx := context.Background()
	err := c.service.SetSetting(ctx, event.GuildID, backend.SettingEnforceGuildRep, value)
	if err != nil {
		onError(s, event, err)
		return
	}

	err = s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    event.Message.Content,
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: c.buildGuildTagRepToggle(event.GuildID),
		},
	})
	if err != nil {
		onError(s, event, err)
		return
	}
}

func (c *SettingsCmd) InteractSetGuildCommonRole(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	if event.GuildID == "" {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: resources.T("settings.errors.server_only"),
		})
		return
	}

	if len(event.MessageComponentData().Values) == 0 {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: resources.T("settings.errors.invalid_role_empty"),
		})
		return
	}

	roleID := event.MessageComponentData().Values[0]
	ctx := context.Background()
	err := c.service.SetSetting(ctx, event.GuildID, backend.SettingGuildCommonRole, roleID)
	if err != nil {
		onError(s, event, err)
		return
	}

	menu := event.Message.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.SelectMenu)
	menu.DefaultValues = []discordgo.SelectMenuDefaultValue{
		{
			Type: discordgo.SelectMenuDefaultValueRole,
			ID:   roleID,
		},
	}
	err = s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    event.Message.Content,
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: event.Message.Components,
		},
	})
	if err != nil {
		onError(s, event, err)
		return
	}
}

func (c *SettingsCmd) InteractSetGuildVerifyRoles(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	if event.GuildID == "" {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: resources.T("settings.errors.server_only"),
		})
		return
	}

	roleIds := event.MessageComponentData().Values
	rolesStr := strings.Join(roleIds, ",")
	ctx := context.Background()
	err := c.service.SetSetting(ctx, event.GuildID, backend.SettingGuildVerifyRoles, rolesStr)
	if err != nil {
		onError(s, event, err)
		return
	}

	menu := event.Message.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.SelectMenu)
	menu.Options = []discordgo.SelectMenuOption{}
	roles, err := s.GuildRoles(event.GuildID)
	if err != nil {
		onError(s, event, err)
		return
	}
	guildRoles := c.guilds.GetGuildRolesFrom(roles)
	for _, guild := range guildRoles {
		option := discordgo.SelectMenuOption{
			Label: guild.Name,
			Value: guild.ID,
		}
		for _, roleID := range roleIds {
			if roleID == guild.ID {
				option.Default = true
				break
			}
		}
		menu.Options = append(menu.Options, option)
	}

	err = s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    event.Message.Content,
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: event.Message.Components,
		},
	})
	if err != nil {
		onError(s, event, err)
		return
	}
}

func (c *SettingsCmd) InteractSetRolesToRemoveWhenNotInGuild(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	if event.GuildID == "" {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: resources.T("settings.errors.server_only"),
		})
		return
	}

	roleIds := event.MessageComponentData().Values
	rolesStr := strings.Join(roleIds, ",")
	ctx := context.Background()
	err := c.service.SetSetting(ctx, event.GuildID, backend.SettingRolesToRemoveWhenNotInGuild, rolesStr)
	if err != nil {
		onError(s, event, err)
		return
	}

	menu := event.Message.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.SelectMenu)
	menu.Options = []discordgo.SelectMenuOption{}
	roles, err := s.GuildRoles(event.GuildID)
	if err != nil {
		onError(s, event, err)
		return
	}
	for _, role := range roles {
		option := discordgo.SelectMenuOption{
			Label: role.Name,
			Value: role.ID,
		}
		for _, roleID := range roleIds {
			if roleID == role.ID {
				option.Default = true
				break
			}
		}
		menu.Options = append(menu.Options, option)
	}
	err = s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    event.Message.Content,
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: event.Message.Components,
		},
	})
	if err != nil {
		onError(s, event, err)
		return
	}
}

func (c *SettingsCmd) InteractSetRequiredAPIKeyPermissions(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	if event.GuildID == "" {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: resources.T("settings.errors.server_only"),
		})
		return
	}

	permissions := event.MessageComponentData().Values
	permissionsStr := strings.Join(permissions, ",")
	ctx := context.Background()
	err := c.service.SetSetting(ctx, event.GuildID, backend.SettingGuildRequiredPermissions, permissionsStr)
	if err != nil {
		onError(s, event, err)
		return
	}

	menu := event.Message.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.SelectMenu)
	menu.Options = []discordgo.SelectMenuOption{}
	for _, permission := range Permissions {
		option := discordgo.SelectMenuOption{
			Label: permission,
			Value: permission,
		}
		for _, currentPermission := range permissions {
			if currentPermission == permission {
				option.Default = true
				break
			}
		}
		menu.Options = append(menu.Options, option)
	}

	err = s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    event.Message.Content,
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: event.Message.Components,
		},
	})
	if err != nil {
		onError(s, event, err)
		return
	}
}
