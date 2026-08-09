package handler

import "testing"

func TestNormalizeBrickInventoryItems(t *testing.T) {
	items, err := normalizeBrickInventoryItems([]brickInventoryItemRequest{
		{PartID: "plate-1x2", Color: 14, Quantity: 0},
		{PartID: "brick-2x4", Color: 4, Quantity: 6},
		{PartID: "brick-1x1", Color: 1, Quantity: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].PartID != "brick-1x1" || items[1].PartID != "brick-2x4" {
		t.Fatalf("normalized items = %#v", items)
	}
	if items[0].Quantity != 2 || items[1].Quantity != 6 {
		t.Fatalf("normalized quantities = %#v", items)
	}
}

func TestNormalizeBrickInventoryItemsRejectsInvalidEntries(t *testing.T) {
	for name, items := range map[string][]brickInventoryItemRequest{
		"unknown part":  {{PartID: "imaginary", Color: 4, Quantity: 1}},
		"unknown color": {{PartID: "brick-2x4", Color: 999, Quantity: 1}},
		"too many":      {{PartID: "brick-2x4", Color: 4, Quantity: 1000}},
		"duplicate": {
			{PartID: "brick-2x4", Color: 4, Quantity: 1},
			{PartID: "brick-2x4", Color: 4, Quantity: 2},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := normalizeBrickInventoryItems(items); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
