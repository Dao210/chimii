import { useEffect, useRef, useState } from "react";
import { AppState, View } from "react-native";
import { WebView } from "react-native-webview";
import { useQuery } from "@tanstack/react-query";
import type { BuildPlan } from "@chimii/core/build/types";
import { api } from "@/data/api";
import { makerKeys } from "@/data/queries/maker";
import { Text } from "@/components/ui/text";
import { Button } from "@/components/ui/button";
import { Loading, Notice, useMaker } from "./shared";
import { LDRAW_CATALOG, LDRAW_CATALOG_VERSION } from "@/lib/maker-catalog.generated";
import rendererHTML from "@/lib/maker-renderer.generated";
export default function BuildViewer({ plan, step, highlight = [] }: { plan: BuildPlan; step?: number; highlight?: string[] }) {
 const { ws, focused } = useMaker(); const [foreground, setForeground] = useState(AppState.currentState === "active");
 useEffect(() => { const sub = AppState.addEventListener("change", value => setForeground(value === "active")); return () => sub.remove(); }, []);
 const assets = useQuery({ queryKey: makerKeys.item(ws, "model-assets", plan.content_hash), queryFn: async ({ signal }) => {
   const ids = [...new Set(plan.placements.map(p => plan.parts[p.part_id]!.ldraw_id))]; const result: Record<string, string> = {};
   // Four bounded requests, rather than one per placement. Cancellation covers body reads.
   let cursor = 0; await Promise.all(Array.from({ length: Math.min(4, ids.length) }, async () => { while (cursor < ids.length) { const id = ids[cursor++]!; const bundled = plan.catalog_version === LDRAW_CATALOG_VERSION ? LDRAW_CATALOG[id.toLowerCase()] : undefined; result[id] = bundled?.glbBase64 ?? await api.buildPartAsset(plan.catalog_version, id, { signal }); } }));
   return result;
 }, enabled: focused && foreground && !!ws, staleTime: Infinity, gcTime: 60_000, retry: 1 });
 if (!focused || !foreground) return <View style={{ height: 310 }} />;
 if (assets.isError) return <Notice error={assets.error} retry={() => void assets.refetch()} />;
 if (!assets.data) return <Loading />;
 return <ActiveViewer plan={plan} step={step} highlight={highlight} assets={assets.data} />;
}
function ActiveViewer({ plan, step, highlight, assets }: { plan: BuildPlan; step?: number; highlight: string[]; assets: Record<string, string> }) {
 const web = useRef<WebView>(null); const [ready, setReady] = useState(false); const [loaded, setLoaded] = useState(false); const [failed, setFailed] = useState(false); const [attempt, setAttempt] = useState(0);
 const current = useRef({ step, highlight }); current.current = { step, highlight };
 useEffect(() => { if (ready) web.current?.postMessage(JSON.stringify({ type: "model", plan, assets, ...current.current })); }, [ready, plan, assets]);
 useEffect(() => { if (loaded) web.current?.postMessage(JSON.stringify({ type: "step", step, highlight })); }, [step, highlight, loaded]);
 useEffect(() => { if (loaded || failed) return; const timeout = setTimeout(() => setFailed(true), 20000); return () => clearTimeout(timeout); }, [loaded, failed, attempt]);
 const reload = () => { setReady(false); setLoaded(false); setFailed(false); setAttempt(v => v + 1); };
 return <View className="rounded-xl bg-secondary overflow-hidden gap-2">
  {failed ? <View className="p-4 gap-3"><Text>模型暂时没有加载完成，材料和步骤仍可查看。</Text><Button variant="outline" onPress={reload}><Text>重新加载模型</Text></Button></View> : <View style={{ height: 310 }}><WebView key={attempt} ref={web} source={{ html: rendererHTML }} originWhitelist={["about:blank"]} onShouldStartLoadWithRequest={request => request.url === "about:blank" || request.url.startsWith("data:text/html")}
    onMessage={event => { try { const message = JSON.parse(event.nativeEvent.data); if (message.type === "ready") setReady(true); if (message.type === "loaded") setLoaded(true); if (message.type === "error") setFailed(true); } catch { setFailed(true); } }}
    onError={() => setFailed(true)} onContentProcessDidTerminate={reload} onRenderProcessGone={reload} javaScriptEnabled domStorageEnabled={false} sharedCookiesEnabled={false} thirdPartyCookiesEnabled={false} allowFileAccess={false} setSupportMultipleWindows={false} scrollEnabled={false} bounces={false} mixedContentMode="never" style={{ backgroundColor: "transparent" }} />{!loaded && <View pointerEvents="none" className="absolute inset-0"><Loading /></View>}</View>}
  <View className="flex-row justify-center gap-2"><Button variant="outline" size="lg" accessibilityLabel="缩小模型" disabled={!loaded} onPress={() => web.current?.postMessage(JSON.stringify({ type: "zoom", factor: 1.2 }))}><Text>−</Text></Button><Button variant="outline" size="lg" disabled={!loaded} onPress={() => web.current?.postMessage(JSON.stringify({ type: "reset" }))}><Text>复位视角</Text></Button><Button variant="outline" size="lg" accessibilityLabel="放大模型" disabled={!loaded} onPress={() => web.current?.postMessage(JSON.stringify({ type: "zoom", factor: 0.8 }))}><Text>＋</Text></Button></View>
  <Text className="text-xs text-center text-muted-foreground pb-3">拖动旋转 · 双指缩放 · 发亮的是本步新增</Text>
 </View>;
}
