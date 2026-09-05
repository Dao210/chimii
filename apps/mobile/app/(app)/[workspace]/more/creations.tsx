import { useState } from "react";
import { ActivityIndicator, FlatList, Pressable, RefreshControl, View } from "react-native";
import { Image } from "expo-image";
import { router } from "expo-router";
import { useQuery } from "@tanstack/react-query";
import { SafeAreaView } from "react-native-safe-area-context";
import { Text } from "@/components/ui/text";
import { Button } from "@/components/ui/button";
import { useWorkspaceStore } from "@/data/workspace-store";
import { brickCreationsOptions, circuitCreationsOptions } from "@/data/queries/creations";
import { creationSummaries, type CreationKind } from "@/lib/creations";
import { useColorScheme } from "@/lib/use-color-scheme";
import { THEME } from "@/lib/theme";

export default function CreationsPage() {
  const wsId = useWorkspaceStore(s => s.currentWorkspaceId);
  const slug = useWorkspaceStore(s => s.currentWorkspaceSlug);
  const [kind, setKind] = useState<CreationKind>("build");
  const builds = useQuery(brickCreationsOptions(wsId, kind === "build"));
  const circuits = useQuery(circuitCreationsOptions(wsId, kind === "circuit"));
  const query = kind === "build" ? builds : circuits;
  const rows = kind === "build" ? creationSummaries(builds.data?.creations) : creationSummaries([], circuits.data?.creations);
  const { colorScheme } = useColorScheme();
  return <SafeAreaView className="flex-1 bg-background" edges={["bottom"]}>
    <View className="flex-row gap-2 px-4 py-3">
      <Button variant={kind === "build" ? "default" : "outline"} onPress={() => setKind("build")} accessibilityState={{ selected: kind === "build" }}><Text>Bricks</Text></Button>
      <Button variant={kind === "circuit" ? "default" : "outline"} onPress={() => setKind("circuit")} accessibilityState={{ selected: kind === "circuit" }}><Text>Circuits</Text></Button>
    </View>
    <Text className="px-4 pb-3 text-xs text-muted-foreground">{kind === "build" ? "Your 60 most recent brick creations" : "Your 50 most recent circuit creations"}</Text>
    {query.isLoading ? <ActivityIndicator accessibilityLabel="Loading creations" /> : query.isError ? <View className="p-4 gap-3"><Text className="text-destructive">Could not load creations.</Text><Button variant="outline" onPress={() => void query.refetch()}><Text>Retry</Text></Button></View> :
      <FlatList data={rows} keyExtractor={v => v.id}
        refreshControl={<RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} />}
        ListEmptyComponent={<Text className="p-4 text-muted-foreground">No creations yet. Create one in Chimii on the web to find it here.</Text>}
        ItemSeparatorComponent={() => <View className="ml-4 h-px bg-border" />}
        renderItem={({ item }) => <Pressable accessibilityRole="button" accessibilityLabel={`${item.title}, ${item.progress}`} onPress={() => slug && router.push(`/${slug}/creation/${kind}/${item.id}`)} className="flex-row items-center gap-3 px-4 py-4 active:bg-muted">
          <Image source={kind === "build" ? "sf:cube.box" : "sf:bolt"} tintColor={THEME[colorScheme].mutedForeground} style={{ width: 24, height: 24 }} />
          <View className="flex-1 gap-1"><Text className="font-medium" numberOfLines={2}>{item.title}</Text><Text className="text-xs text-muted-foreground">{item.description}</Text><Text className="text-xs text-muted-foreground">{item.progress}</Text></View>
          <Image source="sf:chevron.right" tintColor={THEME[colorScheme].mutedForeground} style={{ width: 12, height: 12 }} />
        </Pressable>} />}
  </SafeAreaView>;
}
