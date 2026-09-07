import { beforeEach, describe, expect, it, vi } from "vitest";
const fs = vi.hoisted(() => ({
  getInfoAsync: vi.fn(),
  readAsStringAsync: vi.fn(),
  writeAsStringAsync: vi.fn(),
  makeDirectoryAsync: vi.fn(),
  moveAsync: vi.fn(),
  deleteAsync: vi.fn(),
}));
vi.mock("expo-file-system/legacy", () => ({
  documentDirectory: "file:///private/",
  ...fs,
}));
import {
  readInventoryDraft,
  writeInventoryDraft,
} from "./maker-inventory-storage";
beforeEach(() => {
  vi.clearAllMocks();
  fs.getInfoAsync.mockResolvedValue({ exists: true });
});
describe("inventory draft files", () => {
  it("round-trips both kinds of unsubmitted inventory with their base revisions", async () => {
    const draft = {
      bricks: {
        revision: 8,
        items: [{ part_id: "brick", color: 4, quantity: 12 }],
      },
      circuits: { kit: { revision: 4, quantities: { lamp: 1 } } },
    };
    await writeInventoryDraft("account_workspace", draft);
    const raw = fs.writeAsStringAsync.mock.calls[0]![1];
    fs.readAsStringAsync.mockResolvedValue(raw);
    expect(await readInventoryDraft("account_workspace")).toEqual(draft);
    expect(fs.moveAsync).toHaveBeenCalledWith({
      from: "file:///private/maker-drafts/account_workspace.json.tmp",
      to: "file:///private/maker-drafts/account_workspace.json",
    });
  });
  it("does not restore an incompatible version or invalid quantities", async () => {
    for (const value of [
      { version: 2 },
      {
        version: 1,
        bricks: {
          revision: 1,
          items: [{ part_id: "brick", color: 4, quantity: -2 }],
        },
      },
    ]) {
      fs.readAsStringAsync.mockResolvedValue(JSON.stringify(value));
      expect(await readInventoryDraft("scope")).toEqual({});
    }
  });
  it("removes a submitted draft instead of bringing it back on next launch", async () => {
    await writeInventoryDraft("scope", { bricks: undefined, circuits: {} });
    expect(fs.deleteAsync).toHaveBeenCalledWith(
      "file:///private/maker-drafts/scope.json",
      { idempotent: true },
    );
    expect(fs.writeAsStringAsync).not.toHaveBeenCalled();
  });
});
