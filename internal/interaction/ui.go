package interaction

import (
	"fmt"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"github.com/vennekilde/gw2-alliance-bot/internal/guild"
	"github.com/vennekilde/gw2-alliance-bot/internal/world"
)

type UIBuilder struct {
	guilds *guild.Guilds
}

func (ui *UIBuilder) buildStatusFields(user *api.User) []*discordgo.MessageEmbedField {
	fields := ui.buildAccountTableFields(user)
	guildsField := ui.buildGuildsField(user.Accounts)
	if guildsField != nil {
		fields = append(fields, guildsField)
	}
	temporaryTableFields := ui.buildTemporaryAccessTableFields(user.EphemeralAssociations)
	if len(temporaryTableFields) > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Temporary Access",
			Value: "Worlds you have been granted a temporary access to",
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
func (ui *UIBuilder) buildAccountTableFields(user *api.User) []*discordgo.MessageEmbedField {
	var fields []*discordgo.MessageEmbedField

	accColumn := ui.buildAccountNameColumnField(user.Accounts)
	fields = append(fields, accColumn)

	worldColumn := ui.buildWorldNameColumnField(user.Accounts)
	fields = append(fields, worldColumn)

	statusColumn := ui.buildStatusColumnField(user.Accounts, user.Bans)
	fields = append(fields, statusColumn)

	return fields
}

func (ui *UIBuilder) buildAccountNameColumnField(accounts []api.Account) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   "Account",
		Inline: true,
	}

	for _, account := range accounts {
		if account.Id != "" {
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
		Name:   "World",
		Inline: true,
	}

	for _, account := range accounts {
		if account.Id != "" {
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

func (ui *UIBuilder) buildStatusColumnField(accounts []api.Account, bans []api.Ban) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   "Status",
		Inline: true,
	}

	activeBan := api.ActiveBan(bans)

	for _, account := range accounts {
		if account.Id != "" {
			statusStr := linkStatus(account.Expired, nil, activeBan != nil)
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
		if account.Id != "" {
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
			guilds := ui.guilds.GetGuildInfo(account.Guilds)
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
				if field.Value == "" {
					field.Value = guild.Name
				} else {
					field.Value += "\n" + guild.Name
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
func (ui *UIBuilder) buildTemporaryAccessTableFields(ephemeralAssocs []api.EphemeralAssociation) []*discordgo.MessageEmbedField {
	var fields []*discordgo.MessageEmbedField

	if len(ephemeralAssocs) == 0 {
		return fields
	}

	accColumn := ui.buildTemporaryWorldNameColumnField(ephemeralAssocs)
	fields = append(fields, accColumn)

	expiresColumn := ui.buildExpiresColumnField(ephemeralAssocs)
	fields = append(fields, expiresColumn)

	return fields
}

func (ui *UIBuilder) buildTemporaryWorldNameColumnField(ephemeralAssocs []api.EphemeralAssociation) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   "World (Temporary)",
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

func (ui *UIBuilder) buildExpiresColumnField(ephemeralAssocs []api.EphemeralAssociation) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   "Expires",
		Inline: true,
	}

	for _, ephemeralAssoc := range ephemeralAssocs {
		if ephemeralAssoc.Until != nil && ephemeralAssoc.World != nil {
			expires := ui.expiresLabel(*ephemeralAssoc.Until)
			if field.Value == "" {
				field.Value = expires
			} else {
				field.Value += "\n" + expires
			}
		}
	}
	return field
}

func (ui *UIBuilder) buildGuildsField(accounts []api.Account) *discordgo.MessageEmbedField {
	field := &discordgo.MessageEmbedField{
		Name:   "Guilds",
		Inline: true,
	}

	var guildNames []string
	guildIDSet := map[string]struct{}{}

	for _, account := range accounts {
		if account.Guilds != nil {
			guilds := ui.guilds.GetGuildInfo(account.Guilds)
			for _, guild := range guilds {
				if _, ok := guildIDSet[guild.ID]; !ok {
					guildIDSet[guild.ID] = struct{}{}
					guildNames = append(guildNames, fmt.Sprintf("[%s] %s", guild.Tag, guild.Name))
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

func (ui *UIBuilder) spacerField() *discordgo.MessageEmbedField {
	return &discordgo.MessageEmbedField{
		Name:  "\u200B",
		Value: "\u200B",
	}
}

func (ui *UIBuilder) expiresLabel(expires time.Time) (label string) {
	until := time.Until(expires)
	if until.Hours() > 24 {
		label = fmt.Sprintf("%d days", int(until.Hours()/24))
	} else {
		label = until.String()
	}
	return label
}

func linkStatus(isExpired *bool, validUntil *time.Time, banned bool) string {
	if banned {
		return "Banned"
	}

	valid := true
	if validUntil != nil {
		valid := validUntil.Before(time.Now())
		if valid {
			return "Temporary"
		}
	}
	if (isExpired != nil && *isExpired) || !valid {
		return "Not linked with Guild Wars 2 account!\nType /verify to link with your Guild Wars 2 account"
	}

	return "Active"
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
