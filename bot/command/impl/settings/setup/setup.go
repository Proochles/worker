package setup

import (
	"context"
	"github.com/TicketsBot/common/permission"
	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/worker/bot/command"
	"github.com/TicketsBot/worker/bot/command/registry"
	"github.com/TicketsBot/worker/bot/customisation"
	"github.com/TicketsBot/worker/i18n"
	"github.com/rxdn/gdl/objects/channel/embed"
	"github.com/rxdn/gdl/objects/interaction"
	"golang.org/x/sync/errgroup"
)

type SetupCommand struct {
}

func (SetupCommand) Properties() registry.Properties {
	return registry.Properties{
		Name:            "setup",
		Description:     i18n.HelpSetup,
		Type:            interaction.ApplicationCommandTypeChatInput,
		PermissionLevel: permission.Admin,
		Category:        command.Settings,
		Children: []registry.Command{
			AutoSetupCommand{},
			PrefixSetupCommand{},
			WelcomeMessageSetupCommand{},
			LimitSetupCommand{},
			TranscriptsSetupCommand{},
			CategorySetupCommand{},
		},
	}
}

func (c SetupCommand) GetExecutor() interface{} {
	return c.Execute
}

func (c SetupCommand) Execute(ctx registry.CommandContext) {
	ctx.ReplyWithFieldsPermanent(customisation.Green, i18n.TitleSetup, i18n.SetupChoose, c.buildFields(ctx))
}

func (SetupCommand) buildFields(ctx registry.CommandContext) []embed.EmbedField {
	fields := make([]embed.EmbedField, 9)

	group, _ := errgroup.WithContext(context.Background())

	group.Go(getFieldFunc(ctx, fields, 0, "t!setup auto", i18n.SetupAutoDescription, true))
	group.Go(getFieldFunc(ctx, fields, 1, "Dashboard", i18n.SetupDashboardDescription, true))
	fields[2] = embed.EmbedField{
		Name:   "\u200b",
		Value:  "‎",
		Inline: true,
	}
	group.Go(getFieldFunc(ctx, fields, 3, "t!setup prefix", i18n.SetupPrefixDescription, true))
	group.Go(getFieldFunc(ctx, fields, 4, "t!setup limit", i18n.SetupLimitDescription, true))
	group.Go(getFieldFunc(ctx, fields, 5, "t!setup welcomemessage", i18n.SetupWelcomeMessageDescription, false))
	group.Go(getFieldFunc(ctx, fields, 6, "t!setup transcripts", i18n.SetupTranscriptsDescription, true))
	group.Go(getFieldFunc(ctx, fields, 7, "t!setup category", i18n.SetupCategoryDescription, true))
	group.Go(getFieldFunc(ctx, fields, 8, "Reaction Panels", i18n.SetupReactionPanelsDescription, false, ctx.GuildId))

	// should never happen
	if err := group.Wait(); err != nil {
		sentry.Error(err)
		return nil
	}

	return fields
}

func newFieldFromTranslation(ctx registry.CommandContext, name string, value i18n.MessageId, inline bool, format ...interface{}) embed.EmbedField {
	return embed.EmbedField{
		Name:   name,
		Value:  i18n.GetMessageFromGuild(ctx.GuildId(), value, format...),
		Inline: inline,
	}
}

func getFieldFunc(ctx registry.CommandContext, fields []embed.EmbedField, index int, name string, value i18n.MessageId, inline bool, format ...interface{}) func() error {
	return func() error {
		fields[index] = newFieldFromTranslation(ctx, name, value, inline, format...)
		return nil
	}
}
