import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { circuitFixture } from "./test-fixture";
import { renderWithI18n } from "../test/i18n";
import { CircuitBehaviorPreview } from "./circuit-behavior-preview";

afterEach(cleanup);
it("explores the checked signal states without changing the saved design", () => {
  const { creation } = circuitFixture("boson-button-light");
  creation.document.composition = {
    inputs: ["BOS0002-R"],
    operation: "direct",
    output: "BOS0017-R",
  };
  creation.document.behavior = {
    model: "boson-digital-v1",
    passed: true,
    cases: [
      {
        inputs: { "BOS0002-R": false },
        powered: true,
        expected: false,
        actual: false,
      },
      {
        inputs: { "BOS0002-R": true },
        powered: true,
        expected: true,
        actual: true,
      },
      {
        inputs: { "BOS0002-R": true },
        powered: false,
        expected: false,
        actual: false,
      },
    ],
  };
  const before = JSON.stringify(creation);
  renderWithI18n(<CircuitBehaviorPreview document={creation.document} />);
  expect(screen.getByRole("status")).toHaveTextContent("Off");
  fireEvent.click(screen.getByRole("button", { name: /i2r/ }));
  expect(screen.getByRole("status")).toHaveTextContent("On");
  fireEvent.click(screen.getByRole("button", { name: /Preview power/ }));
  expect(screen.getByRole("status")).toHaveTextContent("Off");
  expect(JSON.stringify(creation)).toBe(before);
});
