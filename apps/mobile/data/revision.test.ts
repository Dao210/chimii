import type { TimelineEntry } from "@chimii/core/types";
import { describe, expect, it } from "vitest";

import {
  buildCommentUpdateBody,
  commentContentFromTimeline,
} from "./revision";

describe("mobile comment requests", () => {
  it("reads existing content and sends only fields supported by Chimii", () => {
    const timeline = [
      {
        type: "comment",
        id: "comment-1",
        content: "Original",
      } as TimelineEntry,
    ];

    expect(commentContentFromTimeline(timeline, "comment-1")).toBe("Original");
    expect(buildCommentUpdateBody("Latest", ["attachment-1"])).toEqual({
      content: "Latest",
      attachment_ids: ["attachment-1"],
    });
  });

});
