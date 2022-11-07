package internal

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/bwmarrin/discordgo"
	"github.com/vennekilde/gw2-alliance-bot/internal/api/goraml"
	"github.com/vennekilde/gw2-alliance-bot/internal/api/types"
)

var APIKeyErrorRegex = regexp.MustCompile(`(.*)(You need to name your api key ").*(" instead of.*)`)

func (c *Interactions) registerInteractionVerify() {
	c.interactions[InteractionIDModalAPIKey] = c.openAPIKeyModal
	c.interactions[InteractionIDSetAPIKey] = c.setAPIKey

	// Verify
	c.addCommand(&Command{
		command: &discordgo.ApplicationCommand{
			Name:        "verify",
			Description: "Verify with your Guild Wars 2 API Key",
		},
		handler: func(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
			code := GetAPIKeyCode(2, user.ID)

			var guild *discordgo.Guild
			for _, guild = range c.discord.State.Guilds {
				if guild.ID == event.GuildID {
					break
				}
			}

			status, _, err := c.backend.V1.V1UsersService_idService_user_idVerificationStatusGet(user.ID, serviceID, map[string]interface{}{}, map[string]interface{}{})
			if err != nil {
				c.onError(s, event, err)
			}

			embeds := []*discordgo.MessageEmbed{
				{
					Title: "Instructions",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name: `1. Click "Create API Key"`,
							Value: fmt.Sprintf(`Ensure the api key meets the following criteria:

									**Name your API key**
									%s - %s

									**Permissions**
									- Characters
									- Progression
									(See the image below)`, guild.Name, code),
						},
						{
							Name:  `2. Click "Set API Key"`,
							Value: "Insert your newly created api key from step 1.",
						},
					},
					Image: &discordgo.MessageEmbedImage{
						URL: "https://i.imgur.com/8CIsUhH.png",
					},
				},
			}

			_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
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
				c.onError(s, event, err)
			}

			if status.Status != types.EnumVerificationStatusStatusACCESS_DENIED_ACCOUNT_NOT_LINKED {
				var memberName string
				if event.Member.Nick != "" {
					memberName = event.Member.Nick
				} else {
					memberName = event.Member.User.Username
				}
				fields := c.buildStatusFields(memberName, &status)

				embeds = []*discordgo.MessageEmbed{
					{
						Title: "Already Verified!",
						Color: 0x57F287, // green
						Fields: append(
							[]*discordgo.MessageEmbedField{
								{
									Name:  "Your Discord account is already linked with a Guild Wars 2 account",
									Value: `If you wish the change to another Guild Wars 2 account, you can follow the instructions posted above`,
								},
							},
							fields...,
						),
					},
				}

				repComponents, _ := c.buildGuildComponents(event.GuildID, &status)
				if len(repComponents) > 0 {
					embeds = append(embeds,
						&discordgo.MessageEmbed{
							Title: "You can represent a guild on this server!",
							Color: 0x3498DB, // blue
							Fields: []*discordgo.MessageEmbedField{
								{
									Name:  `Your Guild Wars 2 account is in a guild represented on this server`,
									Value: `Click on the guild you wish to represent below`,
								},
							},
						})
					// Ensure user is in the verified group
					err = c.guildRoleHandler.AddVerificationRole(event.GuildID, user.ID)
					if err != nil {
						c.onError(s, event, err)
						return
					}
				}

				_, err = s.FollowupMessageCreate(event.Interaction, false, &discordgo.WebhookParams{
					Flags:  discordgo.MessageFlagsEphemeral,
					Embeds: embeds,
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: repComponents,
						},
					},
				})
				if err != nil {
					c.onError(s, event, err)
				}
			}
		},
	})
}

func (c *Interactions) openAPIKeyModal(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	err := s.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: InteractionIDSetAPIKey,
			Flags:    discordgo.MessageFlagsEphemeral,
			Title:    "Insert API Key",
			Content:  "Create your api key at at https://account.arena.net/applications/create. Permissions: Characters & Progression",
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
		c.onError(s, event, err)
	}
}

func (c *Interactions) setAPIKey(s *discordgo.Session, event *discordgo.InteractionCreate, user *discordgo.User) {
	apiKey := event.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	body := types.APIKeyData{
		Apikey:  apiKey,
		Primary: true,
	}
	resp, err := c.backend.V1.V1UsersService_idService_user_idApikeyPut(user.ID, serviceID, body, map[string]interface{}{}, map[string]interface{}{})
	if err != nil {
		if apiErr, ok := err.(goraml.APIError); ok {
			if apiErr.Code != 200 {
				if typedErr, ok := apiErr.Message.(*types.Error); ok {
					// Quick fix for proper apikey name error
					code := GetAPIKeyCode(2, user.ID)

					var guild *discordgo.Guild
					for _, guild = range c.discord.State.Guilds {
						if guild.ID == event.GuildID {
							break
						}
					}

					typedErr.SafeDisplayError = APIKeyErrorRegex.ReplaceAllString(typedErr.SafeDisplayError, fmt.Sprintf("${1}\n${2}%s - %s${3}", guild.Name, code))
				}
				c.onError(s, event, apiErr)
				return
			}
		} else {
			c.onError(s, event, fmt.Errorf("error while executing command"))
			return
		}
	}

	switch resp.StatusCode {
	case 200:
		// skip default handler
	default:
		c.onError(s, event, errors.New("unable to set api key - reason unknown"))
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
		c.onError(s, event, err)
		return
	}

	c.onCommandRep(s, event, user)
}
