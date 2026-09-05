import { Redirect } from "expo-router";
import { useWorkspaceStore } from "@/data/workspace-store";
export default function LegacyCreations() {
 const slug = useWorkspaceStore(s => s.currentWorkspaceSlug);
 return <Redirect href={`/${slug}/creations`} />;
}
