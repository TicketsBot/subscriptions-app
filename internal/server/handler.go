package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/pkg/errors"
	"github.com/rxdn/gdl/objects/channel/embed"
	"github.com/rxdn/gdl/objects/channel/message"
	"github.com/rxdn/gdl/objects/interaction"
	"github.com/rxdn/gdl/objects/user"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

func (s *Server) HandleInteraction(ctx *gin.Context) {
	var body interaction.Interaction
	if err := ctx.ShouldBindBodyWith(&body, binding.JSON); err != nil {
		ctx.JSON(400, errorJson("Failed to parse body"))
		return
	}

	switch body.Type {
	case interaction.InteractionTypePing:
		ctx.JSON(200, interaction.NewResponsePong())
	case interaction.InteractionTypeApplicationCommand:
		var commandData interaction.ApplicationCommandInteraction
		if err := ctx.ShouldBindBodyWith(&commandData, binding.JSON); err != nil {
			_ = ctx.Error(errors.Wrap(err, "Failed to parse application command payload"))
			return
		}

		res := handleCommand(s, commandData)
		ctx.JSON(http.StatusOK, res)
	default:
		_ = ctx.Error(fmt.Errorf("interaction type %d not implemented", body.Type))
	}
}

const (
	red  = 0xeb4034
	blue = 0x4287f5
)

func handleCommand(s *Server, data interaction.ApplicationCommandInteraction) interaction.ResponseChannelMessage {
	command := data.Data

	if !contains(s.config.Discord.AllowedGuilds, data.GuildId.Value) {
		return interaction.NewResponseChannelMessage(interaction.ApplicationCommandCallbackData{
			Content: "This guild is not in the allowed guilds list",
			Flags:   uint(message.FlagEphemeral),
		})
	}

	switch command.Name {
	case "lookup":
		if len(command.Options) == 0 || command.Options[0].Name != "email" {
			return interaction.NewResponseChannelMessage(interaction.ApplicationCommandCallbackData{
				Content: "Missing email",
				Flags:   uint(message.FlagEphemeral),
			})
		}

		email, ok := command.Options[0].Value.(string)
		if !ok {
			return interaction.NewResponseChannelMessage(interaction.ApplicationCommandCallbackData{
				Content: "Email was wrong type",
				Flags:   uint(message.FlagEphemeral),
			})
		}

		s.mu.RLock()
		hasInitialData := s.pledges != nil
		patron, ok := s.pledges[email]
		s.mu.RUnlock()

		if !hasInitialData {
			return interaction.NewResponseChannelMessage(interaction.ApplicationCommandCallbackData{
				Content: "Initial data not loaded yet, please try again in a few minutes",
				Flags:   uint(message.FlagEphemeral),
			})
		}

		var user user.User
		if data.Member != nil {
			user = data.Member.User
		} else if data.User != nil {
			user = *data.User
		} // Other should be infallible

		if ok {
			tiers := make([]string, len(patron.Tiers))
			for i, tier := range patron.Tiers {
				tiers[i] = tier.String()
			}

			discord := "Not linked"
			if patron.DiscordId != nil {
				discord = fmt.Sprintf("<@%d> (%d)", *patron.DiscordId, *patron.DiscordId)
			}

			return interaction.NewResponseChannelMessage(interaction.ApplicationCommandCallbackData{
				Embeds: []*embed.Embed{
					{
						Title:     "Account Found",
						Url:       fmt.Sprintf("https://www.patreon.com/user?u=%d", patron.Id),
						Timestamp: ptr(time.Now()),
						Color:     blue,
						Author: &embed.EmbedAuthor{
							Name:    user.Username,
							IconUrl: user.AvatarUrl(256),
						},
						Fields: []*embed.EmbedField{
							{
								Name:   "Status",
								Value:  patron.Attributes.PatronStatus,
								Inline: true,
							},
							{
								Name:   "Last Charge Status",
								Value:  patron.Attributes.LastChargeStatus,
								Inline: true,
							},
							{
								Name:   "Last Charge Date",
								Value:  fmt.Sprintf("<t:%d>", patron.Attributes.LastChargeDate.Unix()),
								Inline: true,
							},
							{
								Name:   "Join Date",
								Value:  fmt.Sprintf("<t:%d>", patron.Attributes.PledgeRelationshipStart.Unix()),
								Inline: true,
							},
							{
								Name:   "Active Tiers",
								Value:  strings.Join(tiers, ", "),
								Inline: true,
							},
							{
								Name:   "Discord Account",
								Value:  discord,
								Inline: true,
							},
						},
					},
				},
			})
		} else {
			return interaction.NewResponseChannelMessage(interaction.ApplicationCommandCallbackData{
				Embeds: []*embed.Embed{
					{
						Title:       "Account Not Found",
						Description: fmt.Sprintf("No Patreon account with email `%s` found", email),
						Timestamp:   ptr(time.Now()),
						Color:       red,
						Author: &embed.EmbedAuthor{
							Name:    user.Username,
							IconUrl: user.AvatarUrl(256),
						},
					},
				},
			})
		}
	default:
		s.logger.Warn("Unknown command", zap.String("command", command.Name))
		return interaction.NewResponseChannelMessage(interaction.ApplicationCommandCallbackData{
			Content: "Unknown command",
			Flags:   uint(message.FlagEphemeral),
		})
	}
}
