import { lazy, Suspense, useState } from "react";
import { Alert, ScrollView, Switch, View } from "react-native";
import { router } from "expo-router";
import { SafeAreaView } from "react-native-safe-area-context";
import { useQuery } from "@tanstack/react-query";
import { brickColorName } from "@/lib/maker";
import { Text } from "@/components/ui/text";
import { Button } from "@/components/ui/button";
import { PartThumbnail } from "./part-thumbnail";
import { CircuitBoard } from "./circuit-board";
import { Loading, Notice, Panel, MakerDetailHeader, useMaker } from "./shared";
import {
  buildDetailOptions,
  buildProgressOptions,
  circuitDetailOptions,
  useBuildProgress,
  useCircuitProgress,
} from "@/data/queries/maker";
const BuildViewer = lazy(() => import("./build-viewer"));
function Frame({
  title,
  children,
  footer,
}: {
  title: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
}) {
  return (
    <SafeAreaView edges={["bottom"]} className="flex-1 bg-background">
      <MakerDetailHeader title={title} />
      <ScrollView
        contentContainerStyle={{ padding: 20, gap: 16, paddingBottom: 24 }}
      >
        {children}
      </ScrollView>
      {footer && (
        <View className="px-4 pt-3 pb-2 border-t border-border gap-2">
          {footer}
        </View>
      )}
    </SafeAreaView>
  );
}
export function CreationDetail({
  id,
  kind,
  guide = false,
}: {
  id: string;
  kind: string;
  guide?: boolean;
}) {
  if (kind === "build") return <BuildDetail id={id} guide={guide} />;
  if (kind === "circuit") return <CircuitDetail id={id} guide={guide} />;
  return (
    <Frame title="作品">
      <Text>无法识别这个作品类型。</Text>
    </Frame>
  );
}
function BuildDetail({ id, guide }: { id: string; guide: boolean }) {
  const { ws, slug } = useMaker();
  const detail = useQuery(buildDetailOptions(ws, id));
  const progress = useQuery(buildProgressOptions(ws, id));
  const save = useBuildProgress(ws, id);
  const creation = detail.data;
  const saved = progress.data;
  const busy = save.isPending || progress.isFetching;
  const retry = () => {
    save.reset();
    void detail.refetch();
    void progress.refetch();
  };
  if (detail.isLoading)
    return (
      <Frame title="积木作品">
        <Loading />
      </Frame>
    );
  if (!creation)
    return (
      <Frame title="积木作品">
        <Notice error={detail.error} retry={retry} />
      </Frame>
    );
  const plan = creation.build_plan;
  const step = Math.max(1, saved?.current_step ?? 1);
  const total = plan.steps.length;
  const current = plan.steps.find((v) => v.number === step);
  const ids = new Set(current?.added_placement_ids);
  const added = plan.placements.filter((v) => ids.has(v.id));
  const materials = new Map<
    string,
    { name: string; color: number; count: number }
  >();
  for (const placement of plan.placements) {
    const key = `${placement.part_id}:${placement.color}`;
    const old = materials.get(key);
    materials.set(key, {
      name: plan.parts[placement.part_id]?.name || placement.part_id,
      color: placement.color,
      count: (old?.count ?? 0) + 1,
    });
  }
  const persist = async (next: number, completed = false) => {
    if (!saved || busy) return false;
    try {
      await save.mutateAsync({
        current_step: next,
        completed,
        expected_revision: saved.revision,
      });
      return true;
    } catch {
      return false;
    }
  };
  const start = async () => {
    if (!saved) return;
    if (saved.current_step === 0 && !(await persist(1))) return;
    router.push(`/${slug}/creation/build/${id}/guide`);
  };
  return (
    <Frame
      title={guide ? `第 ${step} / ${total} 步` : creation.title}
      footer={
        guide ? (
          <>
            <View className="flex-row gap-3">
              <Button
                className="flex-1"
                size="lg"
                variant="outline"
                disabled={busy || !saved || step <= 1}
                onPress={() => void persist(step - 1)}
              >
                <Text>上一步</Text>
              </Button>
              <Button
                className="flex-1"
                size="lg"
                disabled={busy || !saved || !!saved.completed_at}
                onPress={() =>
                  void persist(Math.min(total, step + 1), step === total)
                }
              >
                <Text>
                  {save.isPending
                    ? "正在保存…"
                    : step === total
                      ? "完成搭建"
                      : "搭好了，下一步"}
                </Text>
              </Button>
            </View>
            {saved?.completed_at && (
              <Button
                variant="outline"
                size="lg"
                onPress={() => router.navigate(`/${slug}/creations`)}
              >
                <Text>回到作品集</Text>
              </Button>
            )}
            <Text className="text-xs text-center text-muted-foreground">
              {saved?.completed_at
                ? "已保存：完成搭建"
                : saved?.current_step
                  ? `已保存到第 ${saved.current_step} 步`
                  : "尚未开始搭建"}
            </Text>
          </>
        ) : (
          <Button
            size="lg"
            disabled={!saved || busy}
            onPress={() => void start()}
          >
            <Text>
              {saved?.current_step ? "继续步骤引导" : "准备好了，开始搭建"}
            </Text>
          </Button>
        )
      }
    >
      {!guide && (
        <>
          <Text className="text-2xl font-bold">{creation.title}</Text>
          <Text className="text-muted-foreground">
            {plan.placements.length} 块积木 · {total} 步
          </Text>
        </>
      )}
      <Suspense fallback={<Loading />}>
        <BuildViewer
          plan={plan}
          step={guide ? step : undefined}
          highlight={guide ? current?.added_placement_ids : []}
        />
      </Suspense>
      <Notice
        error={detail.error || progress.error || save.error}
        retry={retry}
      />
      {guide ? (
        <Panel>
          <Text className="text-xl font-bold">拿出这些积木</Text>
          {added.map((v) => (
            <View key={v.id} className="flex-row items-center gap-3">
              <PartThumbnail
                version={plan.catalog_version}
                ldraw={plan.parts[v.part_id]?.ldraw_id || ""}
                color={v.color}
              />
              <Text className="flex-1">
                {plan.parts[v.part_id]?.name || v.part_id} ·{" "}
                {brickColorName(v.color)}
              </Text>
            </View>
          ))}
          <Text className="text-muted-foreground">
            把它们放到图中发亮的位置。对齐凸点，轻轻按紧。
          </Text>
          <Button
            variant="outline"
            size="lg"
            onPress={() =>
              Alert.alert(
                `第 ${step} 步的小提示`,
                `先旋转模型找到发亮的零件，核对外形与颜色，再检查是否对齐。\n\n${added.map((v) => `${plan.parts[v.part_id]?.name || v.part_id}：旋转 ${v.rotation}°`).join("\n")}\n\n连接前后都可拖动模型，查看遮挡的位置。`,
              )
            }
          >
            <Text>这一步需要帮助</Text>
          </Button>
        </Panel>
      ) : (
        <Panel>
          <Text className="text-xl font-bold">准备材料</Text>
          {[...materials].map(([key, value]) => (
            <Text key={key}>
              {value.name} · {brickColorName(value.color)} × {value.count}
            </Text>
          ))}
          <Text className="text-sm text-muted-foreground">
            {plan.inventory?.configured
              ? "这份方案按生成时保存的库存规划。"
              : "按材料清单核对手边零件后再开始。"}
          </Text>
        </Panel>
      )}
      <Text className="text-xs text-muted-foreground">
        {creation.validation.buildable
          ? "方案已通过结构规则检查。"
          : "请核对方案中的结构问题。"}{" "}
        实际稳固程度仍需在搭建时确认。
      </Text>
    </Frame>
  );
}
function CircuitDetail({ id, guide }: { id: string; guide: boolean }) {
  const { ws, slug } = useMaker();
  const detail = useQuery(circuitDetailOptions(ws, id));
  const save = useCircuitProgress(ws, id);
  const [checked, setChecked] = useState(false);
  const creation = detail.data;
  const busy = save.isPending || detail.isFetching;
  const retry = () => {
    save.reset();
    void detail.refetch();
  };
  if (detail.isLoading)
    return (
      <Frame title="电路作品">
        <Loading />
      </Frame>
    );
  if (!creation)
    return (
      <Frame title="电路作品">
        <Notice error={detail.error} retry={retry} />
      </Frame>
    );
  const doc = creation.document;
  const index = creation.current_step;
  const step = doc.project.steps[index]!;
  const last = index === doc.project.steps.length - 1;
  const persist = async (
    next: number,
    observation: "not_tried" | "worked" | "needs_help" = "not_tried",
  ) => {
    if (busy) return;
    try {
      await save.mutateAsync({
        current_step: next,
        observation,
        expected_revision: creation.progress_revision,
      });
      setChecked(false);
    } catch {
      /* Canonical step remains unchanged. */
    }
  };
  return (
    <Frame
      title={
        guide
          ? `第 ${index + 1} / ${doc.project.steps.length} 步`
          : creation.document.title
      }
      footer={
        guide ? (
          <>
            <View className="flex-row gap-3">
              <Button
                className="flex-1"
                size="lg"
                variant="outline"
                disabled={busy || index === 0}
                onPress={() => void persist(index - 1)}
              >
                <Text>上一步</Text>
              </Button>
              {!last ? (
                <Button
                  className="flex-1"
                  size="lg"
                  disabled={busy}
                  onPress={() => void persist(index + 1)}
                >
                  <Text>{save.isPending ? "正在保存…" : "接好了，下一步"}</Text>
                </Button>
              ) : (
                <Button
                  className="flex-1"
                  size="lg"
                  variant="outline"
                  disabled={busy}
                  onPress={() => router.navigate(`/${slug}/creations`)}
                >
                  <Text>回到作品集</Text>
                </Button>
              )}
            </View>
            <Text className="text-xs text-center text-muted-foreground">
              已保存到第 {index + 1} 步
            </Text>
          </>
        ) : (
          <Button
            size="lg"
            onPress={() => router.push(`/${slug}/creation/circuit/${id}/guide`)}
          >
            <Text>开始 / 继续连接</Text>
          </Button>
        )
      }
    >
      {!guide && (
        <>
          <Text className="text-2xl font-bold">{doc.title}</Text>
          <Text className="text-muted-foreground">
            {doc.project.description.zh}
          </Text>
        </>
      )}
      <CircuitBoard document={doc} step={guide ? index : undefined} />
      <Notice error={detail.error || save.error} retry={retry} />
      {guide ? (
        <Panel>
          <Text className="text-xl font-bold">{step.title.zh}</Text>
          <Text className="leading-7">{step.instruction.zh}</Text>
          <Button
            variant="outline"
            size="lg"
            onPress={() =>
              Alert.alert(
                "这一步的小提示",
                [
                  step.instruction.zh,
                  ...doc.project.troubleshooting.map((v) => v.zh),
                ].join("\n\n"),
              )
            }
          >
            <Text>这一步需要帮助</Text>
          </Button>
        </Panel>
      ) : (
        <>
          <Panel>
            <Text className="text-xl font-bold">开始前</Text>
            {doc.preparation.map((v, i) => (
              <Text key={i}>{v.zh}</Text>
            ))}
          </Panel>
          <Panel>
            <Text className="text-xl font-bold">准备材料</Text>
            {Object.entries(doc.validation.used_parts).map(([part, count]) => (
              <Text key={part}>
                {doc.parts.find((v) => v.id === part)?.name.zh || part} ×{" "}
                {count}
              </Text>
            ))}
          </Panel>
        </>
      )}
      {guide && last && (
        <Panel>
          <Text className="text-xl font-bold">试试看</Text>
          <Text>{doc.project.test_instruction.zh}</Text>
          <View className="flex-row gap-3 items-center">
            <Switch
              accessibilityLabel="家长已检查连接"
              value={checked}
              onValueChange={setChecked}
            />
            <Text className="flex-1">请家长检查连接后，再接通电源测试。</Text>
          </View>
          <Button
            size="lg"
            disabled={!checked || busy}
            onPress={() => void persist(index, "worked")}
          >
            <Text>成功运行了</Text>
          </Button>
          <Button
            size="lg"
            variant="outline"
            disabled={!checked || busy}
            onPress={() => void persist(index, "needs_help")}
          >
            <Text>还需要帮助</Text>
          </Button>
        </Panel>
      )}
      {creation.observation !== "not_tried" && (
        <Panel>
          <Text>
            {creation.observation === "worked"
              ? "已记录家庭反馈：成功运行"
              : creation.observation === "needs_help"
                ? "已记录家庭反馈：需要帮助"
                : "保存了其他测试反馈"}
          </Text>
          {creation.observation === "needs_help" &&
            doc.project.troubleshooting.map((v, i) => (
              <Text key={i}>{v.zh}</Text>
            ))}
        </Panel>
      )}
      <Text className="text-xs text-muted-foreground">
        {doc.validation.passed
          ? "连接方案通过规则检查。"
          : "连接方案尚未通过规则检查。"}{" "}
        实际测试结果由家庭反馈记录。
      </Text>
    </Frame>
  );
}
