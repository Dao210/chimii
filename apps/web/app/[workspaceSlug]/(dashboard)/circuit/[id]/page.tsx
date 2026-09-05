"use client";
import { use } from "react";
import { CircuitDetailPage } from "@chimii/views/circuit";
import { ErrorBoundary } from "@chimii/ui/components/common/error-boundary";
export default function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <ErrorBoundary resetKeys={[id]}>
      <CircuitDetailPage creationId={id} />
    </ErrorBoundary>
  );
}
