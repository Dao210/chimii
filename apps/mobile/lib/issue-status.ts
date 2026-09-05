import type { IssuePriority, IssueStatus } from "@chimii/core/types";


export type IssueStatusCategory = IssueStatus;
export type StatusSource = { status: string; status_category?: string | null };
export const STATUS_CATEGORIES: IssueStatusCategory[] = [
  "backlog", "todo", "in_progress", "in_review", "done", "blocked", "cancelled",
];
export const BOARD_CATEGORIES = STATUS_CATEGORIES.filter((s) => s !== "cancelled");
export const CLOSED_CATEGORIES: readonly IssueStatusCategory[] = ["done", "cancelled"];
export const STATUS_LABEL: Record<IssueStatus, string> = {
  backlog: "Backlog", todo: "Todo", in_progress: "In Progress", in_review: "In Review",
  done: "Done", blocked: "Blocked", cancelled: "Cancelled",
};
export const PRIORITY_LABEL: Record<IssuePriority, string> = {
  none: "No priority", low: "Low", medium: "Medium", high: "High", urgent: "Urgent",
};
export function isIssueStatusCategory(value: string): value is IssueStatusCategory {
  return STATUS_CATEGORIES.some((status) => status === value);
}
export function statusCategoryOfKey(key: string): IssueStatusCategory {
  return isIssueStatusCategory(key) ? key : "todo";
}
// Unknown values remain visible. They never acquire closed-state semantics.
export function issueStatusCategory(issue: StatusSource): IssueStatusCategory | null {
  if (issue.status_category && isIssueStatusCategory(issue.status_category)) return issue.status_category;
  return isIssueStatusCategory(issue.status) ? issue.status : null;
}
export function issueColumnCategory(issue: StatusSource): IssueStatusCategory {
  return issueStatusCategory(issue) ?? "todo";
}
export function issueBehavesAs(issue: StatusSource, category: IssueStatusCategory) {
  return issueStatusCategory(issue) === category;
}
export function issueBehavesAsAny(issue: StatusSource, categories: readonly IssueStatusCategory[]) {
  const category = issueStatusCategory(issue);
  return category !== null && categories.includes(category);
}
export interface StatusOption {
  key: IssueStatus; category: IssueStatusCategory; label: string; color: string | null;
}
export interface IssueStatusEntry {
  key: IssueStatus; category: IssueStatusCategory; name: string;
  color: string; is_system: boolean; archived_at: string | null;
}
export interface IssueStatusCatalog {
  statuses: IssueStatusEntry[]; activeStatuses: IssueStatusEntry[];
  categoryOf: (key: string) => IssueStatusCategory;
  labelOf: (key: string) => string;
  entryOf: (key: string) => IssueStatusEntry | undefined;
  colorOf: (key: string) => string | null;
  inCategory: (category: IssueStatusCategory) => IssueStatusEntry[];
  isLoaded: boolean;
}
export function buildIssueStatusCatalog(): IssueStatusCatalog {
  const statuses = STATUS_CATEGORIES.map((key) => ({
    key, category: key, name: STATUS_LABEL[key], color: "", is_system: true, archived_at: null,
  }));
  return {
    statuses, activeStatuses: statuses, categoryOf: statusCategoryOfKey,
    labelOf: (key) => isIssueStatusCategory(key) ? STATUS_LABEL[key] : key,
    entryOf: (key) => statuses.find((entry) => entry.key === key),
    colorOf: () => null, inCategory: (category) => statuses.filter((s) => s.key === category),
    isLoaded: true,
  };
}
export function isCustomStatus(catalog: IssueStatusCatalog, key: string) {
  return catalog.entryOf(key)?.is_system === false;
}
export function statusOptions(catalog: IssueStatusCatalog): StatusOption[] {
  return STATUS_CATEGORIES.map((key) => ({key, category: key, label: catalog.labelOf(key), color: null}));
}
