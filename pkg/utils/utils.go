package utils

import (
	"fmt"
	"main/pkg/logger"
	"main/pkg/types"
	"regexp"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func NormalizeString(input string) string {
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	return strings.ToLower(reg.ReplaceAllString(input, ""))
}

func Filter[T any](slice []T, f func(T) bool) []T {
	var n []T
	for _, e := range slice {
		if f(e) {
			n = append(n, e)
		}
	}
	return n
}

func Map[T, V any](slice []T, f func(T) V) []V {
	n := make([]V, len(slice))
	for index, e := range slice {
		n[index] = f(e)
	}
	return n
}

func FindDashboardByName(dashboards []types.GrafanaDashboardInfo, name string) (*types.GrafanaDashboardInfo, bool) {
	normalizedName := NormalizeString(name)

	for _, dashboard := range dashboards {
		if strings.Contains(NormalizeString(dashboard.Title), normalizedName) {
			return &dashboard, true
		}
	}

	return nil, false
}

func FindPanelByName(panels []types.PanelStruct, name string) (*types.PanelStruct, bool) {
	normalizedName := NormalizeString(name)

	for _, panel := range panels {
		panelNameWithDashboardName := NormalizeString(panel.DashboardName + panel.Name)

		if strings.Contains(panelNameWithDashboardName, normalizedName) {
			return &panel, true
		}
	}

	return nil, false
}

func FindAlertRuleByName(groups []types.GrafanaAlertGroup, name string) (*types.GrafanaAlertRule, bool) {
	normalizedName := NormalizeString(name)

	for _, group := range groups {
		for _, rule := range group.Rules {
			ruleName := NormalizeString(group.Name + rule.Name)
			if strings.Contains(ruleName, normalizedName) {
				return &rule, true
			}
		}
	}

	return nil, false
}

func ParseRenderOptions(query string) (types.RenderOptions, bool) {
	args := strings.Split(query, " ")
	if len(args) <= 1 {
		return types.RenderOptions{}, false // should have at least 1 argument
	}

	params := map[string]string{}

	_, args = args[0], args[1:] // removing first argument as it's always /render
	for len(args) > 0 {
		if !strings.Contains(args[0], "=") {
			break
		}

		paramSplit := strings.SplitN(args[0], "=", 2)
		params[paramSplit[0]] = paramSplit[1]

		_, args = args[0], args[1:]
	}

	return types.RenderOptions{
		Query:  strings.Join(args, " "),
		Params: params,
	}, len(args) > 0
}

func SerializeQueryString(qs map[string]string) string {
	tmp := make([]string, len(qs))
	counter := 0

	for key, value := range qs {
		tmp[counter] = key + "=" + value
		counter++
	}

	return strings.Join(tmp, "&")
}

func MergeMaps(first, second map[string]string) map[string]string {
	for key, value := range second {
		first[key] = value
	}

	return first
}

func GetEmojiByStatus(state string) string {
	switch strings.ToLower(state) {
	case "inactive", "ok", "normal":
		return "🟢"
	case "pending":
		return "🟡"
	case "firing", "alerting":
		return "🔴"
	default:
		return "[" + state + "]"
	}
}

func GetEmojiBySilenceStatus(state string) string {
	switch strings.ToLower(state) {
	case "active":
		return "🟢"
	case "expired":
		return "⚪"
	default:
		return "[" + state + "]"
	}
}

func ParseSilenceOptions(query string, c tele.Context) (*types.Silence, string) {
	args := strings.Split(query, " ")
	if len(args) <= 2 {
		return nil, fmt.Sprintf("Usage: %s <duration> <params>", args[0])
	}

	_, args = args[0], args[1:] // removing first argument as it's always /silence
	durationString, args := args[0], args[1:]

	duration, err := time.ParseDuration(durationString)
	if err != nil {
		return nil, "Invalid duration provided"
	}

	silence := types.Silence{
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(duration),
		Matchers:  []types.SilenceMatcher{},
		CreatedBy: c.Sender().FirstName,
		Comment: fmt.Sprintf(
			"Muted using grafana-interacter for %s by %s",
			duration,
			c.Sender().FirstName,
		),
	}

	for len(args) > 0 {
		if strings.Contains(args[0], "!=") {
			// not equals
			argsSplit := strings.SplitN(args[0], "!=", 2)
			silence.Matchers = append(silence.Matchers, types.SilenceMatcher{
				IsEqual: false,
				IsRegex: false,
				Name:    argsSplit[0],
				Value:   argsSplit[1],
			})
		} else if strings.Contains(args[0], "!~") {
			// not matches regexp
			argsSplit := strings.SplitN(args[0], "!~", 2)
			silence.Matchers = append(silence.Matchers, types.SilenceMatcher{
				IsEqual: false,
				IsRegex: true,
				Name:    argsSplit[0],
				Value:   argsSplit[1],
			})
		} else if strings.Contains(args[0], "=~") {
			// matches regexp
			argsSplit := strings.SplitN(args[0], "=~", 2)
			silence.Matchers = append(silence.Matchers, types.SilenceMatcher{
				IsEqual: true,
				IsRegex: true,
				Name:    argsSplit[0],
				Value:   argsSplit[1],
			})
		} else if strings.Contains(args[0], "=") {
			// equals
			argsSplit := strings.SplitN(args[0], "=", 2)
			silence.Matchers = append(silence.Matchers, types.SilenceMatcher{
				IsEqual: true,
				IsRegex: false,
				Name:    argsSplit[0],
				Value:   argsSplit[1],
			})
		} else {
			break
		}

		_, args = args[0], args[1:]
	}

	if len(args) > 0 {
		// plain string, silencing by alertname
		silence.Matchers = append(silence.Matchers, types.SilenceMatcher{
			IsEqual: true,
			IsRegex: false,
			Name:    "alertname",
			Value:   strings.Join(args, " "),
		})
	}

	if len(silence.Matchers) == 0 {
		return nil, "Usage: /silence <duration> <params>"
	}

	return &silence, ""
}

func FilterFiringOrPendingAlertGroups(groups []types.GrafanaAlertGroup) []types.GrafanaAlertGroup {
	var returnGroups []types.GrafanaAlertGroup

	for _, group := range groups {
		rules := []types.GrafanaAlertRule{}
		hasAnyRules := false

		for _, rule := range group.Rules {
			if rule.State == "firing" || rule.State == "alerting" || rule.State == "pending" {
				rules = append(rules, rule)
				hasAnyRules = true
			}
		}

		if hasAnyRules {
			returnGroups = append(returnGroups, types.GrafanaAlertGroup{
				Name:  group.Name,
				File:  group.File,
				Rules: rules,
			})
		}
	}

	return returnGroups
}

func StrToFloat64(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		logger.GetDefaultLogger().Fatal().Err(err).Str("value", s).Msg("Could not parse float")
	}

	return f
}
