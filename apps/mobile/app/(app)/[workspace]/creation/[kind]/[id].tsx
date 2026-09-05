import { ActivityIndicator, Alert, Linking, ScrollView, View } from "react-native";
import { useLocalSearchParams } from "expo-router";
import { useQuery } from "@tanstack/react-query";
import { SafeAreaView } from "react-native-safe-area-context";
import { Text } from "@/components/ui/text";
import { Button } from "@/components/ui/button";
import { useWorkspaceStore } from "@/data/workspace-store";
import { brickCreationsOptions, circuitCreationsOptions } from "@/data/queries/creations";
import { creationSummaries, creationWebURL } from "@/lib/creations";

export default function CreationPage() {
  const { kind, id } = useLocalSearchParams<{ kind: string; id: string }>();
  const wsId = useWorkspaceStore(s => s.currentWorkspaceId);
  const slug = useWorkspaceStore(s => s.currentWorkspaceSlug);
  const builds = useQuery(brickCreationsOptions(wsId, kind === "build"));
  const circuits = useQuery(circuitCreationsOptions(wsId, kind === "circuit"));
  const query = kind === "build" ? builds : circuits;
  const rows = kind === "build" ? creationSummaries(builds.data?.creations) : creationSummaries([], circuits.data?.creations);
  const item = rows.find(v => v.id === id);
  const url = kind === "build" || kind === "circuit" ? creationWebURL(process.env.EXPO_PUBLIC_WEB_URL, slug ?? "", kind, id) : null;
  return <SafeAreaView className="flex-1 bg-background" edges={["bottom"]}>
    {query.isLoading ? <ActivityIndicator /> : query.isError ? <View className="p-4 gap-3"><Text className="text-destructive">Could not load this creation.</Text><Button variant="outline" onPress={() => void query.refetch()}><Text>Retry</Text></Button></View> : !item ? <Text className="p-4 text-muted-foreground">This creation is outside the recent list or is no longer available.</Text> :
      <ScrollView contentContainerStyle={{ padding: 20, gap: 16 }}>
        <Text className="text-xs uppercase text-muted-foreground">{kind === "build" ? "Brick creation" : "Circuit creation"}</Text>
        <Text className="text-2xl font-semibold">{item.title}</Text>
        <Text className="text-muted-foreground">{item.description}</Text>
        <View className="rounded-lg bg-muted p-4"><Text>{item.progress}</Text></View>
        <Text className="text-sm text-muted-foreground">View the complete model and continue building on the web. Your browser may ask you to sign in with the same account.</Text>
        <Button disabled={!url} onPress={() => { if (url) void Linking.openURL(url).catch(() => Alert.alert("Could not open creation", "Please try again.")); }}><Text>Open on the web</Text></Button>
        {!url && <Text className="text-sm text-muted-foreground">Web access is not configured for this app environment.</Text>}
      </ScrollView>}
  </SafeAreaView>;
}
