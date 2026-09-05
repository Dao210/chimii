import { Alert, FlatList, View } from "react-native";
import { Stack, router, useLocalSearchParams } from "expo-router";
import { SafeAreaView } from "react-native-safe-area-context";
import { useQuery } from "@tanstack/react-query";
import { Text } from "@/components/ui/text";
import { Button } from "@/components/ui/button";
import { Loading, Notice, Quantity, useMaker } from "@/components/maker/shared";
import { circuitCatalogOptions, circuitInventoryOptions, useSaveCircuitInventory } from "@/data/queries/maker";
export default function CircuitInventory() {
 const { kit = "" } = useLocalSearchParams<{ kit: string }>(); const { ws, draft, patch } = useMaker();
 const catalog = useQuery(circuitCatalogOptions(ws, kit)); const inventory = useQuery(circuitInventoryOptions(ws, kit)); const save = useSaveCircuitInventory(ws, kit);
 const local = draft.circuits?.[kit]; const values = local?.quantities ?? inventory.data?.quantities ?? {};
 const change = (part: string, n: number) => { if (!inventory.data) return; patch({ circuits: { ...draft.circuits, [kit]: { revision: local?.revision ?? inventory.data.revision, quantities: { ...values, [part]: n } } } }); };
 const clear = () => { const next = { ...draft.circuits }; delete next[kit]; patch({ circuits: next }); };
 const store = async () => { if (!inventory.data || !catalog.data) return; try { await save.mutateAsync({ catalog_version: catalog.data.catalog.version, quantities: values, expected_revision: local?.revision ?? inventory.data.revision }); clear(); router.back(); } catch { /* Draft stays intact. */ } };
 return <SafeAreaView edges={["bottom"]} className="flex-1 bg-background"><Stack.Screen options={{ title: "确认我的套件" }} />
  <View className="p-5 gap-3"><Text className="text-lg font-bold">{catalog.data?.catalog.name}</Text><Text className="text-muted-foreground">只填实际拥有的数量，没有的零件保持为 0。</Text><Notice error={catalog.error || inventory.error} retry={() => { void catalog.refetch(); void inventory.refetch(); }} /><Notice error={save.error} />
  {save.isError && <Button variant="outline" onPress={() => Alert.alert("读取最新套件？", "会丢弃本机未保存的数量修改。", [{ text: "保留", style: "cancel" }, { text: "读取最新", onPress: () => { clear(); save.reset(); void inventory.refetch(); } }])}><Text>重新读取套件</Text></Button>}</View>
  {catalog.isLoading || inventory.isLoading ? <Loading /> : <FlatList data={catalog.data?.catalog.parts ?? []} keyExtractor={v => v.id} renderItem={({ item }) => <View className="p-5 border-b border-border gap-3"><Text className="font-semibold">{item.name.zh}</Text><Text className="text-sm text-muted-foreground">{item.purpose.zh}</Text><Quantity name={item.name.zh} value={values[item.id] ?? 0} onChange={n => change(item.id, n)} disabled={!inventory.data?.can_edit || save.isPending} /></View>} />}
  <View className="p-4 border-t border-border gap-2">{inventory.data && !inventory.data.can_edit && <Text className="text-muted-foreground">请让家长或空间管理员修改套件。</Text>}<Button size="lg" disabled={!inventory.data?.can_edit || !catalog.data || save.isPending} onPress={() => void store()}><Text>{save.isPending ? "正在保存…" : "确认并保存套件"}</Text></Button></View>
 </SafeAreaView>;
}
