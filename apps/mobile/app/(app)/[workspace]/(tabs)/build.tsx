import { useEffect, useState } from "react";
import { View } from "react-native";
import { router } from "expo-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Text } from "@/components/ui/text";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/text-field";
import { MakerPage, MakerScroll, Panel, RowLink, Notice, Loading, Choice, useMaker } from "@/components/maker/shared";
import { buildSessionOptions, inventoryOptions, makerConfigOptions, useBuildActions } from "@/data/queries/maker";
import { brickCreationsOptions, creationKeys } from "@/data/queries/creations";
import { createRequestId } from "@/lib/request-id";
export default function Build() {
 const { ws, slug, draft, patch, focused } = useMaker();
 const qc = useQueryClient();
 const [free, setFree] = useState("");
 const inventory = useQuery(inventoryOptions(ws, focused));
 const config = useQuery(makerConfigOptions(ws));
 const recent = useQuery(brickCreationsOptions(ws, focused));
 const session = useQuery(buildSessionOptions(ws, draft.sessionId || "", focused));
 const actions = useBuildActions(ws, draft.sessionId);
 const current = session.data; const question = current?.question;
 const busy = actions.session.isPending || actions.answer.isPending || actions.cancel.isPending;
 const active = !!draft.sessionId && (!current || ["queued", "generating", "clarifying"].includes(current.status));
 useEffect(() => { if (current?.status === "completed") void qc.invalidateQueries({ queryKey: creationKeys.list(ws, "build") }); }, [current?.status, qc, ws]);
 const start = async () => {
   const id = draft.buildRequest || createRequestId(); patch({ buildRequest: id });
   try { const value = await actions.session.mutateAsync({ prompt: draft.prompt!.trim(), client_request_id: id }); patch({ sessionId: value.id }); } catch { /* Notice preserves input and retry ID. */ }
 };
 const answer = async (value: string) => {
   if (!question || current?.revision === undefined) return;
   try { await actions.answer.mutateAsync({ answers: { [question.id]: value }, revision: current.revision }); setFree(""); } catch { /* Notice */ }
 };
 const reset = async () => {
   if (active && current?.revision !== undefined) { try { await actions.cancel.mutateAsync(current.revision); } catch { return; } }
   patch({ sessionId: undefined, buildRequest: undefined }); actions.session.reset(); actions.answer.reset(); setFree("");
 };
 const resume = recent.data?.creations.find(v => !v.progress.completed_at && v.progress.current_step > 0);
 return <MakerPage title="Build" subtitle="把一个想法，搭成真的"><MakerScroll>
   {active ? <Panel><Text className="text-xl font-bold">{current?.status === "clarifying" ? "先确定一个小细节" : "正在准备你的搭建方案"}</Text><Text className="text-muted-foreground">{draft.prompt}</Text>
    {session.isLoading && <Loading />}
    {question && current?.status === "clarifying" ? <><Text className="text-lg">{question.prompt}</Text>{(question.choices?.length ? question.choices : question.options.map(v => ({ id: v, label: v }))).map(v => <Choice key={v.id} label={v.label} disabled={busy || session.isFetching} onPress={() => void answer(v.id)} />)}
     {question.allow_free_text && <><TextField value={free} onChangeText={setFree} placeholder="说说你的选择" maxLength={280} /><Button size="lg" disabled={busy || !free.trim()} onPress={() => void answer(free.trim())}><Text>确认这个想法</Text></Button></>}
    </> : current && <><Loading /><Text className="text-muted-foreground">正在检查零件和结构。可以先看看其他页面，稍后回来继续。</Text></>}
    <Notice error={session.error} retry={() => void session.refetch()} />
    <Button variant="ghost" size="lg" disabled={busy || current?.revision === undefined} onPress={() => void reset()}><Text>修改想法</Text></Button>
   </Panel> : <Panel><Text className="text-2xl font-bold">今天想搭什么？</Text><Text className="text-muted-foreground">描述外形或用途，我们一起确定材料和步骤。</Text>
    <TextField accessibilityLabel="搭建想法" multiline maxLength={280} value={draft.prompt || ""} onChangeText={prompt => patch({ prompt, buildRequest: undefined, sessionId: undefined })} placeholder="一辆能推着走的小汽车…" style={{ height: 112, textAlignVertical: "top", paddingTop: 12 }} />
    <View className="flex-row flex-wrap gap-2">{["一辆小汽车", "一只小恐龙", "一个小机器人"].map(prompt => <Choice key={prompt} label={prompt} onPress={() => patch({ prompt, buildRequest: undefined, sessionId: undefined })} />)}</View>
    <Button size="lg" disabled={busy || !draft.prompt?.trim() || !config.data?.build_available} onPress={() => void start()}><Text>{busy ? "正在提交…" : "开始搭建"}</Text></Button>
    {config.data && !config.data.build_available && <Text className="text-muted-foreground">生成服务暂未开启，已有作品仍可继续搭建。</Text>}
    <Notice error={config.error} retry={() => void config.refetch()} />
   </Panel>}
   <Notice error={actions.session.error || actions.answer.error || actions.cancel.error} />
   {current?.status === "failed" && <Panel><Text>这次方案没有生成成功。</Text><Text className="text-muted-foreground">{current.message || "可以换一个简单些的想法，再试一次。"}</Text><Button variant="outline" onPress={() => void reset()}><Text>重新开始</Text></Button></Panel>}
   {current?.creation_id && <RowLink label="方案准备好了" detail="查看材料，开始一步步搭建" onPress={() => router.push(`/${slug}/creation/build/${current.creation_id}`)} />}
   <RowLink label="使用我的积木" detail={inventory.data?.configured ? `已记录 ${inventory.data.items.reduce((sum, v) => sum + v.quantity, 0)} 块，按实际库存生成` : "先到 Block 记录积木，让方案更适合你"} icon="cube-outline" onPress={() => router.navigate(`/${slug}/block`)} />
   {resume && <RowLink label={`继续：${resume.title}`} detail={`已保存到第 ${resume.progress.current_step} 步`} onPress={() => router.push(`/${slug}/creation/build/${resume.id}`)} />}
 </MakerScroll></MakerPage>;
}
