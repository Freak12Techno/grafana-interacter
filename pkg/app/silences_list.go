package app

import (
	"fmt"
	"main/pkg/constants"
	"main/pkg/types"
	"main/pkg/types/render"
	"main/pkg/utils/generic"

	tele "gopkg.in/telebot.v3"
)

func (a *App) HandleGrafanaListSilences(c tele.Context) error {
	a.Logger.Info().
		Str("sender", c.Sender().Username).
		Str("text", c.Text()).
		Msg("Got Grafana list silence query")

	return a.HandleListSilences(c, a.Grafana, constants.GrafanaUnsilencePrefix)
}

func (a *App) HandleAlertmanagerListSilences(c tele.Context) error {
	a.Logger.Info().
		Str("sender", c.Sender().Username).
		Str("text", c.Text()).
		Msg("Got Alertmanager list silence query")

	if !a.Alertmanager.Enabled() {
		return c.Reply("Alertmanager is disabled.")
	}

	return a.HandleListSilences(c, a.Alertmanager, constants.AlertmanagerUnsilencePrefix)
}

func (a *App) HandleListSilences(
	c tele.Context,
	silenceManager types.SilenceManager,
	unsilencePrefix string,
) error {
	silencesWithAlerts, err := types.GetSilencesWithAlerts(silenceManager)
	if err != nil {
		return c.Reply(fmt.Sprintf("Error fetching silences: %s\n", err))
	}

	silencesGrouped := generic.SplitArrayIntoChunks(silencesWithAlerts, constants.SilencesInOneMessage)
	if len(silencesGrouped) == 0 {
		silencesGrouped = [][]types.SilenceWithAlerts{{}}
	}

	for index, chunk := range silencesGrouped {
		template, renderErr := a.TemplateManager.Render("silences_list", render.RenderStruct{
			Grafana:      a.Grafana,
			Alertmanager: a.Alertmanager,
			Data: types.SilencesListStruct{
				Silences:      chunk,
				ShowHeader:    index == 0,
				SilencesCount: len(silencesWithAlerts),
			},
		})

		if renderErr != nil {
			a.Logger.Error().Err(renderErr).Msg("Error rendering silences_list template")
			return c.Reply(fmt.Sprintf("Error rendering template: %s", renderErr))
		}

		menu := &tele.ReplyMarkup{ResizeKeyboard: true}

		rows := make([]tele.Row, constants.SilencesInOneMessage)

		for silenceIndex, silence := range chunk {
			button := menu.Data(
				fmt.Sprintf("❌Unsilence %s", silence.Silence.ID),
				unsilencePrefix,
				silence.Silence.ID,
			)

			rows[silenceIndex] = menu.Row(button)
		}

		menu.Inline(rows...)

		if sendErr := a.BotReply(c, template, menu); sendErr != nil {
			a.Logger.Error().Err(sendErr).Msg("Error sending message")
		}
	}

	return nil
}
