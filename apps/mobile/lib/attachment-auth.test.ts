import { describe, it, expect } from "vitest";
import { attachmentHeaders } from "./attachment-auth";
describe("private attachment authentication", () => {
  it("authenticates only same-origin attachment downloads", () => {
    expect(attachmentHeaders("https://chimii.com/api/attachments/1/download", "https://chimii.com", "token", "ws")).toEqual({ Authorization: "Bearer token", "X-Workspace-Slug": "ws" });
    for (const url of ["https://other.example/api/attachments/1/download", "https://chimii.com.evil.example/api/attachments/1/download", "https://chimii.com:444/api/attachments/1/download", "https://chimii.com/public/image.png", "https://user@chimii.com/api/attachments/1/download", "javascript:alert(1)"]) {
      expect(attachmentHeaders(url, "https://chimii.com", "token", "ws")).toEqual({});
    }
  });
  it("sends no credentials after logout", () => {
    expect(attachmentHeaders("https://chimii.com/api/attachments/1/download", "https://chimii.com", null, "ws")).toEqual({});
  });
});
