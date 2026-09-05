import { buildIssueStatusCatalog } from "@/lib/issue-status";

// Mirrors packages/core/types/issue.ts: Chimii has seven fixed statuses.
const catalog = buildIssueStatusCatalog();
export function useIssueStatuses() { return catalog; }
