package metrics

import (
	"regexp"
	"strings"

	"github.com/chimii-ai/chimii/server/pkg/taskfailure"
)

const (
	labelSource         = "source"
	labelRuntimeMode    = "runtime_mode"
	labelProvider       = "provider"
	labelTerminalStatus = "terminal_status"
	labelFailureReason  = "failure_reason"
	labelTokenType      = "token_type"
	labelModel          = "model"
	labelModelAlias     = "model_alias"

	// PR3 labels (funnel / community / commercial).
	labelSignupSource = "signup_source"
	labelPlatform     = "platform"
	labelPath         = "path"
	labelCadence      = "cadence"
	labelTriggerKind  = "trigger_kind"
	labelReason       = "reason"
	labelRecoverable  = "recoverable"
	labelKind         = "kind"
	labelStatus       = "status"
	labelEventKind    = "event_kind"
	labelAction       = "action"
	labelResult       = "result"
	labelOp           = "op"
	labelGate         = "gate"
)

var businessMetricLabels = map[string][]string{
	"chimii_agent_task_enqueued_total":     {labelSource, labelRuntimeMode},
	"chimii_agent_task_dispatched_total":   {labelSource, labelRuntimeMode},
	"chimii_agent_task_started_total":      {labelSource, labelRuntimeMode, labelProvider},
	"chimii_agent_task_terminal_total":     {labelSource, labelRuntimeMode, labelTerminalStatus},
	"chimii_agent_task_failed_total":       {labelSource, labelRuntimeMode, labelFailureReason},
	"chimii_agent_task_queue_wait_seconds": {labelSource, labelRuntimeMode},
	"chimii_agent_task_run_seconds":        {labelSource, labelRuntimeMode, labelTerminalStatus},
	"chimii_agent_task_total_seconds":      {labelSource, labelRuntimeMode, labelTerminalStatus},
	"chimii_agent_task_in_progress":        {labelSource, labelRuntimeMode},
	"chimii_agent_task_iteration_count":    {labelSource, labelTerminalStatus},
	"chimii_llm_tokens_total":              {labelProvider, labelModel, labelTokenType, labelRuntimeMode, labelSource},
	"chimii_llm_cost_usd_total":            {labelProvider, labelModel, labelTokenType, labelRuntimeMode, labelSource},
	"chimii_llm_unpriced_tokens_total":     {labelProvider, labelModelAlias, labelTokenType},
	"chimii_llm_request_total":             {labelProvider, labelModel, labelRuntimeMode},
	"chimii_task_queued_expired_total":     {labelSource, labelRuntimeMode},
	"chimii_task_lease_expired_total":      {labelSource},

	// PR3 funnel / community / commercial.
	"chimii_signup_total":                             {labelSignupSource},
	"chimii_workspace_created_total":                  {labelSource},
	"chimii_team_invite_sent_total":                   {},
	"chimii_team_invite_accepted_total":               {},
	"chimii_onboarding_started_total":                 {labelPlatform},
	"chimii_onboarding_questionnaire_submitted_total": {},
	"chimii_onboarding_source_submitted_total":        {},
	"chimii_onboarding_completed_total":               {labelPath},
	"chimii_cloud_waitlist_joined_total":              {},
	"chimii_issue_created_total":                      {labelSource, labelPlatform},
	"chimii_chat_message_sent_total":                  {labelPlatform},
	"chimii_agent_created_total":                      {labelRuntimeMode, labelSource},
	"chimii_squad_created_total":                      {},
	"chimii_autopilot_created_total":                  {labelCadence},
	"chimii_issue_executed_total":                     {labelSource},
	"chimii_runtime_registered_total":                 {labelRuntimeMode, labelProvider},
	"chimii_runtime_ready_total":                      {labelRuntimeMode, labelProvider},
	"chimii_runtime_ready_seconds":                    {labelRuntimeMode, labelProvider},
	"chimii_runtime_failed_total":                     {labelRuntimeMode, labelProvider, labelFailureReason, labelRecoverable},
	"chimii_runtime_offline_total":                    {labelRuntimeMode, labelProvider},
	"chimii_daemon_ws_message_received_total":         {labelKind},
	"chimii_autopilot_run_started_total":              {labelCadence, labelTriggerKind},
	"chimii_autopilot_run_terminal_total":             {labelCadence, labelTriggerKind, labelTerminalStatus},
	"chimii_autopilot_run_skipped_total":              {labelCadence, labelReason},
	"chimii_webhook_delivery_total":                   {labelProvider, labelStatus},
	"chimii_webhook_rate_limited_total":               {labelGate},
	"chimii_github_event_received_total":              {labelEventKind, labelAction},
	"chimii_github_pr_review_total":                   {labelResult},
	"chimii_cloudruntime_request_total":               {labelOp, labelStatus},
	"chimii_cloudruntime_request_duration_seconds":    {labelOp},
	"chimii_feedback_submitted_total":                 {labelKind, labelPlatform},
	"chimii_contact_sales_submitted_total":            {labelSource},
	"chimii_chat_output_local_path_total":             {labelKind},
}

var forbiddenMetricLabels = map[string]struct{}{
	"workspace_id": {},
	"user_id":      {},
	"agent_id":     {},
	"task_id":      {},
	"issue_id":     {},
	"runtime_id":   {},
	"session_id":   {},
	"ip":           {},
}

var (
	knownSources = map[string]string{
		"issue":           "issue",
		"chat":            "chat",
		"autopilot":       "autopilot",
		"autopilot_issue": "autopilot_issue",
		"quick_create":    "quick_create",
		"manual":          "manual",
		"api":             "api",
		"other":           "other",
	}
	knownRuntimeModes = map[string]string{
		"local":   "local",
		"cloud":   "cloud",
		"unknown": "unknown",
	}
	knownRuntimeProviders = map[string]string{
		"antigravity":   "antigravity",
		"claude":        "claude",
		"codebuddy":     "codebuddy",
		"codex":         "codex",
		"copilot":       "copilot",
		"cursor":        "cursor",
		"gemini":        "gemini",
		"grok":          "grok",
		"hermes":        "hermes",
		"kiro":          "kiro",
		"kimi":          "kimi",
		"chimii_agent": "chimii_agent",
		"openclaw":      "openclaw",
		"opencode":      "opencode",
		"deveco":        "deveco",
		"pi":            "pi",
		"qwen":          "qwen",
		"other":         "other",
	}
	knownTerminalStatuses = map[string]string{
		"completed": "completed",
		"failed":    "failed",
		"cancelled": "cancelled",
		"blocked":   "blocked",
		"other":     "other",
	}
	knownTokenTypes = map[string]string{
		"input":       "input",
		"output":      "output",
		"cache_read":  "cache_read",
		"cache_write": "cache_write",
	}
	knownFailureReasons = map[string]string{}
	modelAliasUnsafeRe  = regexp.MustCompile(`[^a-z0-9._:/+-]+`)
)

func init() {
	for _, reason := range taskfailure.AllReasons() {
		knownFailureReasons[reason.String()] = reason.String()
	}
}

func validateBusinessMetricLabels() {
	for metric, labels := range businessMetricLabels {
		for _, label := range labels {
			if _, forbidden := forbiddenMetricLabels[label]; forbidden {
				panic("forbidden high-cardinality label " + label + " on " + metric)
			}
		}
	}
}

func metricLabels(metric string) []string {
	labels, ok := businessMetricLabels[metric]
	if !ok {
		panic("missing business metric label definition for " + metric)
	}
	return labels
}

func NormalizeTaskSource(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if normalized, ok := knownSources[value]; ok {
		return normalized
	}
	return "other"
}

func NormalizeRuntimeMode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if normalized, ok := knownRuntimeModes[value]; ok {
		return normalized
	}
	return "unknown"
}

func NormalizeRuntimeProvider(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if normalized, ok := knownRuntimeProviders[value]; ok {
		return normalized
	}
	return "other"
}

func NormalizeTerminalStatus(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if normalized, ok := knownTerminalStatuses[value]; ok {
		return normalized
	}
	return "other"
}

func NormalizeFailureReason(value string) string {
	value = strings.TrimSpace(value)
	if normalized, ok := knownFailureReasons[value]; ok {
		return normalized
	}
	return taskfailure.Classify(value).String()
}

func NormalizeTokenType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if normalized, ok := knownTokenTypes[value]; ok {
		return normalized
	}
	return "input"
}

func NormalizeModelAlias(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	value = modelAliasUnsafeRe.ReplaceAllString(value, "_")
	if len(value) > 128 {
		return value[:128]
	}
	return value
}
