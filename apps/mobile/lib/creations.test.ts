import { describe, expect, it } from "vitest";
import { creationSummaries, creationWebURL } from "./creations";

describe("creation entry parity", () => {
 it("uses one-based display for Circuit and preserves unknown observations", () => {
  const rows=creationSummaries([], [{id:"c",title:"Radio",project_id:"p",observation:"future",current_step:0,created_at:"now"}]);
  expect(rows[0]?.progress).toBe("Saved at step 1 · Unknown result");
 });
 it("constructs web routes with no credentials or query token", () => {
  expect(creationWebURL("https://chimii.ai?token=secret","my space","build","one/two")).toBe("https://chimii.ai/my%20space/creations/one%2Ftwo");
  expect(creationWebURL("https://chimii.ai","w","circuit","c")).toBe("https://chimii.ai/w/circuit/c");
  expect(creationWebURL("https://user:password@chimii.ai","w","build","c")).toBeNull();
  expect(creationWebURL("javascript:alert(1)","w","build","c")).toBeNull();
  expect(creationWebURL(undefined,"w","build","c")).toBeNull();
 });
});
