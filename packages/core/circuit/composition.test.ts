import { describe, expect, it } from "vitest";
import { CircuitCreationSchema } from "./schemas";
import { compositionFixture } from "./test-fixture";

describe("functional composition response", () => {
  it("reads a generated combination with exhaustive logic evidence", () => {
    const result = CircuitCreationSchema.parse(compositionFixture());
    expect(result.document.composition?.output).toBe("BOS0021");
    expect(result.document.behavior?.cases).toHaveLength(3);
  });

  it.each([
    "missing behavior",
    "wrong result",
    "duplicate state",
    "unknown model",
    "missing composition",
    "missing input",
    "wrong operation",
    "wrong kit",
  ])("rejects %s", (defect) => {
    const raw = JSON.parse(JSON.stringify(compositionFixture()));
    const doc = raw.document;
    switch (defect) {
      case "missing behavior":
        delete doc.behavior;
        break;
      case "wrong result":
        doc.behavior.cases[0].actual = true;
        break;
      case "duplicate state":
        doc.behavior.cases[1] = doc.behavior.cases[0];
        break;
      case "unknown model":
        doc.behavior.model = "physical-certification";
        break;
      case "missing composition":
        delete doc.composition;
        break;
      case "missing input":
        doc.behavior.cases[0].inputs = {};
        break;
      case "wrong operation":
        doc.composition.operation = "not";
        break;
      case "wrong kit":
        doc.kit_id = "other-boson-kit";
        break;
    }
    expect(CircuitCreationSchema.safeParse(raw).success).toBe(false);
  });
});
