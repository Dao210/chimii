import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useWorkspaceId } from "../hooks";
import { buildKeys } from "./queries";
import { generateUUID } from "../utils";

export function useCreateBuildSession() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ prompt, clientRequestId = generateUUID() }: { prompt: string; clientRequestId?: string }) =>
      api.createBuildSession(prompt, clientRequestId),
    onSuccess: (session) => queryClient.setQueryData(buildKeys.session(workspaceId, session.id), session),
  });
}

export function useSubmitBuildAnswers() {
  const workspaceId = useWorkspaceId();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ sessionId, answers }: { sessionId: string; answers: Record<string, string> }) =>
      api.submitBuildAnswers(sessionId, answers),
    onSuccess: (session) => queryClient.setQueryData(buildKeys.session(workspaceId, session.id), session),
  });
}
