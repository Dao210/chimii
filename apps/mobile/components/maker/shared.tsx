import type { PropsWithChildren } from "react";
import { ActivityIndicator, Pressable, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { router } from "expo-router";
import { Ionicons } from "@expo/vector-icons";
import { useIsFocused } from "@react-navigation/native";
import { Text } from "@/components/ui/text";
import { Button } from "@/components/ui/button";
import { useWorkspaceStore } from "@/data/workspace-store";
import { useAuthStore } from "@/data/auth-store";
import { useMakerDraftStore, selectMakerDraft } from "@/data/stores/maker-draft-store";
import { makerError, makerScope } from "@/lib/maker";
import { useColorScheme } from "@/lib/use-color-scheme";
import { THEME } from "@/lib/theme";

export function useMaker() {
  const ws = useWorkspaceStore(s => s.currentWorkspaceId);
  const slug = useWorkspaceStore(s => s.currentWorkspaceSlug);
  const user = useAuthStore(s => s.user);
  const scope = makerScope(user?.id, ws);
  const draft = useMakerDraftStore(selectMakerDraft(scope));
  const patch = useMakerDraftStore(s => s.patch);
  const focused = useIsFocused();
  const { colorScheme } = useColorScheme();
  return { ws, slug, user, scope, draft, patch: (v: Parameters<typeof patch>[1]) => patch(scope, v), focused, theme: THEME[colorScheme] };
}
export function MakerPage({ title, subtitle, children }: PropsWithChildren<{ title: string; subtitle: string }>) {
  const { slug, user } = useMaker();
  return <SafeAreaView edges={["top"]} className="flex-1 bg-background">
    <View className="flex-row items-center justify-between px-5 pt-3 pb-4 gap-3">
      <View className="flex-1"><Text className="text-3xl font-bold">{title}</Text><Text className="mt-1 text-sm text-muted-foreground">{subtitle}</Text></View>
      <Pressable accessibilityRole="button" accessibilityLabel="家庭与账户" onPress={() => router.push(`/${slug}/account`)} className="size-12 rounded-full bg-secondary items-center justify-center">
        <Text className="font-bold text-lg">{(user?.name || user?.email || "我").slice(0, 1).toUpperCase()}</Text>
      </Pressable>
    </View>{children}
  </SafeAreaView>;
}
export function MakerScroll({ children }: PropsWithChildren) {
  return <ScrollView keyboardShouldPersistTaps="handled" contentContainerStyle={{ padding: 20, paddingTop: 4, gap: 20, paddingBottom: 32 }}>{children}</ScrollView>;
}
export function Panel({ children }: PropsWithChildren) { return <View className="rounded-xl border border-border bg-card p-4 gap-3">{children}</View>; }
export function Notice({ error, retry }: { error: unknown; retry?: () => void }) {
  if (!error) return null;
  return <View accessibilityRole="alert" className="p-4 rounded-xl bg-secondary gap-3"><Text>{makerError(error)}</Text>{retry && <Button variant="outline" size="lg" onPress={retry}><Text>重新加载</Text></Button>}</View>;
}
export function Loading() { return <View className="p-8 items-center gap-3"><ActivityIndicator /><Text className="text-muted-foreground">正在加载…</Text></View>; }
export function Choice({ label, selected, onPress, disabled }: { label: string; selected?: boolean; onPress: () => void; disabled?: boolean }) {
  return <Button size="lg" variant={selected ? "default" : "outline"} onPress={onPress} disabled={disabled} accessibilityState={{ selected, disabled }}><Text>{label}</Text></Button>;
}
export function Quantity({ name, value, onChange, disabled }: { name: string; value: number; onChange: (value: number) => void; disabled?: boolean }) {
  return <View className="flex-row items-center gap-2"><Button size="lg" variant="outline" disabled={disabled || value <= 0} accessibilityLabel={`减少${name}`} onPress={() => onChange(value - 1)}><Text>−</Text></Button><Text accessibilityLabel={`${name}数量 ${value}`} className="min-w-8 text-center font-bold">{value}</Text><Button size="lg" variant="outline" disabled={disabled || value >= 9999} accessibilityLabel={`增加${name}`} onPress={() => onChange(value + 1)}><Text>＋</Text></Button></View>;
}
export function RowLink({ label, detail, icon = "chevron-forward", onPress }: { label: string; detail?: string; icon?: React.ComponentProps<typeof Ionicons>["name"]; onPress: () => void }) {
  const { theme } = useMaker();
  return <Pressable accessibilityRole="button" onPress={onPress} className="flex-row items-center gap-3 rounded-xl bg-secondary p-4"><View className="flex-1 gap-1"><Text className="font-semibold">{label}</Text>{detail && <Text className="text-sm text-muted-foreground">{detail}</Text>}</View><Ionicons name={icon} color={theme.foreground} size={22} /></Pressable>;
}
