package handler

import (
	"encoding/json"
	"errors"
	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"strings"
	"time"
)

// Question IDs carry their revision. Legacy clients can submit displayed labels;
// new clients submit stable choice IDs. Both are normalized before persistence.
func validateBuildAnswer(session db.BuildSession, req submitBuildAnswersRequest) (map[string]string, bool, error) {
	bad := errors.New("stale or invalid answer")
	if len(req.Answers) != 1 || (session.ExpiresAt.Valid && !session.ExpiresAt.Time.After(time.Now())) {
		return nil, false, bad
	}
	var history map[string]string
	if err := json.Unmarshal(session.Answers, &history); err != nil {
		return nil, false, err
	}
	for key, value := range req.Answers {
		value = strings.TrimSpace(value)

		if old, ok := history[key]; ok {
			if old == value || history[key+":choice"] == value {
				return nil, true, nil
			}
			return nil, false, bad
		}
		if session.Status != "clarifying" || (req.Revision > 0 && req.Revision != session.Revision) {
			return nil, false, bad
		}
		var question buildstudio.ClarifyingQuestion
		if err := json.Unmarshal(session.Question, &question); err != nil {
			return nil, false, err
		}
		if key != question.ID || value == "" || len([]rune(value)) > 120 {
			return nil, false, bad
		}
		chosen := ""
		for _, c := range question.Choices {
			if value == c.ID || value == c.Label {
				chosen = c.ID
				value = c.Label
				break
			}
		}
		if chosen == "" && !question.AllowFreeText && !buildContainsString(question.Options, value) {
			return nil, false, bad
		}
		out := map[string]string{key: value, key + ":question": question.Prompt}
		if chosen != "" {
			out[key+":choice"] = chosen
		}
		return out, false, nil
	}
	return nil, false, bad
}
