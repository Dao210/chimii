package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/chimii-ai/chimii/server/internal/auth"
	cloudruntime "github.com/chimii-ai/chimii/server/internal/cloudruntime"
	"github.com/chimii-ai/chimii/server/pkg/agent"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/chimii-ai/chimii/server/pkg/protocol"
	"github.com/chimii-ai/chimii/server/pkg/redact"
	"github.com/jackc/pgx/v5/pgtype"
)

func (h *Handler) ClaimCloudTask(ctx context.Context, runtime db.AgentRuntime) (*cloudruntime.ClaimedRun, error) {
	if runtime.ExecutionType != "cloud" || runtime.RuntimeMode != "cloud" {
		return nil, fmt.Errorf("runtime is not a server-side cloud runtime")
	}
	if !h.runtimeExecutionAvailable(runtime) {
		return nil, cloudruntime.ErrProviderDisabled
	}
	task, err := h.TaskService.ClaimTaskForRuntime(ctx, runtime.ID)
	if err != nil || task == nil {
		return nil, err
	}
	workspaceID := uuidToString(runtime.WorkspaceID)
	if handled, repairErr := h.repairStaleCommentPlanIfNeeded(ctx, task, workspaceID); handled {
		if repairErr != nil {
			return nil, fmt.Errorf("repair stale cloud task plan: %s", repairErr.message)
		}
		return nil, nil
	}
	capabilities := protocol.DaemonCapabilityCoalescedCommentsV1
	resp, deliveredCommentIDs, _, _, failure := h.buildClaimedTaskResponse(ctx, capabilities, task, runtime, uuidToString(runtime.ID), workspaceID)
	if failure != nil {
		return nil, fmt.Errorf("build cloud task input: %s", failure.message)
	}
	if !runtime.OwnerID.Valid {
		_, _ = h.TaskService.CancelTask(ctx, task.ID)
		return nil, fmt.Errorf("runtime owner required to mint task token")
	}
	token, err := auth.GenerateAgentTaskToken()
	if err != nil {
		h.requeueCloudClaim(ctx, *task, "token_generation")
		return nil, fmt.Errorf("generate cloud task token: %w", err)
	}
	commentBacked := task.TriggerCommentID.Valid || len(task.CoalescedCommentIds) > 0
	receipt, err := h.TaskService.FinalizeTaskClaim(ctx, *task, db.CreateTaskTokenParams{
		TokenHash:   auth.HashToken(token),
		TaskID:      task.ID,
		AgentID:     task.AgentID,
		WorkspaceID: runtime.WorkspaceID,
		UserID:      runtime.OwnerID,
		ExpiresAt:   pgtype.Timestamptz{Time: time.Now().Add(24 * time.Hour), Valid: true},
	}, deliveredCommentIDs, commentBacked)
	if err != nil {
		h.requeueCloudClaim(ctx, *task, "claim_finalization")
		return nil, fmt.Errorf("finalize cloud task claim: %w", err)
	}
	resp.AuthToken = token
	resp.DeliveredCommentIDs = uuidStringsOrEmpty(receipt)

	contextType, contextID := cloudTaskContext(resp)
	systemPrompt := cloudSystemPrompt(resp)
	return &cloudruntime.ClaimedRun{
		TaskID:         resp.ID,
		WorkspaceID:    resp.WorkspaceID,
		RuntimeID:      resp.RuntimeID,
		AgentID:        resp.AgentID,
		IssueID:        resp.IssueID,
		Provider:       runtime.Provider,
		Model:          cloudTaskModel(resp),
		Prompt:         cloudTaskPrompt(resp),
		SystemPrompt:   systemPrompt,
		PriorSessionID: resp.PriorSessionID,
		ContextType:    contextType,
		ContextID:      contextID,
	}, nil
}

func (h *Handler) requeueCloudClaim(ctx context.Context, task db.AgentTaskQueue, reason string) {
	if _, err := h.TaskService.RequeueTaskAfterClaimFailure(ctx, task); err != nil {
		slog.Error("cloud runtime claim requeue failed", "task_id", uuidToString(task.ID), "reason", reason, "error", err)
	}
}

func cloudTaskModel(task AgentTaskResponse) string {
	if task.Agent == nil {
		return ""
	}
	return task.Agent.Model
}

func cloudTaskContext(task AgentTaskResponse) (string, string) {
	switch {
	case task.ChatSessionID != "":
		return "chat", task.ChatSessionID
	case task.AutopilotRunID != "":
		return "autopilot", task.AutopilotRunID
	case task.QuickCreatePrompt != "":
		return "quick_create", task.ID
	case task.IssueID != "":
		return "issue", task.IssueID
	default:
		return "direct", task.ID
	}
}

func cloudSystemPrompt(task AgentTaskResponse) string {
	var prompt strings.Builder
	prompt.WriteString("You are executing inside Chimii's server-side Cloud runtime. This execution type is explicit and must never fall back to a local CLI runtime. Host shell and filesystem tools are disabled; do not claim to have run commands or changed files. Use only the task context supplied in this conversation and give a direct, truthful result.\n")
	if task.Agent != nil {
		if task.Agent.Name != "" {
			fmt.Fprintf(&prompt, "\nAgent name: %s\n", task.Agent.Name)
		}
		if strings.TrimSpace(task.Agent.Instructions) != "" {
			prompt.WriteString("\nAgent instructions:\n")
			prompt.WriteString(task.Agent.Instructions)
			prompt.WriteByte('\n')
		}
	}
	if strings.TrimSpace(task.WorkspaceContext) != "" {
		prompt.WriteString("\nWorkspace context:\n")
		prompt.WriteString(task.WorkspaceContext)
		prompt.WriteByte('\n')
	}
	if task.Agent != nil {
		const maxSkillPromptBytes = 64 * 1024
		written := 0
		for _, skill := range task.Agent.Skills {
			content := strings.TrimSpace(skill.Content)
			if content == "" || written >= maxSkillPromptBytes {
				continue
			}
			remaining := maxSkillPromptBytes - written
			if len(content) > remaining {
				content = content[:remaining]
			}
			fmt.Fprintf(&prompt, "\nSkill: %s\n%s\n", skill.Name, content)
			written += len(content)
		}
	}
	return prompt.String()
}

func cloudTaskPrompt(task AgentTaskResponse) string {
	var prompt strings.Builder
	switch {
	case task.ChatSessionID != "":
		if task.ChatIntro {
			prompt.WriteString("Introduce yourself briefly using your agent instructions.")
		} else {
			prompt.WriteString("Reply to the following user message using only the supplied context:\n\n")
			prompt.WriteString(task.ChatMessage)
		}
	case task.AutopilotRunID != "":
		fmt.Fprintf(&prompt, "Autopilot: %s\n\n%s", task.AutopilotTitle, task.AutopilotDescription)
		if len(task.AutopilotTriggerPayload) > 0 {
			fmt.Fprintf(&prompt, "\n\nTrigger payload:\n%s", string(task.AutopilotTriggerPayload))
		}
	case task.QuickCreatePrompt != "":
		prompt.WriteString("Prepare the requested issue content from this input. You cannot create the issue directly in text-only Cloud mode, so return a precise proposed title and description without claiming it was created:\n\n")
		prompt.WriteString(task.QuickCreatePrompt)
	case task.TriggerCommentID != nil:
		fmt.Fprintf(&prompt, "Issue %s: %s\n\nRespond to this comment", task.IssueID, task.ThreadName)
		if task.TriggerAuthorName != "" {
			fmt.Fprintf(&prompt, " from %s", task.TriggerAuthorName)
		}
		prompt.WriteString(":\n")
		prompt.WriteString(task.TriggerCommentContent)
		for _, comment := range task.CoalescedComments {
			fmt.Fprintf(&prompt, "\n\nEarlier comment from %s:\n%s", comment.AuthorName, comment.Content)
		}
	default:
		fmt.Fprintf(&prompt, "Issue %s: %s", task.IssueID, task.ThreadName)
		if task.HandoffNote != "" {
			fmt.Fprintf(&prompt, "\n\nHandoff note:\n%s", task.HandoffNote)
		}
		prompt.WriteString("\n\nAnalyze the supplied issue context and return a concrete result. Host tools are disabled, so do not claim to have inspected a repository, run commands, or changed files.")
	}
	return prompt.String()
}

func (h *Handler) StartCloudTask(ctx context.Context, run cloudruntime.ClaimedRun) error {
	_, err := h.TaskService.StartTask(ctx, parseUUID(run.TaskID))
	return err
}

func (h *Handler) PinCloudTaskSession(ctx context.Context, run cloudruntime.ClaimedRun, sessionID, workDir string) error {
	return h.Queries.UpdateAgentTaskSession(ctx, db.UpdateAgentTaskSessionParams{
		ID:        parseUUID(run.TaskID),
		SessionID: pgtype.Text{String: sessionID, Valid: sessionID != ""},
		WorkDir:   pgtype.Text{String: workDir, Valid: workDir != ""},
	})
}

func (h *Handler) AppendCloudTaskMessages(ctx context.Context, run cloudruntime.ClaimedRun, messages []cloudruntime.TaskMessage) error {
	for _, message := range messages {
		message.Content = redact.Text(message.Content)
		message.Output = redact.Text(message.Output)
		message.Input = redact.InputMap(message.Input)
		var inputJSON []byte
		if message.Input != nil {
			inputJSON, _ = json.Marshal(message.Input)
		}
		created, err := h.Queries.CreateTaskMessage(ctx, db.CreateTaskMessageParams{
			TaskID:  parseUUID(run.TaskID),
			Seq:     int32(message.Seq),
			Type:    message.Type,
			Tool:    pgtype.Text{String: message.Tool, Valid: message.Tool != ""},
			Content: pgtype.Text{String: message.Content, Valid: message.Content != ""},
			Input:   inputJSON,
			Output:  pgtype.Text{String: message.Output, Valid: message.Output != ""},
		})
		if err != nil {
			return fmt.Errorf("persist cloud task message: %w", err)
		}
		h.publishTask(protocol.EventTaskMessage, run.WorkspaceID, "system", "", run.TaskID,
			taskMessageToPayload(created, run.TaskID, run.IssueID))
	}
	return nil
}

func (h *Handler) CompleteCloudTask(ctx context.Context, run cloudruntime.ClaimedRun, result *cloudruntime.RunResult, workDir string) error {
	if result == nil {
		return errors.New("cloud runtime returned no result")
	}
	taskID := parseUUID(run.TaskID)
	costTicks := cloudAuthoritativeCostTicks(result.CostUSD)
	if err := h.Queries.UpsertTaskUsage(ctx, db.UpsertTaskUsageParams{
		TaskID:           taskID,
		Provider:         run.Provider,
		Model:            result.Model,
		InputTokens:      int64(result.Usage.InputTokens),
		OutputTokens:     int64(result.Usage.OutputTokens),
		CacheReadTokens:  int64(result.Usage.CacheReadInputTokens),
		CacheWriteTokens: int64(result.Usage.CacheCreationInputTokens),
		CostUsdTicks:     costTicks,
	}); err != nil {
		return fmt.Errorf("persist cloud task usage: %w", err)
	}
	payload, err := json.Marshal(TaskCompleteRequest{
		Output:    redact.Text(result.Output),
		SessionID: result.SessionID,
		WorkDir:   workDir,
	})
	if err != nil {
		return err
	}
	task, err := h.TaskService.CompleteTask(ctx, taskID, payload, result.SessionID, workDir, false)
	if err != nil {
		return err
	}
	h.emitIssueExecutedOnFirstCompletion(ctx, task)
	h.reconcileCommentsOnCompletion(ctx, task)
	h.TaskService.CaptureTaskUsage(ctx, *task, run.Provider, result.Model,
		int64(result.Usage.InputTokens), int64(result.Usage.OutputTokens),
		int64(result.Usage.CacheReadInputTokens), int64(result.Usage.CacheCreationInputTokens), costTicks.Int64)
	h.TaskService.NotifyTaskFinished(*task)
	if err := h.Queries.DeleteTaskTokensByTask(ctx, task.ID); err != nil {
		slog.Warn("cloud runtime completion token cleanup failed", "task_id", run.TaskID, "error", err)
	}
	return nil
}

func cloudAuthoritativeCostTicks(costUSD float64) pgtype.Int8 {
	if costUSD <= 0 || math.IsNaN(costUSD) || math.IsInf(costUSD, 0) {
		return pgtype.Int8{}
	}
	ticks := costUSD * float64(agent.CostUSDTicksPerUSD)
	if ticks > math.MaxInt64 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: int64(math.Round(ticks)), Valid: true}
}

func (h *Handler) FailCloudTask(ctx context.Context, run cloudruntime.ClaimedRun, runErr error, classified cloudruntime.ClassifiedError, sessionID, workDir string) error {
	errorText := "cloud runtime task failed"
	if runErr != nil {
		errorText = redact.Text(runErr.Error())
	}
	task, err := h.TaskService.FailTask(ctx, parseUUID(run.TaskID), errorText, sessionID, workDir, classified.Reason, false)
	if err != nil {
		return err
	}
	h.TaskService.NotifyTaskFinished(*task)
	if err := h.Queries.DeleteTaskTokensByTask(ctx, task.ID); err != nil {
		slog.Warn("cloud runtime failure token cleanup failed", "task_id", run.TaskID, "error", err)
	}
	return nil
}

func (h *Handler) CloudTaskStatus(ctx context.Context, run cloudruntime.ClaimedRun) (string, error) {
	task, err := h.Queries.GetAgentTask(ctx, parseUUID(run.TaskID))
	if err != nil {
		return "", err
	}
	return task.Status, nil
}

func (h *Handler) AcknowledgeCloudTaskCancelled(ctx context.Context, run cloudruntime.ClaimedRun) error {
	h.TaskService.FinalizeDeferredCancelledChat(ctx, parseUUID(run.TaskID))
	return nil
}
