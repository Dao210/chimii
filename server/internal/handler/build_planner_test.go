package handler

import "testing"

func TestParseBuildIntentAcceptsConstrainedJSON(t *testing.T) {
	intent, err := parseBuildIntent(`{"archetype":"flyer","title":"云朵小飞龙","features":["wide-wings","wide-wings","not-real","balanced-tail"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if intent.Archetype != "flyer" || intent.Title != "云朵小飞龙" {
		t.Fatalf("unexpected intent: %#v", intent)
	}
	if len(intent.Features) != 2 {
		t.Fatalf("features not filtered: %#v", intent.Features)
	}
}

func TestParseBuildIntentRejectsGeometryAndUnknownFields(t *testing.T) {
	for _, raw := range []string{
		`{"archetype":"spaceship","title":"X","features":["wide-wings"]}`,
		`{"archetype":"flyer","title":"X","features":["wide-wings"],"coordinates":[1,2,3]}`,
		`{"archetype":"flyer","title":"X","features":["invented-feature"]}`,
		"```json\n{\"archetype\":\"flyer\",\"title\":\"X\",\"features\":[\"wide-wings\"]}\n```",
		`{"archetype":"flyer","title":"X","features":["wide-wings"]} trailing prose`,
		`{"archetype":"flyer","title":"X","features":["wide-wings"]}{}`,
	} {
		if _, err := parseBuildIntent(raw); err == nil {
			t.Fatalf("should reject %s", raw)
		}
	}
}
