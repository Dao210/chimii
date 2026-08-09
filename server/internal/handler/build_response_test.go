package handler

import (
	"testing"

	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
)

func TestToBuildCreationResponseCanonicalizesHistoricalNullCollections(t *testing.T) {
	row := db.BuildCreation{
		Recipe: []byte(`{}`),
		BuildPlan: []byte(`{
			"validation": {
				"buildable": true,
				"issues": null,
				"part_count": 1,
				"step_count": 1,
				"used_parts": null
			},
			"inventory": {"items": null}
		}`),
		Validation: []byte(`{
			"buildable": true,
			"issues": null,
			"part_count": 1,
			"step_count": 1,
			"used_parts": null
		}`),
	}

	response, err := toBuildCreationResponse(row)
	if err != nil {
		t.Fatal(err)
	}
	if response.Validation.Issues == nil || response.BuildPlan.Validation.Issues == nil {
		t.Fatal("validation issues must be canonical empty arrays")
	}
	if response.Validation.UsedParts == nil || response.BuildPlan.Validation.UsedParts == nil {
		t.Fatal("validation used_parts must be canonical empty objects")
	}
	if response.BuildPlan.Inventory.Items == nil {
		t.Fatal("inventory items must be a canonical empty array")
	}
}
