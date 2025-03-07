package tickets

import (
	"errors"
	"fmt"
	permcache "github.com/TicketsBot/common/permission"
	"github.com/TicketsBot/database"
	"github.com/TicketsBot/worker/bot/command"
	"github.com/TicketsBot/worker/bot/command/context"
	"github.com/TicketsBot/worker/bot/command/registry"
	"github.com/TicketsBot/worker/bot/customisation"
	"github.com/TicketsBot/worker/bot/dbclient"
	"github.com/TicketsBot/worker/bot/logic"
	"github.com/TicketsBot/worker/bot/utils"
	"github.com/TicketsBot/worker/i18n"
	"github.com/rxdn/gdl/objects/channel/message"
	"github.com/rxdn/gdl/objects/interaction"
	"strings"
)

type StartTicketCommand struct {
}

func (StartTicketCommand) Properties() registry.Properties {
	return registry.Properties{
		Name:             "Start Ticket",
		Type:             interaction.ApplicationCommandTypeMessage,
		PermissionLevel:  permcache.Everyone, // Customisable level
		Category:         command.Tickets,
		InteractionOnly:  true,
		DefaultEphemeral: true,
	}
}

func (c StartTicketCommand) GetExecutor() interface{} {
	return c.Execute
}

func (StartTicketCommand) Execute(ctx registry.CommandContext) {
	interaction, ok := ctx.(*context.SlashCommandContext)
	if !ok {
		return
	}

	settings, err := dbclient.Client.Settings.Get(ctx.GuildId())
	if err != nil {
		ctx.HandleError(err)
		return
	}

	userPermissionLevel, err := ctx.UserPermissionLevel()
	if err != nil {
		ctx.HandleError(err)
		return
	}

	if userPermissionLevel < permcache.PermissionLevel(settings.ContextMenuPermissionLevel) {
		ctx.Reply(customisation.Red, i18n.Error, i18n.MessageNoPermission)
		return
	}

	messageId := interaction.Interaction.Data.TargetId

	msg, ok := interaction.ResolvedMessage(messageId)
	if err != nil {
		ctx.HandleError(errors.New("Message missing from resolved data"))
		return
	}

	var panel *database.Panel
	if settings.ContextMenuPanel != nil {
		p, err := dbclient.Client.Panel.GetById(*settings.ContextMenuPanel)
		if err != nil {
			ctx.HandleError(err)
			return
		}

		panel = &p
	}

	ticket, err := logic.OpenTicket(ctx, panel, msg.Content, nil)
	if err != nil {
		// Already handled
		return
	}

	if ticket.ChannelId != nil {
		sendTicketStartedFromMessage(ctx, ticket, msg)

		if settings.ContextMenuAddSender {
			addMessageSender(ctx, ticket, msg)
			sendMovedMessage(ctx, ticket, msg)
			if err := dbclient.Client.TicketMembers.Add(ticket.GuildId, ticket.Id, msg.Author.Id); err != nil {
				ctx.HandleError(err)
				return
			}
		}
	}
}

// Send info message
func sendTicketStartedFromMessage(ctx registry.CommandContext, ticket database.Ticket, msg message.Message) {
	// format
	messageLink := fmt.Sprintf("https://discord.com/channels/%d/%d/%d", ctx.GuildId(), ctx.ChannelId(), msg.Id)
	contentFormatted := strings.ReplaceAll(utils.StringMax(msg.Content, 2048, "..."), "`", "\\`")

	msgEmbed := utils.BuildEmbed(
		ctx, customisation.Green, i18n.Ticket, i18n.MessageTicketStartedFrom, nil,
		messageLink, msg.Author.Id, ctx.ChannelId(), contentFormatted,
	)

	if _, err := ctx.Worker().CreateMessageEmbed(*ticket.ChannelId, msgEmbed); err != nil {
		ctx.HandleError(err)
		return
	}
}

func addMessageSender(ctx registry.CommandContext, ticket database.Ticket, msg message.Message) {
	// If the sender was the ticket opener, or staff, they already have access
	// However, support teams makes this tricky
	if msg.Author.Id == ticket.UserId {
		return
	}

	// Get perms
	ch, err := ctx.Worker().GetChannel(*ticket.ChannelId)
	if err != nil {
		ctx.HandleError(err)
		return
	}

	for _, overwrite := range ch.PermissionOverwrites {
		// Check if already present
		if overwrite.Id == msg.Author.Id {
			return
		}
	}

	// Build permissions
	additionalPermissions, err := dbclient.Client.TicketPermissions.Get(ctx.GuildId())
	if err != nil {
		ctx.HandleError(err)
		return
	}

	overwrite := logic.BuildUserOverwrite(msg.Author.Id, additionalPermissions)
	if err := ctx.Worker().EditChannelPermissions(*ticket.ChannelId, overwrite); err != nil {
		ctx.HandleError(err)
		return
	}
}

func sendMovedMessage(ctx registry.CommandContext, ticket database.Ticket, msg message.Message) {
	reference := &message.MessageReference{
		MessageId:       msg.Id,
		ChannelId:       ctx.ChannelId(),
		GuildId:         ctx.GuildId(),
		FailIfNotExists: false,
	}

	msgEmbed := utils.BuildEmbed(ctx, customisation.Green, i18n.Ticket, i18n.MessageMovedToTicket, nil, *ticket.ChannelId)

	if _, err := ctx.Worker().CreateMessageEmbedReply(msg.ChannelId, msgEmbed, reference); err != nil {
		ctx.HandleError(err)
		return
	}
}
