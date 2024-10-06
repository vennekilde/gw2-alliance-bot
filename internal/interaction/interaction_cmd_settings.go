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
	InteractionIDSettingsSetWvWWorldDisable           = "setting-set-wvw-world-disable"
	InteractionIDSettingsSetWvWWorldEU                = "setting-set-wvw-world-eu"
	InteractionIDSettingsSetWvWWorldEUNational        = "setting-set-wvw-world-eu-national"
	InteractionIDSettingsSetWvWWorldNA                = "setting-set-wvw-world-na"
	InteractionIDSettingsSetPrimaryWorldRole          = "setting-set-prmary-world-role"
	InteractionIDSettingsSetLinkedWorldRole           = "setting-set-linked-world-role"
	InteractionIDSettingsSetWvWAssociatedRoles        = "setting-set-wvw-associated-roles"
	InteractionIDSettingsSetAccRepEnable              = "setting-set-acc-rep-enable"
	InteractionIDSettingsSetAccRepDisable             = "setting-set-acc-rep-disable"
	InteractionIDSettingsSetGuildTagRepEnable         = "setting-set-guild-tag-rep-enable"
	InteractionIDSettingsSetGuildTagRepDisable        = "setting-set-guild-tag-rep-disable"
	InteractionIDSettingsSetEnforceGuildTagRepEnable  = "setting-set-enforce-guild-tag-rep-enable"
	InteractionIDSettingsSetEnforceGuildTagRepDisable = "setting-set-enforce-guild-tag-rep-disable"
	InteractionIDSettingsSetGuildCommonRole           = "setting-set-guild-common-role"
	InteractionIDSettingsSetGuildVerifyRoles          = "setting-set-guild-verify-roles"
	InteractionIDSettingsSetAPIKeyPermissions         = "setting-set-api-key-permissions"
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
	i.interactions[InteractionIDSettingsSetAPIKeyPermissions] = c.InteractSetRequiredAPIKeyPermissions

	var permission int64 = discordgo.PermissionAdministrator
	var permissionDM bool = false

	// Settings command
	i.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:                     "settings",
			Description:              "Modify settings for the Guild Wars 2 Alliance Bot",
			DefaultMemberPermissions: &permission,
			DMPermission:             &permissionDM,
		},
		handler: func(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
			//ctx := context.Background()
			currentWorld := c.service.GetSetting(event.GuildID, backend.SettingWvWWorld)
			currentWorldID, _ := strconv.Atoi(currentWorld)
			wvwWorldSelectComponents := buildWvWWorldSelectMenu(currentWorldID)

			_, err := s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Content:    "WvW World Settings",
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
				Content:    "WvW World Roles",
				Flags:      discordgo.MessageFlagsEphemeral,
				Components: worldRoleSelectComponents,
			})
			if err != nil {
				onError(s, event, err)
			}

			accRepComponents := c.buildAccountRepToggle(event.GuildID)
			_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Content:    "Account rep will append the account name to a user's nickname",
				Flags:      discordgo.MessageFlagsEphemeral,
				Components: accRepComponents,
			})
			if err != nil {
				onError(s, event, err)
			}

			guildTagRepComponents := c.buildGuildTagRepToggle(event.GuildID)
			_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Content:    "Guild representation settings",
				Flags:      discordgo.MessageFlagsEphemeral,
				Components: guildTagRepComponents,
			})
			if err != nil {
				onError(s, event, err)
			}

			currentCommonGuildRole := c.service.GetSetting(event.GuildID, backend.SettingGuildCommonRole)
			currentRequiredPermissions := c.service.GetSettingSlice(event.GuildID, backend.SettingGuildRequiredPermissions)
			currentGuildVerifyRoles := c.service.GetSettingSlice(event.GuildID, backend.SettingGuildVerifyRoles)
			roles, err := s.GuildRoles(event.GuildID)
			if err != nil {
				onError(s, event, err)
			}

			guildCommonRoleComponents := c.buildGuildVerificationMenu(roles, currentCommonGuildRole, currentGuildVerifyRoles, currentRequiredPermissions)
			_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Content:    "The common guild role will be added to all users that are also in a guild role",
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
		Label:    "Disable",
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
		Placeholder: "Select a world from EU",
		Options:     euWorldOptions,
	}
	euNational := discordgo.SelectMenu{
		CustomID:    InteractionIDSettingsSetWvWWorldEUNational,
		Placeholder: "Select a national world from EU",
		Options:     euNationalWorldOptions,
	}
	na := discordgo.SelectMenu{
		CustomID:    InteractionIDSettingsSetWvWWorldNA,
		Placeholder: "Select a world from NA",
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
		Placeholder: "Select a role for primary world",
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
		Placeholder: "Select a role for linked world",
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
		Placeholder: "Select associated roles that will be removed if user is not in the primary or linked world",
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
	label := "Enable"
	customID := InteractionIDSettingsSetAccRepEnable
	style := discordgo.SuccessButton
	if accRepEnabled == "true" {
		label = "Disable"
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
	label := "Prepend guild tag to nickname"
	customID := InteractionIDSettingsSetGuildTagRepEnable
	style := discordgo.SuccessButton
	if guildRepEnabled == "true" {
		label = "Disable prepend guild tag to nickname"
		customID = InteractionIDSettingsSetGuildTagRepDisable
		style = discordgo.DangerButton
	}

	guildRepEnforcement := c.service.GetSetting(guildID, backend.SettingEnforceGuildRep)
	labelEnforcement := "Enforce Guild Rep"
	customIDEnforcement := InteractionIDSettingsSetEnforceGuildTagRepEnable
	styleEnforcement := discordgo.SuccessButton
	if guildRepEnforcement == "true" {
		labelEnforcement = "Disable Enforcement"
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

func (c *SettingsCmd) buildGuildVerificationMenu(roles []*discordgo.Role, currentCommonGuildRole string, currentGuildVerifyRoles []string, currentAPIKeyPermissions []string) []discordgo.MessageComponent {
	zero := 0
	rolesSelect := discordgo.SelectMenu{
		MenuType:    discordgo.RoleSelectMenu,
		CustomID:    InteractionIDSettingsSetGuildCommonRole,
		Placeholder: "Select a common guild role",
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
		Placeholder: "Select guilds that will be verified with the common role",
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
		Placeholder: "Select required API key permissions",
		MinValues:   &zero,
		MaxValues:   len(permissionsOptions),
		Options:     permissionsOptions,
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
			Content: "This command can only be used in a server",
		})
		return
	}
	response := "disabled"
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
				Content: "Invalid world index",
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
		Content: fmt.Sprintf("WvW world mapping updated to %s", response),
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
			Content: "Invalid role setting",
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
		Content: fmt.Sprintf("%s world role updated to <@&%s>", label, roleID),
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
		Content: fmt.Sprintf("Associated wvw roles updated to <@&%s>", strings.Join(roleIDs, ">, <@&")),
	})
	if err != nil {
		onError(s, event, err)
		return
	}
}

func (c *SettingsCmd) InteractSetAccRep(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	if event.GuildID == "" {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: "This command can only be used in a server",
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
			Content: "This command can only be used in a server",
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
			Content: "This command can only be used in a server",
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
			Content: "This command can only be used in a server",
		})
		return
	}

	if len(event.MessageComponentData().Values) == 0 {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: "Invalid role (empty)",
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
			Content: "This command can only be used in a server",
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

func (c *SettingsCmd) InteractSetRequiredAPIKeyPermissions(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	if event.GuildID == "" {
		s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
			Content: "This command can only be used in a server",
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
