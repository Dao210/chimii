import { useLocalSearchParams } from "expo-router";
import { CreationDetail } from "@/components/maker/creation-detail";
export default function Detail() { const { id = "", kind = "" } = useLocalSearchParams<{ id: string; kind: string }>(); return <CreationDetail id={id} kind={kind} />; }
