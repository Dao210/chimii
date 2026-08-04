package cloudruntime

import (
	"context"
	"errors"
	"testing"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		err       error
		reason    string
		retryable bool
	}{
		{context.DeadlineExceeded, FailureTimeout, true},
		{ErrUnsafeWorkDir, FailureSandbox, false},
		{ErrSessionCorrupt, FailureSession, false},
		{ErrSessionNotFound, FailureSession, false},
		{errors.New("provider returned status 401"), FailureAuth, false},
		{errors.New("provider returned status 429"), FailureRateLimited, true},
		{errors.New("provider returned status 503"), FailureProvider, true},
		{errors.New("provider stream failed"), FailureProvider, true},
	}
	for _, test := range tests {
		got := ClassifyError(test.err)
		if got.Reason != test.reason || got.Retryable != test.retryable {
			t.Errorf("ClassifyError(%v) = %+v", test.err, got)
		}
	}
}
