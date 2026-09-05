import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  openURL: vi.fn(),
  attachmentHeaders: vi.fn(),
  makeDirectoryAsync: vi.fn(),
  readDirectoryAsync: vi.fn(),
  downloadAsync: vi.fn(),
  deleteAsync: vi.fn(),
  shareAsync: vi.fn(),
}));

vi.mock("react-native", () => ({ Linking: { openURL: mocks.openURL } }));
vi.mock("@/data/api", () => ({ api: { attachmentHeaders: mocks.attachmentHeaders } }));
vi.mock("./attachment-url", () => ({ resolveAttachmentUrl: (url: string) => url }));
vi.mock("expo-file-system/legacy", () => ({ cacheDirectory: "file:///cache/", ...mocks }));
vi.mock("expo-sharing", () => ({ isAvailableAsync: async () => true, shareAsync: mocks.shareAsync }));

import { openAttachment } from "./open-attachment";

describe("native attachment sharing", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.attachmentHeaders.mockReturnValue({ Authorization: "Bearer session" });
    mocks.readDirectoryAsync.mockResolvedValue([]);
    mocks.downloadAsync.mockResolvedValue({ status: 200, uri: "file:///cache/download.pdf" });
  });

  it("keeps a shared file readable after Android dismisses the chooser", async () => {
    await openAttachment("https://chimii.com/api/attachments/1/download", "report.pdf");
    expect(mocks.shareAsync).toHaveBeenCalledWith("file:///cache/download.pdf");
    expect(mocks.deleteAsync).not.toHaveBeenCalled();
  });

  it("removes a failed download without opening the share picker", async () => {
    mocks.downloadAsync.mockResolvedValue({ status: 401 });
    await expect(openAttachment("https://chimii.com/api/attachments/1/download")).rejects.toThrow("401");
    expect(mocks.shareAsync).not.toHaveBeenCalled();
    expect(mocks.deleteAsync).toHaveBeenCalledOnce();
  });

  it("prunes expired cached shares while preserving recent files", async () => {
    mocks.readDirectoryAsync.mockResolvedValue(["1-old", `${Date.now()}-recent`]);
    await openAttachment("https://chimii.com/api/attachments/1/download");
    expect(mocks.deleteAsync).toHaveBeenCalledExactlyOnceWith("file:///cache/attachments/1-old", { idempotent: true });
  });
});
