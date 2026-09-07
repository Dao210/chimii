import { useEffect, useMemo, useState } from "react";
import { Alert, FlatList, Pressable, View } from "react-native";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { Text } from "@/components/ui/text";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/text-field";
import {
  Choice,
  Loading,
  MakerPage,
  Notice,
  Quantity,
  useMaker,
} from "@/components/maker/shared";
import {
  catalogOptions,
  inventoryOptions,
  partsOptions,
  useSaveInventory,
} from "@/data/queries/maker";
import { PartThumbnail } from "@/components/maker/part-thumbnail";
import {
  setBrickQuantity,
  brickColorName,
  sameBrickQuantities,
} from "@/lib/maker";
export default function Block() {
  const { ws, draft, patch, focused } = useMaker();
  const [mode, setMode] = useState("mine");
  const [search, setSearch] = useState("");
  const [debounced, setDebounced] = useState("");
  const [expanded, setExpanded] = useState<string | null>(null);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(search.trim()), 300);
    return () => clearTimeout(timer);
  }, [search]);
  const inventory = useQuery(inventoryOptions(ws, focused));
  const catalog = useQuery(catalogOptions(ws, focused));
  const parts = useInfiniteQuery(
    partsOptions(ws, debounced, focused && mode === "catalog"),
  );
  const save = useSaveInventory(ws);
  // A response can be lost after the server commits and before local cleanup.
  // Only identical quantities at a newer revision prove this draft was applied.
  useEffect(() => {
    if (
      draft.bricks &&
      inventory.data &&
      inventory.data.revision > draft.bricks.revision &&
      sameBrickQuantities(draft.bricks.items, inventory.data.items)
    )
      patch({ bricks: undefined });
  }, [draft.bricks, inventory.data, patch]);
  const items = draft.bricks?.items ?? inventory.data?.items ?? [];
  const partSpecs = useMemo(
    () =>
      new Map(
        [
          ...(catalog.data?.parts ?? []),
          ...(parts.data?.pages.flatMap((p) => p.parts) ?? []),
        ].map((p) => [p.id, p]),
      ),
    [catalog.data, parts.data],
  );
  const change = (part: string, color: number, quantity: number) => {
    if (!inventory.data) return;
    patch({
      bricks: {
        revision: draft.bricks?.revision ?? inventory.data.revision,
        items: setBrickQuantity(items, part, color, quantity),
      },
    });
  };
  const store = async () => {
    if (!inventory.data) return;
    try {
      await save.mutateAsync({
        expected_revision: draft.bricks?.revision ?? inventory.data.revision,
        items,
      });
      patch({ bricks: undefined });
    } catch {
      /* Keep complete local draft, including offscreen parts. */
    }
  };
  const reload = () =>
    Alert.alert("读取最新库存？", "这会丢弃本机尚未保存的数量修改。", [
      { text: "保留修改", style: "cancel" },
      {
        text: "读取最新",
        onPress: () => {
          patch({ bricks: undefined });
          save.reset();
          void inventory.refetch();
        },
      },
    ]);
  const rows: {
    id: string;
    part: string;
    color: number | null;
    quantity: number;
  }[] =
    mode === "mine"
      ? items
          .filter((v) =>
            `${partSpecs.get(v.part_id)?.name || ""} ${v.part_id}`
              .toLowerCase()
              .includes(debounced.toLowerCase()),
          )
          .map((v) => ({
            id: `${v.part_id}:${v.color}`,
            part: v.part_id,
            color: v.color,
            quantity: v.quantity,
          }))
      : [
          ...new Map(
            (parts.data?.pages.flatMap((v) => v.parts) ?? []).map((v) => [
              v.id,
              { id: v.id, part: v.id, color: null, quantity: 0 },
            ]),
          ).values(),
        ];
  const colors = catalog.data?.colors ?? [];
  return (
    <MakerPage title="Block" subtitle="先数清积木，再放开想象">
      <View className="px-5 gap-3 pb-3">
        <View className="flex-row gap-2">
          <Choice
            label="我的积木"
            selected={mode === "mine"}
            onPress={() => setMode("mine")}
          />
          <Choice
            label="零件目录"
            selected={mode === "catalog"}
            onPress={() => setMode("catalog")}
          />
        </View>
        <Text className="text-sm text-muted-foreground">
          {items.reduce((sum, v) => sum + v.quantity, 0)} 块 · {items.length}{" "}
          种零件与颜色
          {draft.bricks
            ? " · 有未保存修改"
            : inventory.data?.configured
              ? " · 已保存"
              : " · 尚未记录"}
        </Text>
        <TextField
          accessibilityLabel="搜索积木"
          placeholder="搜索名称或零件编号"
          value={search}
          onChangeText={setSearch}
        />
      </View>
      {inventory.isLoading || catalog.isLoading ? (
        <Loading />
      ) : (
        <FlatList
          data={rows}
          keyExtractor={(v) => v.id}
          keyboardShouldPersistTaps="handled"
          contentContainerStyle={{ paddingHorizontal: 20, paddingBottom: 16 }}
          ListHeaderComponent={
            <>
              <Notice
                error={inventory.error || catalog.error || parts.error}
                retry={() => {
                  void inventory.refetch();
                  void catalog.refetch();
                  if (mode === "catalog") void parts.refetch();
                }}
              />
              <Notice error={save.error} />
              {save.isError && draft.bricks && (
                <Button variant="outline" onPress={reload}>
                  <Text>重新读取库存</Text>
                </Button>
              )}
            </>
          }
          ListEmptyComponent={
            parts.isLoading && mode === "catalog" ? (
              <Loading />
            ) : (
              <View className="py-8 gap-4">
                <Text className="text-muted-foreground">
                  {mode === "mine"
                    ? "还没有匹配的积木。到零件目录选择颜色和数量。"
                    : "没有找到匹配的零件。"}
                </Text>
                {mode === "mine" && (
                  <Button
                    variant="outline"
                    size="lg"
                    onPress={() => {
                      setSearch("");
                      setMode("catalog");
                    }}
                  >
                    <Text>添加积木</Text>
                  </Button>
                )}
              </View>
            )
          }
          onEndReached={() => {
            if (
              mode === "catalog" &&
              parts.hasNextPage &&
              !parts.isFetchingNextPage
            )
              void parts.fetchNextPage();
          }}
          onEndReachedThreshold={0.4}
          ListFooterComponent={parts.isFetchingNextPage ? <Loading /> : null}
          renderItem={({ item }) => (
            <View className="py-4 border-b border-border gap-3">
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={`查看 ${partSpecs.get(item.part)?.name || item.part}`}
                onPress={() =>
                  setExpanded(expanded === item.id ? null : item.id)
                }
                className="flex-row items-center gap-3"
              >
                <PartThumbnail
                  version={catalog.data?.catalog_version || ""}
                  ldraw={partSpecs.get(item.part)?.ldraw_id || ""}
                  color={item.color ?? 14}
                />
                <View className="flex-1">
                  <Text className="font-semibold">
                    {partSpecs.get(item.part)?.name || item.part}
                  </Text>
                  <Text className="text-sm text-muted-foreground">
                    {item.part}
                    {item.color !== null
                      ? ` · ${brickColorName(item.color)}`
                      : " · 选择颜色与数量"}
                  </Text>
                </View>
                <Text>{expanded === item.id ? "收起" : "编辑"}</Text>
              </Pressable>
              {item.color !== null && (
                <Quantity
                  name={`${item.part} ${item.color}`}
                  value={item.quantity}
                  disabled={save.isPending}
                  onChange={(n) => change(item.part, item.color!, n)}
                />
              )}
              {expanded === item.id &&
                colors.map((color) => (
                  <View
                    key={color.code}
                    className="flex-row items-center justify-between gap-2"
                  >
                    <View className="flex-row items-center gap-2 flex-1">
                      <View
                        style={{ backgroundColor: color.hex }}
                        className="size-5 rounded-full border border-border"
                      />
                      <Text className="text-sm flex-1">
                        {brickColorName(color.code)}
                      </Text>
                    </View>
                    <Quantity
                      name={`${item.part} ${color.name}`}
                      value={
                        items.find(
                          (v) =>
                            v.part_id === item.part && v.color === color.code,
                        )?.quantity ?? 0
                      }
                      disabled={save.isPending}
                      onChange={(n) => change(item.part, color.code, n)}
                    />
                  </View>
                ))}
            </View>
          )}
        />
      )}
      <View className="border-t border-border p-4 gap-2">
        <Button
          size="lg"
          disabled={save.isPending || !inventory.data || !draft.bricks}
          onPress={() => void store()}
        >
          <Text>{save.isPending ? "正在保存…" : "保存我的积木"}</Text>
        </Button>
        <Text className="text-xs text-muted-foreground text-center">
          保存后，Build 才会使用这些数量。
        </Text>
      </View>
    </MakerPage>
  );
}
