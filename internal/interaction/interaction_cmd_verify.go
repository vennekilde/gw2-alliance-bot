package interaction

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"github.com/vennekilde/gw2-alliance-bot/internal/backend"
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
			Name:        "verify",
			Description: "Add one or more Gw2 API keys to your Discord account. You can link multiple Gw2 accounts.",
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
					Title: "Instructions",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  `1. Click "Create API Key"`,
							Value: fmt.Sprintf(tmplAPIKeyInstructions, apiKeyNamePrefix, code),
						},
						{
							Name:  `2. Click "Set API Key"`,
							Value: "Insert your newly created api key from step 1.",
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
								Label: "Create API Key",
								URL:   "https://account.arena.net/applications/create",
							},
							discordgo.Button{
								Style:    discordgo.PrimaryButton,
								Label:    "Set API Key",
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
	err := s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: InteractionIDSetAPIKey,
			Flags:    discordgo.MessageFlagsEphemeral,
			Title:    "Insert API Key",
			Content:  "Create your api key at at https://account.arena.net/applications/create. Permissions: Characters, Progression & WvW",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							Style:       discordgo.TextInputShort,
							CustomID:    InteractionIDSetAPIKey,
							Label:       "API Key",
							MinLength:   72,
							MaxLength:   72,
							Required:    true,
							Placeholder: "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXXXXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
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
	ctx := context.Background()
	body := api.APIKeyData{
		Apikey:  apiKey,
		Primary: true,
	}
	resp, err := c.backend.PutPlatformUserAPIKeyWithResponse(ctx, backend.PlatformID, user.ID, nil, body)
	if err != nil {
		onError(s, event, fmt.Errorf("error while executing command"))
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
		onError(s, event, errors.New("unable to set api key - reason unknown"))
		return
	}

	_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
		Flags: discordgo.MessageFlagsEphemeral,
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "Success!",
				Description: "Your api key has been added to the system",
				Color:       0x57F287, // green
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Checking if you are eligible to join a guild role...",
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
	c.RepCmd.guildRoleHandler.CheckRoles(event.GuildID, event.Member, resp2.JSON200.Accounts, "")

	// Start guild selection
	c.RepCmd.onCommandRep(s, event, user)
}
