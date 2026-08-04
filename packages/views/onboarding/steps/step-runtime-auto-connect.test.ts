import { describe, expect, it } from "vitest";
import type { AgentRuntime } from "@chimii/core/types";
import { pickAutomaticRuntime } from "./step-runtime-auto-connect";

function runtime(
  id: string,
  runtimeMode: AgentRuntime["runtime_mode"],
  status: AgentRuntime["status"] = "online",
  executionType: AgentRuntime["execution_type"] = "cli",
): AgentRuntime {
  return {
    id,
    runtime_mode: runtimeMode,
    execution_type: executionType,
    status,
  } as AgentRuntime;
}

describe("pickAutomaticRuntime", () => {
  it("defaults to an online local CLI runtime", () => {
    const selected = pickAutomaticRuntime([
      runtime("local", "local"),
      runtime("managed", "cloud", "online", "cloud"),
    ]);

    expect(selected?.id).toBe("local");
  });

  it("supports an online Fleet CLI runtime when no local CLI is ready", () => {
    const selected = pickAutomaticRuntime([
      runtime("fleet", "cloud"),
      runtime("managed", "cloud", "online", "cloud"),
    ]);

    expect(selected?.id).toBe("fleet");
  });

  it("selects only managed Cloud execution when Cloud is the default", () => {
    const selected = pickAutomaticRuntime(
      [
        runtime("local", "local"),
        runtime("managed", "cloud", "online", "cloud"),
      ],
      "cloud",
    );

    expect(selected?.id).toBe("managed");
  });

  it("does not implicitly fall back from Cloud execution to CLI", () => {
    expect(pickAutomaticRuntime([runtime("local", "local")], "cloud")).toBeNull();
  });

  it("does not connect an offline runtime", () => {
    expect(
      pickAutomaticRuntime([
        runtime("remote", "cloud", "offline"),
        runtime("local", "local", "offline"),
      ]),
    ).toBeNull();
  });
});
