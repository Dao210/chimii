import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { BrickThumbnail } from "./brick-thumbnail";

describe("BrickThumbnail", () => {
  it("renders the generated high-fidelity asset for the requested part and colour", () => {
    const { container } = render(
      <BrickThumbnail ldrawID="3001.dat" colorCode={4} alt="红色 2×4 积木块" eager />,
    );

    const stage = container.querySelector("[data-thumbnail-source='ldraw-glb']");
    const image = screen.getByAltText("红色 2×4 积木块");
    expect(stage).toHaveAttribute("data-ldraw-id", "3001.dat");
    expect(stage).toHaveAttribute("data-color-code", "4");
    expect(image).toHaveAttribute("loading", "eager");
    expect(image).toHaveAttribute("width", "640");
    expect(image).toHaveAttribute("height", "400");

    fireEvent.load(image);
    expect(image).toHaveClass("opacity-100");
  });

  it("switches image assets when the selected colour changes", () => {
    const { rerender } = render(
      <BrickThumbnail ldrawID="4624c04.dat" colorCode={1} alt="蓝色车轮" />,
    );
    const blueSource = screen.getByAltText("蓝色车轮").getAttribute("src");

    rerender(<BrickThumbnail ldrawID="4624c04.dat" colorCode={14} alt="黄色车轮" />);
    const yellowSource = screen.getByAltText("黄色车轮").getAttribute("src");
    expect(yellowSource).not.toBe(blueSource);
  });

  it("never falls back to a misleading approximate silhouette", () => {
    const { container } = render(
      <BrickThumbnail ldrawID="missing.dat" colorCode={4} alt="积木块预览不可用" />,
    );

    expect(container.querySelector("img")).toBeNull();
    expect(container.querySelector("[data-thumbnail-source='missing']")).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "积木块预览不可用" })).toBeInTheDocument();
  });
});
