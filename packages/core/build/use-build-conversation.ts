import { useRef } from "react";
import { useInfiniteQuery } from "@tanstack/react-query";
import { generateUUID } from "../utils";
import { buildConversationOptions } from "./queries";
import { useCancelBuildSession, useSendBuildMessage } from "./mutations";
import type { BuildMessageInput, CircuitBuildInput } from "./types";

export type CreationKind = "brick" | "circuit";
export interface CreationSource {
  kind: CreationKind;
  id: string;
  hash: string;
}

// The route owns the domain and its materials. This hook only manages a turn.
export function useBuildConversation(
  wsId: string,
  kind: CreationKind,
  conversationId = "",
) {
  const history = useInfiniteQuery(
    buildConversationOptions(wsId, conversationId),
  );
  const send = useSendBuildMessage(wsId, conversationId || undefined);
  const cancel = useCancelBuildSession();
  const inFlight = useRef(false);
  const session = history.data?.pages[0]?.session ?? send.data;
  const messages =
    history.data?.pages
      .slice()
      .reverse()
      .flatMap((page) => page.messages) ?? [];
  const latestResult = history.data?.pages[0]?.result;
  const foreign = kind === "brick" ? "circuit" : "brick";
  const readOnly =
    !!session &&
    (session.kind === "auto" ||
      session.kind === foreign ||
      latestResult?.kind === foreign ||
      messages.some(
        (message) =>
          message.metadata.kind === foreign ||
          (kind === "brick"
            ? !!message.metadata.circuit_creation_id
            : !!message.metadata.creation_id),
      ));
  const resultId =
    kind === "brick"
      ? session?.creation_id || latestResult?.creation_id || ""
      : session?.circuit_creation_id || latestResult?.circuit_creation_id || "";
  const isWorking =
    !session?.expired &&
    (session?.status === "queued" || session?.status === "generating");
  const isQuestion =
    !readOnly &&
    !session?.expired &&
    session?.status === "clarifying" &&
    !!session.question;
  const disabled =
    readOnly ||
    isWorking ||
    send.isPending ||
    cancel.isPending ||
    (!!conversationId && !session);

  const submit = async (input: BuildMessageInput) => {
    if (inFlight.current) return;
    inFlight.current = true;
    try {
      return await send.mutateAsync(input);
    } catch {
      // Keep the exact mutation variables for replay after a lost response.
      return undefined;
    } finally {
      inFlight.current = false;
    }
  };
  const sendMessage = (
    prompt: string,
    options: { circuit?: CircuitBuildInput; source?: CreationSource } = {},
  ) => {
    if (disabled || !prompt.trim()) return Promise.resolve(undefined);
    const source =
      !isQuestion && options.source?.kind === kind ? options.source : undefined;
    return submit({
      prompt: prompt.trim(),
      client_request_id: generateUUID(),
      kind,
      ...(kind === "circuit" && options.circuit
        ? { circuit: options.circuit }
        : {}),
      ...(session
        ? {
            expected_session_id: session.id,
            expected_revision: session.revision,
            ...(isQuestion ? { question_id: session.question?.id } : {}),
          }
        : {}),
      ...(source
        ? {
            source_kind: source.kind,
            source_creation_id: source.id,
            expected_content_hash: source.hash,
          }
        : {}),
    });
  };
  const retry = () =>
    send.variables && !disabled
      ? submit(send.variables)
      : Promise.resolve(undefined);
  const cancelTurn = async () => {
    if (!session || cancel.isPending || send.isPending) return;
    try {
      return await cancel.mutateAsync({
        sessionId: session.id,
        revision: session.revision ?? 1,
      });
    } catch {
      return undefined;
    }
  };
  return {
    history,
    messages,
    session,
    resultId,
    readOnly,
    isWorking,
    isQuestion,
    disabled,
    send,
    cancel,
    sendMessage,
    retry,
    cancelTurn,
  };
}

export type BuildConversationController = ReturnType<
  typeof useBuildConversation
>;
