import { useEffect, useRef, useState } from "react";
import { AppState, View } from "react-native";
import { WebView } from "react-native-webview";
import { useQuery } from "@tanstack/react-query";
import type { BuildPlan } from "@chimii/core/build/types";
import { modelAssetsOptions } from "@/data/queries/model-assets";
import { Text } from "@/components/ui/text";
import { Button } from "@/components/ui/button";
import { Loading, Notice, useMaker } from "./shared";
import rendererHTML from "@/lib/maker-renderer.generated";
export default function BuildViewer({
  plan,
  step,
  highlight = [],
}: {
  plan: BuildPlan;
  step?: number;
  highlight?: string[];
}) {
  const { focused } = useMaker();
  const [foreground, setForeground] = useState(
    AppState.currentState === "active",
  );
  useEffect(() => {
    const sub = AppState.addEventListener("change", (value) =>
      setForeground(value === "active"),
    );
    return () => sub.remove();
  }, []);
  if (!focused || !foreground) return <View style={{ height: 310 }} />;
  return <ModelAssets plan={plan} step={step} highlight={highlight} />;
}
function ModelAssets({
  plan,
  step,
  highlight,
}: {
  plan: BuildPlan;
  step?: number;
  highlight: string[];
}) {
  const { ws } = useMaker();
  const assets = useQuery(modelAssetsOptions(ws, plan));
  if (assets.isError)
    return <Notice error={assets.error} retry={() => void assets.refetch()} />;
  if (!assets.data) return <Loading />;
  return (
    <ActiveViewer
      key={plan.content_hash}
      plan={plan}
      step={step}
      highlight={highlight}
      assets={assets.data}
    />
  );
}
function ActiveViewer({
  plan,
  step,
  highlight,
  assets,
}: {
  plan: BuildPlan;
  step?: number;
  highlight: string[];
  assets: Record<string, string>;
}) {
  const web = useRef<WebView>(null);
  const [ready, setReady] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const current = useRef({ step, highlight });
  current.current = { step, highlight };
  useEffect(() => {
    if (ready)
      web.current?.postMessage(
        JSON.stringify({ type: "model", plan, assets, ...current.current }),
      );
  }, [ready, plan, assets]);
  useEffect(() => {
    if (loaded)
      web.current?.postMessage(
        JSON.stringify({ type: "step", step, highlight }),
      );
  }, [step, highlight, loaded]);
  useEffect(() => {
    if (loaded || failed) return;
    const timeout = setTimeout(() => setFailed(true), 20000);
    return () => clearTimeout(timeout);
  }, [loaded, failed, attempt]);
  const reload = () => {
    setReady(false);
    setLoaded(false);
    setFailed(false);
    setAttempt((v) => v + 1);
  };
  return (
    <View className="rounded-xl bg-secondary overflow-hidden gap-2">
      {failed ? (
        <View className="p-4 gap-3">
          <Text>模型暂时没有加载完成，材料和步骤仍可查看。</Text>
          <Button variant="outline" onPress={reload}>
            <Text>重新加载模型</Text>
          </Button>
        </View>
      ) : (
        <View style={{ height: 310 }}>
          <WebView
            key={attempt}
            ref={web}
            source={{ html: rendererHTML }}
            originWhitelist={["about:blank"]}
            onShouldStartLoadWithRequest={(request) =>
              request.url === "about:blank" ||
              request.url.startsWith("data:text/html")
            }
            onMessage={(event) => {
              try {
                const message = JSON.parse(event.nativeEvent.data);
                if (message.type === "ready") setReady(true);
                if (message.type === "loaded") setLoaded(true);
                if (message.type === "error") setFailed(true);
              } catch {
                setFailed(true);
              }
            }}
            onError={() => setFailed(true)}
            onContentProcessDidTerminate={reload}
            onRenderProcessGone={reload}
            javaScriptEnabled
            domStorageEnabled={false}
            sharedCookiesEnabled={false}
            thirdPartyCookiesEnabled={false}
            allowFileAccess={false}
            setSupportMultipleWindows={false}
            scrollEnabled={false}
            bounces={false}
            mixedContentMode="never"
            style={{ backgroundColor: "transparent" }}
          />
          {!loaded && (
            <View pointerEvents="none" className="absolute inset-0">
              <Loading />
            </View>
          )}
        </View>
      )}
      <View className="flex-row justify-center gap-2">
        <Button
          variant="outline"
          size="lg"
          accessibilityLabel="缩小模型"
          disabled={!loaded}
          onPress={() =>
            web.current?.postMessage(
              JSON.stringify({ type: "zoom", factor: 1.2 }),
            )
          }
        >
          <Text>−</Text>
        </Button>
        <Button
          variant="outline"
          size="lg"
          disabled={!loaded}
          onPress={() =>
            web.current?.postMessage(JSON.stringify({ type: "reset" }))
          }
        >
          <Text>复位视角</Text>
        </Button>
        <Button
          variant="outline"
          size="lg"
          accessibilityLabel="放大模型"
          disabled={!loaded}
          onPress={() =>
            web.current?.postMessage(
              JSON.stringify({ type: "zoom", factor: 0.8 }),
            )
          }
        >
          <Text>＋</Text>
        </Button>
      </View>
      <Text className="text-xs text-center text-muted-foreground pb-3">
        拖动旋转 · 双指缩放 · 发亮的是本步新增
      </Text>
    </View>
  );
}
