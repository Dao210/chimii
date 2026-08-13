import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { BlockCatalogCard } from "./block-catalog-card";

const color = { code: 4, name: "Red", hex: "#b40000" };
const labels = { auto: "Auto-build ready", inventory: "Inventory ready", preview: "Preview only" };

describe("BlockCatalogCard", () => {
  it("renders certified capability and selects with a semantic button", () => {
    const onSelect = vi.fn();
    render(<BlockCatalogCard
      part={{ id: "brick-2x4", name: "Brick 2 x 4", category: "Bricks", popularity_rank: 17, certification_level: "certified", auto_build_eligible: true, inventory_eligible: true, ldraw_id: "3001.dat", studs_x: 4, studs_z: 2, plates_y: 3, quantity: 18 }}
      color={color}
      colorName="Red"
      selected={false}
      total={6}
      unlimited={false}
      labels={labels}
      onSelect={onSelect}
    />);

    expect(screen.getByText("Auto-build ready")).toBeInTheDocument();
    expect(screen.getByText("#17 · Bricks")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button"));
    expect(onSelect).toHaveBeenCalledTimes(1);
  });

  it("labels uncertified parts as preview only", () => {
    render(<BlockCatalogCard
      part={{ id: "ldraw-123", name: "Special Part", category: "Other", popularity_rank: 80, certification_level: "asset_only", auto_build_eligible: false, inventory_eligible: false, ldraw_id: "123.dat", studs_x: 0, studs_z: 0, plates_y: 0, quantity: 0 }}
      color={color}
      colorName="Red"
      selected
      total={0}
      unlimited
      labels={labels}
      onSelect={() => {}}
    />);

    expect(screen.getByText("Preview only")).toBeInTheDocument();
    expect(screen.getByText("∞")).toBeInTheDocument();
  });
});
