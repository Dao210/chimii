import { useChatSessionPickerStore } from "./stores/chat-session-picker-store";
import { useReplyTargetStore } from "./stores/reply-target-store";
import { useFailedCommentsStore } from "./stores/failed-comments-store";
import { useChatDraftsStore } from "./stores/chat-drafts-store";
import { useNewIssueDraftStore } from "./stores/new-issue-draft-store";
import { useNewProjectDraftStore } from "./stores/new-project-draft-store";
import { useMentionDraftStore } from "./stores/mention-draft-store";
import { useChatSelectStore } from "./chat-select-store";
import { useCommentSelectStore } from "./comment-select-store";

/** These drafts/selections belong to an account, unlike appearance preferences. */
export function resetSessionState() {
  useChatSessionPickerStore.setState(useChatSessionPickerStore.getInitialState(), true);
  useReplyTargetStore.setState(useReplyTargetStore.getInitialState(), true);
  useFailedCommentsStore.setState(useFailedCommentsStore.getInitialState(), true);
  useChatDraftsStore.setState(useChatDraftsStore.getInitialState(), true);
  useNewIssueDraftStore.setState(useNewIssueDraftStore.getInitialState(), true);
  useNewProjectDraftStore.setState(useNewProjectDraftStore.getInitialState(), true);
  useMentionDraftStore.setState(useMentionDraftStore.getInitialState(), true);
  useChatSelectStore.setState(useChatSelectStore.getInitialState(), true);
  useCommentSelectStore.setState(useCommentSelectStore.getInitialState(), true);
}
