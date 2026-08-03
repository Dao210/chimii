"use client";

import { use } from "react";
import { CreationDetailPage } from "@chimii/views/build";
import { ErrorBoundary } from "@chimii/ui/components/common/error-boundary";

export default function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return <ErrorBoundary resetKeys={[id]}><CreationDetailPage creationId={id} /></ErrorBoundary>;
}
