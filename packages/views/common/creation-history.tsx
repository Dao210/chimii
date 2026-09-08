"use client";

import { Button } from "@chimii/ui/components/ui/button";
import type {
  BuildConversationController,
  BuildSession,
} from "@chimii/core/build";
import { useWorkspacePaths } from "@chimii/core/paths";
import { AppLink } from "../navigation";
import { useT } from "../i18n";

export function CreationHistory({
  conversation,
}: {
  conversation: BuildConversationController;
}) {
  const { t } = useT("build");
  const paths = useWorkspacePaths();
  const { history, messages, session } = conversation;
  if (!session && !history.isError) return null;
  return (
    <details className="mt-6 rounded-2xl border border-border bg-card p-4 text-foreground">
      <summary className="cursor-pointer text-sm font-bold">
        {t(($) => $.conversation_history)}
      </summary>
      {history.isError && (
        <p role="alert" className="mt-3">
          {t(($) => $.session_fetch_error)}{" "}
          <Button variant="outline" onClick={() => void history.refetch()}>
            {t(($) => $.retry_fetch)}
          </Button>
        </p>
      )}
      {history.hasNextPage && (
        <Button
          className="mt-3"
          variant="ghost"
          disabled={history.isFetchingNextPage}
          onClick={() => void history.fetchNextPage()}
        >
          {t(($) => $.conversation_older)}
        </Button>
      )}
      <ol
        className="mt-4 space-y-4"
        aria-label={t(($) => $.conversation_history)}
      >
        {messages.map((message) => (
          <li key={message.id} className="border-l-2 border-border pl-4">
            <p className="text-xs font-bold text-muted-foreground">
              {message.role === "user"
                ? t(($) => $.conversation_you)
                : t(($) => $.conversation_assistant)}
            </p>
            <p className="mt-1 whitespace-pre-wrap break-words text-sm leading-6">
              {message.kind === "error" && !message.metadata.message
                ? message.metadata.error === "BUILD_PLANNER_TIMEOUT"
                  ? t(($) => $.failed_timeout)
                  : t(($) => $.failed_description)
                : message.content}
            </p>
            {(message.metadata.creation_id ||
              message.metadata.circuit_creation_id) && (
              <AppLink
                className="mt-2 inline-flex text-sm font-semibold underline underline-offset-4"
                href={
                  message.metadata.creation_id
                    ? paths.creationDetail(message.metadata.creation_id)
                    : paths.circuitDetail(message.metadata.circuit_creation_id!)
                }
              >
                {t(($) => $.conversation_open)}
              </AppLink>
            )}
          </li>
        ))}
        {!messages.length && session && (
          <li className="whitespace-pre-wrap text-sm">{session.prompt}</li>
        )}
      </ol>
    </details>
  );
}

export function CreationRequestNotice({
  conversation,
  onSent,
}: {
  conversation: BuildConversationController;
  onSent: (session: BuildSession) => void;
}) {
  const { t } = useT("build");
  return (
    <>
      {conversation.send.isPending && (
        <p
          role="status"
          className="my-4 whitespace-pre-wrap break-words text-sm"
        >
          {conversation.send.variables?.prompt} ·{" "}
          {t(($) => $.conversation_sending)}
        </p>
      )}
      {conversation.send.isError && (
        <div
          role="alert"
          className="my-4 rounded-xl border border-destructive/40 bg-card p-4 text-foreground"
        >
          <p className="whitespace-pre-wrap break-words text-sm">
            {conversation.send.variables?.prompt}
          </p>
          <p className="mt-2 text-sm">{t(($) => $.conversation_send_error)}</p>
          <Button
            variant="outline"
            className="mt-3"
            disabled={conversation.disabled}
            onClick={async () => {
              const next = await conversation.retry();
              if (next) onSent(next);
            }}
          >
            {t(($) => $.conversation_retry)}
          </Button>
        </div>
      )}
      {conversation.cancel.isError && (
        <p role="alert" className="my-3 text-sm">
          {t(($) => $.cancel_error)}
        </p>
      )}
    </>
  );
}
