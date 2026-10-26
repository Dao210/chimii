package llm

import (
	"context"
	"encoding/base64"
	"testing"
)

func TestTaskReasoningAndImagesReachProvider(t *testing.T) {
	var captured capturedRequest
	c := New(Config{BaseURL: "https://fixture.invalid", HTTPClient: stubHTTPClient(t, func(got capturedRequest) string { captured = got; return anthropicMessage("fixture", "reviewed") })})
	image := []byte("fixture PNG bytes")
	if _, err := c.GenerateContent(context.Background(), "", "review", "model", GenerationOptions{Effort: "high", Purpose: "build-review", Images: []Image{{MediaType: "image/png", Data: image}}}); err != nil {
		t.Fatal(err)
	}
	if captured.Body["output_config"].(map[string]any)["effort"] != "high" {
		t.Fatal("task reasoning discarded")
	}
	blocks := captured.Body["messages"].([]any)[0].(map[string]any)["content"].([]any)
	if len(blocks) != 2 || blocks[1].(map[string]any)["type"] != "image" {
		t.Fatal("image was replaced by text")
	}
	data := blocks[1].(map[string]any)["source"].(map[string]any)["data"].(string)
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil || string(decoded) != string(image) {
		t.Fatal("image bytes changed")
	}
	if _, err = c.GenerateContent(context.Background(), "", "", "", GenerationOptions{Effort: "unbounded"}); err == nil {
		t.Fatal("invalid reasoning policy accepted")
	}
	if _, err = c.GenerateContent(context.Background(), "", "", "", GenerationOptions{Images: []Image{{MediaType: "image/png"}}}); err == nil {
		t.Fatal("empty image accepted")
	}
}
