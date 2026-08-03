import { redirect } from "next/navigation";
import { paths } from "@chimii/core/paths";

export default async function WorkspaceIndexPage({
  params,
}: {
  params: Promise<{ workspaceSlug: string }>;
}) {
  const { workspaceSlug } = await params;
  redirect(paths.workspace(workspaceSlug).build());
}
