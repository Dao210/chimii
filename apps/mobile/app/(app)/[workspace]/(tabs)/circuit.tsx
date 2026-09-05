import { useRef, useState } from "react";
import { View } from "react-native";
import { router } from "expo-router";
import { useQuery } from "@tanstack/react-query";
import { Text } from "@/components/ui/text";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/text-field";
import {
  MakerPage,
  MakerScroll,
  Panel,
  RowLink,
  Choice,
  Notice,
  Loading,
  useMaker,
} from "@/components/maker/shared";
import {
  circuitCatalogOptions,
  circuitInventoryOptions,
  kitsOptions,
  useCreateCircuit,
} from "@/data/queries/maker";
import { circuitMaterials } from "@chimii/core/circuit/geometry";
import { createRequestId } from "@/lib/request-id";
export default function Circuit() {
  const { ws, slug, draft, patch, focused } = useMaker();
  const [selected, setSelected] = useState("");
  const kits = useQuery(kitsOptions(ws, focused));
  const kit = draft.kit || kits.data?.kits[0]?.kit_id || "";
  const catalog = useQuery(circuitCatalogOptions(ws, kit, focused));
  const inventory = useQuery(circuitInventoryOptions(ws, kit, focused));
  const active = useRef(focused);
  active.current = focused;
  const create = useCreateCircuit(ws);
  const data = catalog.data?.catalog;
  const project = data?.projects.find((v) => v.id === selected);
  const ready =
    !!inventory.data?.confirmed &&
    inventory.data.catalog_version === data?.version;
  const materials =
    project && inventory.data
      ? circuitMaterials(project, inventory.data.quantities)
      : [];
  const createProject = async (projectId?: string) => {
    if (!ready || !data || !inventory.data) return;
    const input = {
      kit_id: kit,
      catalog_version: data.version,
      project_id: projectId,
      prompt: projectId ? undefined : draft.circuitPrompt?.trim(),
      locale: "zh",
      inventory: inventory.data.quantities,
      inventory_revision: inventory.data.revision,
    };
    const signature = JSON.stringify(input);
    const id =
      draft.circuitRequest?.signature === signature
        ? draft.circuitRequest.id
        : createRequestId();
    patch({ circuitRequest: { signature, id } });
    try {
      const value = await create.mutateAsync({
        ...input,
        client_request_id: id,
      });
      patch({ circuitCreationId: value.id });
      if (active.current) router.push(`/${slug}/creation/circuit/${value.id}`);
    } catch {
      /* Retry reuses idempotency ID. */
    }
  };
  return (
    <MakerPage title="Circuit" subtitle="接起来，看看会发生什么">
      <MakerScroll>
        <View className="flex-row flex-wrap gap-2">
          {kits.data?.kits.map((v) => (
            <Choice
              key={v.kit_id}
              label={v.name}
              selected={kit === v.kit_id}
              disabled={create.isPending}
              onPress={() => {
                patch({ kit: v.kit_id });
                setSelected("");
                create.reset();
              }}
            />
          ))}
        </View>
        <Notice
          error={kits.error || catalog.error || inventory.error}
          retry={() => {
            void kits.refetch();
            if (kit) {
              void catalog.refetch();
              void inventory.refetch();
            }
          }}
        />
        {(kits.isLoading || catalog.isLoading) && <Loading />}
        {data && (
          <>
            <RowLink
              label={ready ? "我的电路套件 · 已确认" : "先确认你手边的套件"}
              detail="核对零件数量，保证每一步都有材料"
              icon="construct-outline"
              onPress={() =>
                router.push(
                  `/${slug}/circuit-inventory?kit=${encodeURIComponent(kit)}`,
                )
              }
            />
            {project ? (
              <Panel>
                <Button variant="ghost" onPress={() => setSelected("")}>
                  <Text>返回项目选择</Text>
                </Button>
                <Text className="text-xl font-bold">{project.title.zh}</Text>
                <Text>{project.description.zh}</Text>
                <Text className="font-semibold">准备这些零件</Text>
                {materials.map((v) => (
                  <Text
                    key={v.partId}
                    className={
                      v.missing ? "text-destructive" : "text-muted-foreground"
                    }
                  >
                    {data.parts.find((p) => p.id === v.partId)?.name.zh ||
                      v.partId}{" "}
                    × {v.required}
                    {v.missing ? `（还缺 ${v.missing} 个）` : ""}
                  </Text>
                ))}
                <Button
                  size="lg"
                  disabled={
                    !ready ||
                    create.isPending ||
                    materials.some((v) => v.missing > 0)
                  }
                  onPress={() => void createProject(project.id)}
                >
                  <Text>{create.isPending ? "正在准备…" : "开始连接"}</Text>
                </Button>
              </Panel>
            ) : (
              <>
                <Text className="text-xl font-bold">从一个小实验开始</Text>
                {data.projects.map((v) => (
                  <RowLink
                    key={v.id}
                    label={v.title.zh}
                    detail={`${v.description.zh} · ${v.steps.length} 步`}
                    icon="flash-outline"
                    onPress={() => {
                      setSelected(v.id);
                      create.reset();
                    }}
                  />
                ))}
                <Panel>
                  <Text className="text-xl font-bold">试试自己的想法</Text>
                  <TextField
                    accessibilityLabel="电路想法"
                    multiline
                    style={{
                      height: 88,
                      textAlignVertical: "top",
                      paddingTop: 12,
                    }}
                    maxLength={280}
                    placeholder="我想让灯亮起来…"
                    value={draft.circuitPrompt || ""}
                    onChangeText={(circuitPrompt) => patch({ circuitPrompt })}
                    editable={!create.isPending}
                  />
                  <Button
                    size="lg"
                    disabled={
                      !ready ||
                      !catalog.data?.ai_available ||
                      !draft.circuitPrompt?.trim() ||
                      create.isPending
                    }
                    onPress={() => void createProject()}
                  >
                    <Text>
                      {create.isPending ? "正在生成…" : "生成连接方案"}
                    </Text>
                  </Button>
                  {!catalog.data?.ai_available && (
                    <Text className="text-sm text-muted-foreground">
                      自由生成暂未开启，可以先体验上面的项目。
                    </Text>
                  )}
                </Panel>
              </>
            )}
            <Notice error={create.error} />
            {draft.circuitCreationId && (
              <RowLink
                label="继续刚才的电路作品"
                onPress={() =>
                  router.push(
                    `/${slug}/creation/circuit/${draft.circuitCreationId}`,
                  )
                }
              />
            )}
            <Text className="text-sm text-muted-foreground">
              适合 {data.minimum_age}{" "}
              岁及以上，和家长一起完成。连接前断开电源，完成后再按测试步骤尝试。
            </Text>
          </>
        )}
      </MakerScroll>
    </MakerPage>
  );
}
