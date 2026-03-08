package interaction

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"github.com/vennekilde/gw2-alliance-bot/internal/backend"
	"github.com/vennekilde/gw2-alliance-bot/resources"
)

var APIKeyErrorRegex = regexp.MustCompile(`(.*)(You need to name your api key ").*(" instead of.*)`)

const tmplAPIKeyInstructions = `Ensure the api key meets the following criteria:

**Name your API key**
` + "`%s%s`" + `

**Permissions**
- Characters
- Progression
- WvW
(See the image below)`

type VerifyCmd struct {
	backend *api.ClientWithResponses
	ui      *UIBuilder
	RepCmd  *RepCmd
}

func NewVerifyCmd(backend *api.ClientWithResponses, ui *UIBuilder, repCmd *RepCmd) *VerifyCmd {
	return &VerifyCmd{
		backend: backend,
		ui:      ui,
		RepCmd:  repCmd,
	}
}

func (c *VerifyCmd) Register(i *Interactions) {
	i.interactions[InteractionIDModalAPIKey] = c.openAPIKeyModal
	i.interactions[InteractionIDSetAPIKey] = c.setAPIKeyModal

	// Verify
	i.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:                     resources.T("cmd.verify.name"),
			Description:              resources.T("cmd.verify.description"),
			NameLocalizations:        resources.GetLocalizations("cmd.verify.name"),
			DescriptionLocalizations: resources.GetLocalizations("cmd.verify.description"),
			// Disabled, as users kept using it wrong
			/*Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "apikey",
					Description: "Verify with your Guild Wars 2 API Key",
				},
			},*/
		},
		handler: func(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
			locale := GetInteractionLocale(event)
			if len(event.ApplicationCommandData().Options) > 0 {
				apiKey := event.ApplicationCommandData().Options[0].StringValue()
				c.setAPIKey(s, event, user, apiKey)
				return
			}
			code := GetAPIKeyCode(2, user.ID)

			var apiKeyNamePrefix string
			for _, guild := range s.State.Guilds {
				if guild.ID == event.GuildID {
					apiKeyNamePrefix = fmt.Sprintf("%s - ", guild.Name)
					break
				}
			}

			embeds := []*discordgo.MessageEmbed{
				{
					Title: resources.TL(locale, "verify.instructions.title"),
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  resources.TL(locale, "verify.instructions.step1.title"),
							Value: resources.TL(locale, "verify.instructions.step1.content", resources.TData("apiKeyPrefix", apiKeyNamePrefix, "code", code)),
						},
						{
							Name:  resources.TL(locale, "verify.instructions.step2.title"),
							Value: resources.TL(locale, "verify.instructions.step2.content"),
						},
					},
					Image: &discordgo.MessageEmbedImage{
						URL: "https://i.imgur.com/Ukgu7KK.png",
					},
				},
			}

			_, err := s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
				Flags:  discordgo.MessageFlagsEphemeral,
				Embeds: embeds,
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								Style: discordgo.LinkButton,
								Label: resources.TL(locale, "verify.buttons.create_api_key"),
								URL:   "https://account.arena.net/applications/create",
							},
							discordgo.Button{
								Style:    discordgo.PrimaryButton,
								Label:    resources.TL(locale, "verify.buttons.set_api_key"),
								CustomID: InteractionIDModalAPIKey,
							},
						},
					},
				},
			})
			if err != nil {
				onError(s, event, err)
			}
		},
	})
}

func (c *VerifyCmd) openAPIKeyModal(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	locale := GetInteractionLocale(event)
	err := s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: InteractionIDSetAPIKey,
			Flags:    discordgo.MessageFlagsEphemeral,
			Title:    resources.TL(locale, "verify.modal.title"),
			Content:  resources.TL(locale, "verify.modal.content"),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							Style:       discordgo.TextInputShort,
							CustomID:    InteractionIDSetAPIKey,
							Label:       resources.TL(locale, "verify.modal.label"),
							MinLength:   72,
							MaxLength:   72,
							Required:    true,
							Placeholder: resources.TL(locale, "verify.modal.placeholder"),
						},
					},
				},
			},
		},
	})
	if err != nil {
		onError(s, event, err)
	}
}

func (c *VerifyCmd) setAPIKeyModal(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	apiKey := event.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	c.setAPIKey(s, event, user, apiKey)
}

func (c *VerifyCmd) setAPIKey(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User, apiKey string) {
	locale := GetInteractionLocale(event)
	ctx := context.Background()
	body := api.APIKeyData{
		Apikey:  apiKey,
		Primary: true,
	}
	resp, err := c.backend.PutPlatformUserAPIKeyWithResponse(ctx, backend.PlatformID, user.ID, nil, body)
	if err != nil {
		onError(s, event, errors.New(resources.TL(locale, "errors.command_execution")))
		return
	}

	switch resp.StatusCode() {
	case 200:
		// skip default handler
	case 201:
		// skip default handler
	case 500:
		// Quick fix for proper apikey name error
		code := GetAPIKeyCode(2, user.ID)

		var apiKeyNamePrefix string
		for _, guild := range s.State.Guilds {
			if guild.ID == event.GuildID {
				apiKeyNamePrefix = fmt.Sprintf("%s - ", guild.Name)
				break
			}
		}

		apiErr := errors.New(APIKeyErrorRegex.ReplaceAllString(resp.JSON500.SafeDisplayError, fmt.Sprintf("${1}\n${2}%s%s${3}", apiKeyNamePrefix, code)))
		onError(s, event, apiErr)
		return
	default:
		onError(s, event, errors.New(resources.TL(locale, "verify.errors.unable_to_set")))
		return
	}

	_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
		Flags: discordgo.MessageFlagsEphemeral,
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       resources.TL(locale, "verify.success.title"),
				Description: resources.TL(locale, "verify.success.description"),
				Color:       0x57F287, // green
				Footer: &discordgo.MessageEmbedFooter{
					Text: resources.TL(locale, "verify.success.footer"),
				},
			},
		},
	})
	if err != nil {
		onError(s, event, err)
		return
	}

	// Check roles
	resp2, err := c.backend.GetPlatformUserWithResponse(ctx, backend.PlatformID, user.ID, &api.GetPlatformUserParams{})
	if err != nil {
		onError(s, event, err)
		return
	} else if resp2.JSON200 == nil {
		onError(s, event, errors.New("unexpected response from the server"))
		return
	}
	c.RepCmd.guildRoleHandler.CheckRoles(event.GuildID, event.Member, event.Member.Roles, resp2.JSON200.Accounts, "")

	// Start guild selection
	c.RepCmd.onCommandRep(s, event, user)
}
