import { useState } from "react";
import { FlatList, RefreshControl, View } from "react-native";
import { router } from "expo-router";
import { useQuery } from "@tanstack/react-query";
import { Text } from "@/components/ui/text";
import {
  MakerPage,
  Choice,
  Loading,
  Notice,
  RowLink,
  useMaker,
} from "@/components/maker/shared";
import {
  brickCreationsOptions,
  circuitCreationsOptions,
} from "@/data/queries/creations";
export default function Creations() {
  const { ws, slug, focused } = useMaker();
  const [filter, setFilter] = useState("all");
  const builds = useQuery(brickCreationsOptions(ws, focused));
  const circuits = useQuery(circuitCreationsOptions(ws, focused));
  const rows = [
    ...(builds.data?.creations ?? []).map((v) => ({
      id: v.id,
      kind: "build",
      title: v.title,
      date: v.created_at,
      completed: !!v.progress.completed_at,
      detail: `${v.part_count} 块积木 · ${v.step_count} 步 · ${v.progress.completed_at ? "已完成搭建" : v.progress.current_step ? `已到第 ${v.progress.current_step} 步` : "准备开始"}`,
    })),
    ...(circuits.data?.creations ?? []).map((v) => ({
      id: v.id,
      kind: "circuit",
      title: v.title,
      date: v.created_at,
      completed: v.observation === "worked",
      detail: `电路 · 第 ${v.current_step + 1} 步 · ${v.observation === "worked" ? "反馈：成功运行" : v.observation === "needs_help" ? "反馈：需要帮助" : v.observation === "not_tried" ? "尚未测试" : "未知反馈"}`,
    })),
  ]
    .filter(
      (v) =>
        filter === "all" ||
        filter === v.kind ||
        (filter === "ongoing" && !v.completed),
    )
    .sort((a, b) => b.date.localeCompare(a.date));
  const refresh = () => {
    void builds.refetch();
    void circuits.refetch();
  };
  return (
    <MakerPage title="Creations" subtitle="每一个想法，都留下作品">
      <View className="px-5 pb-4 flex-row flex-wrap gap-2">
        {[
          ["all", "全部"],
          ["ongoing", "进行中"],
          ["build", "积木"],
          ["circuit", "电路"],
        ].map(([key, label]) => (
          <Choice
            key={key}
            label={label!}
            selected={filter === key}
            onPress={() => setFilter(key!)}
          />
        ))}
      </View>
      <FlatList
        data={rows}
        keyExtractor={(v) => `${v.kind}:${v.id}`}
        contentContainerStyle={{
          paddingHorizontal: 20,
          paddingBottom: 24,
          gap: 12,
        }}
        refreshControl={
          <RefreshControl
            refreshing={builds.isRefetching || circuits.isRefetching}
            onRefresh={refresh}
          />
        }
        ListHeaderComponent={
          <Notice error={builds.error || circuits.error} retry={refresh} />
        }
        ListEmptyComponent={
          builds.isLoading || circuits.isLoading ? (
            <Loading />
          ) : (
            <View className="py-8 gap-4">
              <Text className="text-muted-foreground">
                这里还没有匹配的作品，从一个小想法开始吧。
              </Text>
              <RowLink
                label="去 Build 创作"
                onPress={() => router.navigate(`/${slug}/build`)}
              />
            </View>
          )
        }
        ListFooterComponent={
          <Text className="text-xs text-muted-foreground mt-4">
            显示最近 60 件积木作品和 50 件电路作品。
          </Text>
        }
        renderItem={({ item }) => (
          <RowLink
            label={item.title}
            detail={item.detail}
            icon={item.kind === "build" ? "cube-outline" : "flash-outline"}
            onPress={() =>
              router.push(`/${slug}/creation/${item.kind}/${item.id}`)
            }
          />
        )}
      />
    </MakerPage>
  );
}
