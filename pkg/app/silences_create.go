package app

import (
	"fmt"
	"main/pkg/constants"
	"main/pkg/types"
	"main/pkg/types/render"
	"main/pkg/utils"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (a *App) HandleGrafanaNewSilence(c tele.Context) error {
	a.Logger.Info().
		Str("sender", c.Sender().Username).
		Str("text", c.Text()).
		Msg("Got new silence query")

	silenceInfo, err := utils.ParseSilenceFromCommand(c.Text(), c.Sender().FirstName)
	if err != "" {
		return c.Reply(fmt.Sprintf("Error parsing silence option: %s\n", err))
	}

	return a.HandleNewSilence(c, a.Grafana, constants.GrafanaUnsilencePrefix, silenceInfo)
}

func (a *App) HandleAlertmanagerNewSilence(c tele.Context) error {
	a.Logger.Info().
		Str("sender", c.Sender().Username).
		Str("text", c.Text()).
		Msg("Got new Alertmanager silence query")

	if !a.Alertmanager.Enabled() {
		return c.Reply("Alertmanager is disabled.")
	}

	silenceInfo, err := utils.ParseSilenceFromCommand(c.Text(), c.Sender().FirstName)
	if err != "" {
		return c.Reply(fmt.Sprintf("Error parsing silence option: %s\n", err))
	}

	return a.HandleNewSilence(c, a.Alertmanager, constants.AlertmanagerUnsilencePrefix, silenceInfo)
}

// func (a *App) HandleAlertmanagerPrepareNewSilenceFromCallback(c tele.Context) error {
//	a.Logger.Info().
//		Str("sender", c.Sender().Username).
//		Msg("Got new prepare Alertmanager silence callback via button")
//
//	callback := c.Callback()
//	a.RemoveKeyboardItemByCallback(c, callback)
//
//	groups, err := a.Grafana.GetGrafanaAlertingRules()
//	if err != nil {
//		return c.Reply(fmt.Sprintf("Error querying alerts: %s", err))
//	}
//
//	groups = groups.FilterFiringOrPendingAlertGroups()
//
//	silenceInfo, err := a.GenerateSilenceForAlert(c, groups, alertHashToMute, dataSplit[0])
//	if err != nil {
//		return c.Reply(err.Error())
//	}
//
//	return c.Reply("Alert was not found!")
//}

func (a *App) HandleGrafanaCallbackNewSilence(c tele.Context) error {
	a.Logger.Info().
		Str("sender", c.Sender().Username).
		Msg("Got new create Grafana silence callback via button")

	callback := c.Callback()
	a.RemoveKeyboardItemByCallback(c, callback)

	dataSplit := strings.SplitN(callback.Data, " ", 2)
	if len(dataSplit) != 2 {
		return c.Reply("Invalid callback provided!")
	}

	alertHashToMute := dataSplit[1]

	groups, err := a.Grafana.GetGrafanaAlertingRules()
	if err != nil {
		return c.Reply(fmt.Sprintf("Error querying alerts: %s", err))
	}

	silenceInfo, err := a.GenerateSilenceForAlert(c, groups, alertHashToMute, dataSplit[0])
	if err != nil {
		return c.Reply(err.Error())
	}

	return a.HandleNewSilence(c, a.Grafana, constants.GrafanaUnsilencePrefix, silenceInfo)
}

func (a *App) HandleAlertmanagerCallbackNewSilence(c tele.Context) error {
	a.Logger.Info().
		Str("sender", c.Sender().Username).
		Msg("Got new create Alertmanager silence callback via button")

	callback := c.Callback()
	a.RemoveKeyboardItemByCallback(c, callback)

	dataSplit := strings.SplitN(callback.Data, " ", 2)
	if len(dataSplit) != 2 {
		return c.Reply("Invalid callback provided!")
	}

	alertHashToMute := dataSplit[1]

	groups, err := a.Grafana.GetPrometheusAlertingRules()
	if err != nil {
		return c.Reply(fmt.Sprintf("Error querying alerts: %s", err))
	}

	silenceInfo, err := a.GenerateSilenceForAlert(c, groups, alertHashToMute, dataSplit[0])
	if err != nil {
		return c.Reply(err.Error())
	}

	return a.HandleNewSilence(c, a.Alertmanager, constants.AlertmanagerUnsilencePrefix, silenceInfo)
}

func (a *App) HandleNewSilence(
	c tele.Context,
	silenceManager types.SilenceManager,
	unsilencePrefix string,
	silenceInfo *types.Silence,
) error {
	silenceResponse, silenceErr := silenceManager.CreateSilence(*silenceInfo)
	if silenceErr != nil {
		return c.Reply(fmt.Sprintf("Error creating silence: %s", silenceErr))
	}

	silence, silenceErr := silenceManager.GetSilence(silenceResponse.SilenceID)
	if silenceErr != nil {
		return c.Reply(fmt.Sprintf("Error getting created silence: %s", silenceErr))
	}

	alerts, alertsErr := silenceManager.GetSilenceMatchingAlerts(silence)
	if alertsErr != nil {
		return c.Reply(fmt.Sprintf("Error getting alerts for silence: %s", alertsErr))
	}

	template, renderErr := a.TemplateManager.Render("silences_create", render.RenderStruct{
		Grafana:      a.Grafana,
		Alertmanager: a.Alertmanager,
		Data: types.SilenceWithAlerts{
			Silence:       silence,
			AlertsPresent: alerts != nil,
			Alerts:        alerts,
		},
	})
	if renderErr != nil {
		a.Logger.Error().Err(renderErr).Msg("Error rendering silences_create template")
		return c.Reply(fmt.Sprintf("Error rendering template: %s", renderErr))
	}

	menu := &tele.ReplyMarkup{ResizeKeyboard: true}
	menu.Inline(menu.Row(menu.Data(
		"❌Unsilence",
		unsilencePrefix,
		silence.ID,
	)))

	return a.BotReply(c, template, menu)
}
