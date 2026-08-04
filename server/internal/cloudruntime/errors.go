package cloudruntime

import (
	"context"
	"errors"
	"strings"

	"github.com/chimii-ai/chimii/server/pkg/taskfailure"
)

const (
	FailureUnavailable = string(taskfailure.ReasonCloudRuntimeUnavailable)
	FailureAuth        = string(taskfailure.ReasonCloudProviderAuth)
	FailureRateLimited = string(taskfailure.ReasonCloudProviderRateLimited)
	FailureProvider    = string(taskfailure.ReasonCloudProviderError)
	FailureSandbox     = string(taskfailure.ReasonCloudSandboxViolation)
	FailureSession     = string(taskfailure.ReasonCloudSessionCorrupt)
	FailureWorkDir     = string(taskfailure.ReasonCloudWorkDirPrepare)
	FailureTimeout     = string(taskfailure.ReasonCloudRuntimeTimeout)
)

type ClassifiedError struct {
	Reason    string
	Retryable bool
}

func ClassifyError(err error) ClassifiedError {
	if err == nil {
		return ClassifiedError{}
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return ClassifiedError{Reason: FailureTimeout, Retryable: true}
	case errors.Is(err, context.Canceled):
		return ClassifiedError{Reason: FailureUnavailable, Retryable: true}
	case errors.Is(err, ErrUnsafeWorkDir):
		return ClassifiedError{Reason: FailureSandbox}
	case errors.Is(err, ErrWorkDirPrepare):
		return ClassifiedError{Reason: FailureWorkDir, Retryable: true}
	case errors.Is(err, ErrSessionCorrupt), errors.Is(err, ErrSessionScope), errors.Is(err, ErrInvalidSessionID), errors.Is(err, ErrSessionNotFound):
		return ClassifiedError{Reason: FailureSession}
	case errors.Is(err, ErrProviderDisabled), errors.Is(err, ErrModelNotConfigured):
		return ClassifiedError{Reason: FailureUnavailable}
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "status 401"), strings.Contains(message, "status 403"), strings.Contains(message, "api key"), strings.Contains(message, "authentication"):
		return ClassifiedError{Reason: FailureAuth}
	case strings.Contains(message, "status 429"), strings.Contains(message, "rate limit"):
		return ClassifiedError{Reason: FailureRateLimited, Retryable: true}
	case strings.Contains(message, "status 5"), strings.Contains(message, "connection reset"), strings.Contains(message, "unexpected eof"):
		return ClassifiedError{Reason: FailureProvider, Retryable: true}
	default:
		return ClassifiedError{Reason: FailureProvider, Retryable: true}
	}
}
