import { ActivityIndicator, ScrollView, View } from "react-native";
import { router, useLocalSearchParams } from "expo-router";
import { useQuery } from "@tanstack/react-query";
import { Text } from "@/components/ui/text";
import { IconButton } from "@/components/ui/icon-button";
import { inboxListOptions } from "@/data/queries/inbox";
import { useWorkspaceStore } from "@/data/workspace-store";
import { getAutopilotQuotaBody, getInboxDisplayTitle } from "@/lib/inbox-display";
export default function InboxNoticeDetail() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const wsId = useWorkspaceStore((s) => s.currentWorkspaceId);
  const { data: items, isLoading } = useQuery(inboxListOptions(wsId));

  // Read the raw workspace-scoped cache: deduplication can replace a row,
  // but a sheet already opened for a specific notification must remain stable.
  const item = items?.find(
    (candidate) => candidate.id === id && candidate.workspace_id === wsId,
  );
  const body = item ? getAutopilotQuotaBody(item) : null;

  return (
    <View className="flex-1 bg-background">
      <View className="flex-row items-center border-b border-border px-4 py-3">
        <Text className="flex-1 text-lg font-semibold text-foreground">
          {item ? getInboxDisplayTitle(item) : "Notification"}
        </Text>
        <IconButton
          name="close"
          variant="secondary"
          className="size-7 rounded-full"
          onPress={() => router.back()}
          accessibilityLabel="Close notification"
        />
      </View>

      {isLoading ? (
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator />
        </View>
      ) : !item ? (
        <View className="px-4 py-8">
          <Text className="text-sm text-muted-foreground text-center">
            This notification is no longer available.
          </Text>
        </View>
      ) : (
        <ScrollView
          className="flex-1"
          contentContainerClassName="gap-5 px-4 py-5"
          showsVerticalScrollIndicator={false}
        >
          {body ? (
            <Text className="text-base leading-6 text-foreground">
              {body}
            </Text>
          ) : null}

        </ScrollView>
      )}
    </View>
  );
}
