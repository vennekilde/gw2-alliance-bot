package interaction

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"github.com/vennekilde/gw2-alliance-bot/internal/guild"
	"github.com/vennekilde/gw2-alliance-bot/internal/world"
	"github.com/vennekilde/gw2-alliance-bot/resources"
)

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

type UIBuilder struct {
	guilds *guild.Guilds
}

func (ui *UIBuilder) buildStatusFields(user *api.User, locale discordgo.Locale) []*discordgo.MessageEmbedField {
	fields := ui.buildAccountTableFields(user, locale)
	guildsField := ui.buildGuildsField(user.Accounts, locale)
	if guildsField != nil {
		fields = append(fields, guildsField)
	}
	temporaryTableFields := ui.buildTemporaryAccessTableFields(user.EphemeralAssociations, locale)
	if len(temporaryTableFields) > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  resources.TL(locale, "status.fields.temporary_access"),
			Value: resources.TL(locale, "status.temporary_access_description"),
		})
		fields = append(fields, temporaryTableFields...)
	}

	return fields
}

// buildAccountTableFields creates an embed field table of the basic account details
// Example markdown
// Account | World | Status
// --|--|--
// Account.1234 | Far Shiverpeaks | Active
// Account.4321 | Desolation      | Active
func (ui *UIBuilder) buildAccountTableFields(user *api.User, locale discordgo.Locale) []*discordgo.MessageEmbedField {
	var fields []*discordgo.MessageEmbedField

	accColumn := ui.buildAccountNameColumnField(user.Accounts, locale)
	fields = append(fields, accColumn)

	//worldColumn := ui.buildWorldNameColumnField(user.Accounts, locale)
	//fields = append(fields, worldColumn)

	teamColumn := ui.buildTeamNameColumnField(user.Accounts, locale)
	fields = append(fields, teamColumn)

	guildColumn := ui.buildWvWGuildNameColumnField(user.Accounts, locale)
	fields = append(fields, guildColumn)

	//statusColumn := ui.buildStatusColumnField(user.Accounts, user.Bans, locale)
	//fields = append(fields, statusColumn)

	return fields
}

func (ui *UIBuilder) buildAccountNameColumnField(accounts []api.Account, locale discordgo.Locale) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   resources.TL(locale, "status.fields.account"),
		Inline: true,
	}

	for _, account := range accounts {
		if account.ID != "" {
			if field.Value == "" {
				field.Value = account.Name
			} else {
				field.Value += "\n" + account.Name
			}
		}
	}
	return field
}

func (ui *UIBuilder) buildWorldNameColumnField(accounts []api.Account) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   resources.T("status.fields.world"),
		Inline: true,
	}

	for _, account := range accounts {
		if account.ID != "" {
			world := world.WorldNames[account.World]
			if field.Value == "" {
				field.Value = world.Name
			} else {
				field.Value += "\n" + world.Name
			}
		}
	}
	return field
}

func (ui *UIBuilder) buildTeamNameColumnField(accounts []api.Account, locale discordgo.Locale) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   resources.TL(locale, "status.fields.wvw_team"),
		Inline: true,
	}

	for _, account := range accounts {
		if account.ID != "" {
			team, ok := world.TeamNames[account.WvWTeamID]
			if !ok {
				team = world.Team{Name: resources.TL(locale, "status.status_values.unassigned")}
			}
			if field.Value == "" {
				field.Value = team.Name
			} else {
				field.Value += "\n" + team.Name
			}
		}
	}
	return field
}

func (ui *UIBuilder) buildWvWGuildNameColumnField(accounts []api.Account, locale discordgo.Locale) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   resources.TL(locale, "status.fields.wvw_guild"),
		Inline: true,
	}

	for _, account := range accounts {
		if account.ID != "" {
			wvwGuildName := resources.TL(locale, "status.status_values.unassigned")
			if account.WvWGuildID != nil {
				wvwGuild, _ := ui.guilds.GetGuildInfo(*account.WvWGuildID)
				if wvwGuild != nil {
					wvwGuildName = wvwGuild.Name
				}
			}

			if field.Value == "" {
				field.Value = wvwGuildName
			} else {
				field.Value += "\n" + wvwGuildName
			}
		}
	}
	return field
}

func (ui *UIBuilder) buildStatusColumnField(accounts []api.Account, bans []api.Ban, locale discordgo.Locale) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   resources.TL(locale, "status.fields.status"),
		Inline: true,
	}

	activeBan := api.ActiveBan(bans)

	for _, account := range accounts {
		if account.ID != "" {
			statusStr := linkStatus(account.Expired, nil, activeBan != nil, locale)
			if field.Value == "" {
				field.Value = statusStr
			} else {
				field.Value += "\n" + statusStr
			}
		}
	}
	return field
}

func (ui *UIBuilder) buildLastUpdatedColumnField(accounts []api.Account) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   "Updated",
		Inline: true,
	}

	for _, account := range accounts {
		if account.ID != "" {
			lastUpdated := account.DbUpdated
			if lastUpdated.IsZero() {
				continue
			}
			if field.Value == "" {
				field.Value = lastUpdated.Format(time.RFC3339)
			} else {
				field.Value += "\n" + lastUpdated.Format(time.RFC3339)
			}
		}
	}
	return field
}

func (ui *UIBuilder) buildGuildNamesColumnFields(accounts []api.Account) (fields []*discordgo.MessageEmbedField) {
	for _, account := range accounts {
		if account.Guilds != nil {
			guilds, _ := ui.guilds.GetGuildsInfo(account.Guilds)
			for i, guild := range guilds {
				if len(fields) >= i {
					field := &discordgo.MessageEmbedField{
						Inline: true,
						Name:   "\u200B",
					}
					if i == 0 {
						field.Name = "Guilds"
					}
					fields = append(fields, field)
				}

				field := fields[i]
				var name string
				if guild.Name == "" {
					name = fmt.Sprintf("%s - gw2 api error", guild.ID)
				} else {
					name = guild.Name
				}
				if field.Value == "" {
					field.Value = name
				} else {
					field.Value += "\n" + name
				}
			}
		}
	}
	return fields
}

// buildAccountTableFields creates an embed field table of the basic account details
// Example markdown
// World | Expires
// --|--|--
// Far Shiverpeaks | 2 days
// Desolation      | 16 days
func (ui *UIBuilder) buildTemporaryAccessTableFields(ephemeralAssocs []api.EphemeralAssociation, locale discordgo.Locale) []*discordgo.MessageEmbedField {
	var fields []*discordgo.MessageEmbedField

	if len(ephemeralAssocs) == 0 {
		return fields
	}

	accColumn := ui.buildTemporaryWorldNameColumnField(ephemeralAssocs, locale)
	fields = append(fields, accColumn)

	expiresColumn := ui.buildExpiresColumnField(ephemeralAssocs, locale)
	fields = append(fields, expiresColumn)

	return fields
}

func (ui *UIBuilder) buildTemporaryWorldNameColumnField(ephemeralAssocs []api.EphemeralAssociation, locale discordgo.Locale) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   resources.TL(locale, "status.fields.world_temporary"),
		Inline: true,
	}

	for _, ephemeralAssoc := range ephemeralAssocs {
		if ephemeralAssoc.Until != nil && ephemeralAssoc.World != nil {
			world := world.WorldNames[*ephemeralAssoc.World]
			if field.Value == "" {
				field.Value = world.Name
			} else {
				field.Value += "\n" + world.Name
			}
		}
	}
	return field
}

func (ui *UIBuilder) buildExpiresColumnField(ephemeralAssocs []api.EphemeralAssociation, locale discordgo.Locale) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   resources.TL(locale, "status.fields.expires"),
		Inline: true,
	}

	for _, ephemeralAssoc := range ephemeralAssocs {
		if ephemeralAssoc.Until != nil && ephemeralAssoc.World != nil {
			expires := ui.expiresLabel(*ephemeralAssoc.Until, locale)
			if field.Value == "" {
				field.Value = expires
			} else {
				field.Value += "\n" + expires
			}
		}
	}
	return field
}

func (ui *UIBuilder) buildGuildsField(accounts []api.Account, locale discordgo.Locale) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   resources.TL(locale, "status.fields.guilds"),
		Inline: false,
	}

	var guildNames []string
	guildIDSet := map[string]struct{}{}

	for _, account := range accounts {
		if account.Guilds != nil {
			guilds, _ := ui.guilds.GetGuildsInfo(account.Guilds)
			for _, guild := range guilds {
				if _, ok := guildIDSet[guild.ID]; !ok {
					guildIDSet[guild.ID] = struct{}{}
					if guild.Name == "" {
						guildNames = append(guildNames, fmt.Sprintf("%s - gw2 api error", guild.ID))
					} else {
						guildNames = append(guildNames, fmt.Sprintf("[%s] %s", guild.Tag, guild.Name))
					}
				}
			}
		}
	}

	sort.Strings(guildNames)
	for _, guildName := range guildNames {
		if field.Value == "" {
			field.Value = guildName
		} else {
			field.Value += "\n" + guildName
		}
	}

	return field
}

// buildTokensTableEmbeds creates an embeds table of the associated account api tokens
func (ui *UIBuilder) buildTokensTableEmbeds(user *api.User, locale discordgo.Locale) []*discordgo.MessageEmbed {
	var embeds []*discordgo.MessageEmbed
	// fields := ui.buildTokensTableFields(user.Accounts)
	// embeds = append(embeds, &discordgo.MessageEmbed{
	// 	Title:  "Overview",
	// 	Fields: fields,
	// 	Color:  0x3498DB, // blue
	// })
	for _, account := range user.Accounts {
		for _, token := range account.ApiKeys {
			tokenFields := ui.buildTokenTableFields(account, token, locale)
			embeds = append(embeds, &discordgo.MessageEmbed{
				Title:     resources.TL(locale, "apikeys.title", resources.TData("name", token.Name)),
				Fields:    tokenFields,
				Color:     0x3498DB, // blue
				Timestamp: token.LastSuccess.Format(time.RFC3339),
				Footer: &discordgo.MessageEmbedFooter{
					Text: resources.TL(locale, "apikeys.footer"),
				},
			})
		}
	}

	return embeds
}

// buildTokensTableFields creates an embed field table of the associated account api tokens
func (ui *UIBuilder) buildTokenTableFields(acc api.Account, token api.TokenInfo, locale discordgo.Locale) []*discordgo.MessageEmbedField {
	return ui.buildTokenAccountNameColumnField(acc, token, locale)
}

// buildTokensTableFields creates an embed field table of the associated account api tokens
func (ui *UIBuilder) buildTokensTableFields(accounts []api.Account) []*discordgo.MessageEmbedField {
	var fields []*discordgo.MessageEmbedField

	apiKeyColumn := ui.buildAPIKeyNameColumnField(accounts)
	fields = append(fields, apiKeyColumn)

	accColumn := ui.buildAPIKeyAccountNameColumnField(accounts)
	fields = append(fields, accColumn)

	permissionsColumn := ui.buildAPIKeyPermissionColumnField(accounts)
	fields = append(fields, permissionsColumn)

	return fields
}

func (ui *UIBuilder) buildTokenAccountNameColumnField(acc api.Account, token api.TokenInfo, locale discordgo.Locale) []*discordgo.MessageEmbedField {
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   resources.TL(locale, "apikeys.fields.account"),
			Inline: true,
			Value:  acc.Name,
		},
		{
			Name:   resources.TL(locale, "apikeys.fields.permissions"),
			Inline: false,
			Value:  strings.Join(token.Permissions, ", "),
		},
	}
	return fields
}

func (ui *UIBuilder) buildAPIKeyNameColumnField(accounts []api.Account) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   "API Key",
		Inline: true,
	}

	for _, account := range accounts {
		for _, token := range account.ApiKeys {
			if field.Value == "" {
				field.Value = token.Name
			} else {
				field.Value += "\n" + token.Name
			}
		}
	}
	return field
}

func (ui *UIBuilder) buildAPIKeyAccountNameColumnField(accounts []api.Account) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   "Account",
		Inline: true,
	}

	for _, account := range accounts {
		for range account.ApiKeys {
			if field.Value == "" {
				field.Value = account.Name
			} else {
				field.Value += "\n" + account.Name
			}
		}
	}
	return field
}

func (ui *UIBuilder) buildAPIKeyPermissionColumnField(accounts []api.Account) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   "Permissions",
		Inline: true,
	}

	for _, account := range accounts {
		for _, token := range account.ApiKeys {
			if field.Value == "" {
				field.Value = strings.Join(token.Permissions, ", ")
			} else {
				field.Value += "\n" + strings.Join(token.Permissions, ", ")
			}
		}
	}
	return field
}

func (ui *UIBuilder) spacerField() *discordgo.MessageEmbedField {
	return &discordgo.MessageEmbedField{
		Name:  "\u200B",
		Value: "\u200B",
	}
}

func (ui *UIBuilder) expiresLabel(expires time.Time, locale discordgo.Locale) (label string) {
	until := time.Until(expires)
	if until.Hours() > 24 {
		label = resources.TL(locale, "status.expires_format.days", resources.TData("days", int(until.Hours()/24)))
	} else {
		label = until.String()
	}
	return label
}

func linkStatus(isExpired *bool, validUntil *time.Time, banned bool, locale discordgo.Locale) string {
	if banned {
		return resources.TL(locale, "status.status_values.banned")
	}

	valid := true
	if validUntil != nil {
		valid := validUntil.Before(time.Now())
		if valid {
			return resources.TL(locale, "status.status_values.temporary")
		}
	}
	if (isExpired != nil && *isExpired) || !valid {
		return resources.TL(locale, "status.status_values.not_linked")
	}

	return resources.TL(locale, "status.status_values.active")
}

func linkStatusColor(isExpired *bool, validUntil *time.Time, banned bool, good int, bad int, neutral int) int {
	if banned {
		return bad
	}

	valid := true
	if validUntil != nil {
		valid := validUntil.Before(time.Now())
		if valid {
			return neutral
		}
	}
	if (isExpired != nil && *isExpired) || !valid {
		return bad
	}

	return good
}
