import { describe, expect, it } from "vitest";
import type { AgentRuntime } from "@chimii/core/types";
import { pickAutomaticRuntime } from "./step-runtime-auto-connect";

function runtime(
  id: string,
  runtimeMode: AgentRuntime["runtime_mode"],
  status: AgentRuntime["status"] = "online",
): AgentRuntime {
  return {
    id,
    runtime_mode: runtimeMode,
    status,
  } as AgentRuntime;
}

describe("pickAutomaticRuntime", () => {
  it("prefers an online remote runtime over an online local runtime", () => {
    const selected = pickAutomaticRuntime([
      runtime("local", "local"),
      runtime("remote", "cloud"),
    ]);

    expect(selected?.id).toBe("remote");
  });

  it("supports an online local runtime when no remote runtime is ready", () => {
    const selected = pickAutomaticRuntime([
      runtime("remote-offline", "cloud", "offline"),
      runtime("local", "local"),
    ]);

    expect(selected?.id).toBe("local");
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
