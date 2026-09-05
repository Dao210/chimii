import { describe, it, expect } from "vitest";
import { buildIssueStatusCatalog, statusOptions, STATUS_CATEGORIES, issueColumnCategory, issueBehavesAsAny, CLOSED_CATEGORIES } from "./issue-status";

describe("Chimii status contract", () => {
  it("offers only the seven server-supported statuses without a catalog request", () => {
    const catalog = buildIssueStatusCatalog();
    expect(catalog.isLoaded).toBe(true);
    expect(statusOptions(catalog).map((s) => s.key)).toEqual(STATUS_CATEGORIES);
    expect(catalog.labelOf("in_progress")).toBe("In Progress");
  });
  it("keeps an unknown server value visible without treating it as closed", () => {
    const issue = { status: "future_status" };
    expect(issueColumnCategory(issue)).toBe("todo");
    expect(buildIssueStatusCatalog().labelOf(issue.status)).toBe("future_status");
    expect(issueBehavesAsAny(issue, CLOSED_CATEGORIES)).toBe(false);
  });
  it("recognizes both completed and cancelled tasks as closed", () => {
    for (const status of ["done", "cancelled"]) expect(issueBehavesAsAny({ status }, CLOSED_CATEGORIES)).toBe(true);
  });
});
