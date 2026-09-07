import {
  useEffect,
  useCallback,
  useState,
  useSyncExternalStore,
  type PropsWithChildren,
} from "react";
import { onlineManager } from "@tanstack/react-query";
import {
  ActivityIndicator,
  Platform,
  Pressable,
  ScrollView,
  View,
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { Stack, router } from "expo-router";
import { Ionicons } from "@expo/vector-icons";
import { useIsFocused } from "@react-navigation/native";
import { Header } from "@/components/ui/header";
import { IconButton } from "@/components/ui/icon-button";
import { TextField } from "@/components/ui/text-field";
import { Text } from "@/components/ui/text";
import { Button } from "@/components/ui/button";
import { useWorkspaceStore } from "@/data/workspace-store";
import { useAuthStore } from "@/data/auth-store";
import {
  useMakerDraftStore,
  selectMakerDraft,
  restoreMakerDraft,
} from "@/data/stores/maker-draft-store";
import { makerError, makerScope, quantityInput } from "@/lib/maker";
import { useColorScheme } from "@/lib/use-color-scheme";
import { THEME } from "@/lib/theme";
const subscribeOnline = (notify: () => void) => onlineManager.subscribe(notify);
const getOnline = () => onlineManager.isOnline();
function useOnline() {
  return useSyncExternalStore(subscribeOnline, getOnline, getOnline);
}

export function NetworkNotice() {
  const online = useOnline();
  return online ? null : (
    <View
      accessibilityRole="alert"
      accessibilityLiveRegion="polite"
      className="px-5 py-3 bg-secondary"
    >
      <Text>
        网络已断开。你可以继续修改草稿，已暂停的请求会在连接恢复后继续。
      </Text>
    </View>
  );
}

export function useMaker() {
  const ws = useWorkspaceStore((s) => s.currentWorkspaceId);
  const slug = useWorkspaceStore((s) => s.currentWorkspaceSlug);
  const user = useAuthStore((s) => s.user);
  const scope = makerScope(user?.id, ws);
  useEffect(() => {
    if (user?.id && ws) void restoreMakerDraft(scope);
  }, [scope, user?.id, ws]);
  const draft = useMakerDraftStore(selectMakerDraft(scope));
  const patch = useMakerDraftStore((s) => s.patch);
  const update = useCallback(
    (v: Parameters<typeof patch>[1]) => patch(scope, v),
    [patch, scope],
  );
  const focused = useIsFocused();
  const { colorScheme } = useColorScheme();
  return {
    ws,
    slug,
    user,
    scope,
    draft,
    patch: update,
    focused,
    theme: THEME[colorScheme],
  };
}
export function MakerPage({
  title,
  subtitle,
  children,
}: PropsWithChildren<{ title: string; subtitle: string }>) {
  const slug = useWorkspaceStore((s) => s.currentWorkspaceSlug);
  const user = useAuthStore((s) => s.user);
  const ws = useWorkspaceStore((s) => s.currentWorkspaceId);
  const scope = makerScope(user?.id, ws);
  const storageError = useMakerDraftStore((s) => s.storageErrors[scope]);
  return (
    <SafeAreaView edges={["top"]} className="flex-1 bg-background">
      <View className="flex-row items-center justify-between px-5 pt-3 pb-4 gap-3">
        <View className="flex-1">
          <Text className="text-3xl font-bold">{title}</Text>
          <Text className="mt-1 text-sm text-muted-foreground">{subtitle}</Text>
        </View>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="家庭与账户"
          onPress={() => router.push(`/${slug}/account`)}
          className="size-12 rounded-full bg-secondary items-center justify-center"
        >
          <Text className="font-bold text-lg">
            {(user?.name || user?.email || "我").slice(0, 1).toUpperCase()}
          </Text>
        </Pressable>
      </View>
      <NetworkNotice />
      {storageError && (
        <Text accessibilityRole="alert" className="px-5 py-2 text-destructive">
          本机草稿暂未保存，请保持 App 打开并保存修改。
        </Text>
      )}
      {children}
    </SafeAreaView>
  );
}
export function MakerScroll({ children }: PropsWithChildren) {
  return (
    <ScrollView
      keyboardShouldPersistTaps="handled"
      contentContainerStyle={{
        padding: 20,
        paddingTop: 4,
        gap: 20,
        paddingBottom: 32,
      }}
    >
      {children}
    </ScrollView>
  );
}
export function Panel({ children }: PropsWithChildren) {
  return (
    <View className="rounded-xl border border-border bg-card p-4 gap-3">
      {children}
    </View>
  );
}
export function Notice({
  error,
  retry,
}: {
  error: unknown;
  retry?: () => void;
}) {
  if (!error) return null;
  return (
    <View
      accessibilityRole="alert"
      className="p-4 rounded-xl bg-secondary gap-3"
    >
      <Text>{makerError(error)}</Text>
      {retry && (
        <Button variant="outline" size="lg" onPress={retry}>
          <Text>重新加载</Text>
        </Button>
      )}
    </View>
  );
}
export function Loading() {
  const online = useOnline();
  return (
    <View className="p-8 items-center gap-3">
      {online && <ActivityIndicator />}
      <Text className="text-muted-foreground">
        {online ? "正在加载…" : "等待网络连接…"}
      </Text>
    </View>
  );
}
export function Choice({
  label,
  selected,
  onPress,
  disabled,
}: {
  label: string;
  selected?: boolean;
  onPress: () => void;
  disabled?: boolean;
}) {
  return (
    <Button
      size="lg"
      variant={selected ? "default" : "outline"}
      onPress={onPress}
      disabled={disabled}
      accessibilityState={{ selected, disabled }}
    >
      <Text>{label}</Text>
    </Button>
  );
}
export function Quantity({
  name,
  value,
  onChange,
  disabled,
  max = 999,
}: {
  name: string;
  value: number;
  onChange: (value: number) => void;
  disabled?: boolean;
  max?: number;
}) {
  const [editing, setEditing] = useState(false);
  const [text, setText] = useState(String(value));
  return (
    <View className="flex-row items-center gap-2">
      <Button
        size="lg"
        variant="outline"
        disabled={disabled || value <= 0}
        accessibilityLabel={`减少${name}`}
        onPress={() => {
          setText(String(value - 1));
          onChange(value - 1);
        }}
      >
        <Text>−</Text>
      </Button>
      <TextField
        accessibilityLabel={`${name}数量`}
        value={editing ? text : String(value)}
        keyboardType="number-pad"
        selectTextOnFocus
        editable={!disabled}
        onFocus={() => {
          setText(String(value));
          setEditing(true);
        }}
        onBlur={() => setEditing(false)}
        onChangeText={(input) => {
          const digits = input.replace(/[^0-9]/g, "");
          const next = quantityInput(digits, value, max);
          setText(digits === "" ? "" : String(next));
          if (digits !== "") onChange(next);
        }}
        className="w-14 text-center"
        style={{ minHeight: Platform.OS === "android" ? 48 : 44 }}
      />
      <Button
        size="lg"
        variant="outline"
        disabled={disabled || value >= max}
        accessibilityLabel={`增加${name}`}
        onPress={() => {
          setText(String(value + 1));
          onChange(value + 1);
        }}
      >
        <Text>＋</Text>
      </Button>
    </View>
  );
}
export function RowLink({
  label,
  detail,
  icon = "chevron-forward",
  onPress,
}: {
  label: string;
  detail?: string;
  icon?: React.ComponentProps<typeof Ionicons>["name"];
  onPress: () => void;
}) {
  const { colorScheme } = useColorScheme();
  const theme = THEME[colorScheme];
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      className="flex-row items-center gap-3 rounded-xl bg-secondary p-4"
    >
      <View className="flex-1 gap-1">
        <Text className="font-semibold">{label}</Text>
        {detail && (
          <Text className="text-sm text-muted-foreground">{detail}</Text>
        )}
      </View>
      <Ionicons name={icon} color={theme.foreground} size={22} />
    </Pressable>
  );
}

export function MakerDetailHeader({ title }: { title: string }) {
  const slug = useWorkspaceStore((s) => s.currentWorkspaceSlug);
  // Android's nested native stacks overlap the status bar under edge-to-edge.
  // Reuse the existing safe-area header there; iOS retains its native back header.
  return (
    <>
      <Stack.Screen
        options={{
          title,
          headerShown: Platform.OS !== "android",
          headerBackTitle: "返回",
        }}
      />
      {Platform.OS === "android" && (
        <Header
          title={title}
          left={
            <IconButton
              name="arrow-back"
              accessibilityLabel="返回"
              onPress={() => {
                if (router.canGoBack()) router.back();
                else router.replace(`/${slug}/creations`);
              }}
            />
          }
        />
      )}
    </>
  );
}
