import type { TimelineEntry } from "@chimii/core/types";

export function commentContentFromTimeline(
  timeline: TimelineEntry[] | undefined,
  commentId: string,
): string | undefined {
  return timeline?.find(
    (entry) => entry.type === "comment" && entry.id === commentId,
  )?.content;
}

export function buildCommentUpdateBody(
  content: string,
  attachmentIds: string[] | undefined,
) {
  return {
    content,
    ...(attachmentIds ? { attachment_ids: attachmentIds } : {}),
  };
}
