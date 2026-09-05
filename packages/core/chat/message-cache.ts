import type { InfiniteData, QueryClient } from "@tanstack/react-query";
import { chatKeys } from "./queries";
import type { ChatMessage, ChatMessagesPage } from "../types";


function mergeChatMessage(existing: ChatMessage, incoming: ChatMessage): ChatMessage {
  const fills: Partial<ChatMessage> = {};
  let filled = false;
  for (const key of Object.keys(incoming) as (keyof ChatMessage)[]) {
    if (existing[key] !== undefined || incoming[key] === undefined) continue;
    Object.assign(fills, { [key]: incoming[key] });
    filled = true;
  }
  // Reference-stable when nothing was filled, so observers do not re-render.
  return filled ? { ...existing, ...fills } : existing;
}

function mergeIntoList(messages: ChatMessage[], message: ChatMessage): ChatMessage[] {
  const index = messages.findIndex((m) => m.id === message.id);
  if (index < 0) return messages;
  const merged = mergeChatMessage(messages[index]!, message);
  if (merged === messages[index]) return messages;
  return messages.map((m, i) => (i === index ? merged : m));
}

function upsertIntoList(messages: ChatMessage[], message: ChatMessage): ChatMessage[] {
  const merged = mergeIntoList(messages, message);
  if (merged !== messages) return merged;
  return messages.some((m) => m.id === message.id) ? messages : [...messages, message];
}

function upsertIntoPages(
  data: InfiniteData<ChatMessagesPage>,
  message: ChatMessage,
): InfiniteData<ChatMessagesPage> {
  const seen = data.pages.some((page) => page.messages.some((m) => m.id === message.id));
  if (seen) {
    // Merge in place, wherever the row already lives. Appending here instead
    // would drop a copy into every OTHER page.
    let changed = false;
    const pages = data.pages.map((page) => {
      const next = mergeIntoList(page.messages, message);
      if (next === page.messages) return page;
      changed = true;
      return { ...page, messages: next };
    });
    return changed ? { ...data, pages } : data;
  }
  // Page 0 is the newest window (later pages are older cursor pages), so a new
  // message belongs at the end of it.
  return {
    ...data,
    pages: data.pages.map((page, index) =>
      index === 0 ? { ...page, messages: [...page.messages, message] } : page,
    ),
  };
}

function seedPage(message: ChatMessage): InfiniteData<ChatMessagesPage> {
  return {
    pages: [{ messages: [message], limit: 50, has_more: false, next_cursor: null }],
    pageParams: [null],
  };
}

export interface UpsertChatMessageOptions {
    seedIfMissing?: boolean;
}

export function upsertChatMessageToCaches(
  qc: QueryClient,
  sessionId: string,
  message: ChatMessage,
  { seedIfMissing = false }: UpsertChatMessageOptions = {},
): void {
  qc.setQueryData<ChatMessage[] | undefined>(chatKeys.messages(sessionId), (old) => {
    if (!old) return seedIfMissing ? [message] : old;
    return upsertIntoList(old, message);
  });
  qc.setQueryData<InfiniteData<ChatMessagesPage> | undefined>(
    chatKeys.messagesPage(sessionId),
    (old) => {
      if (!old?.pages.length) return seedIfMissing ? seedPage(message) : old;
      return upsertIntoPages(old, message);
    },
  );
}
