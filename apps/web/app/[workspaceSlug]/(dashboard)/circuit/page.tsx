"use client";
import { CircuitPage } from "@chimii/views/circuit";
import { ErrorBoundary } from "@chimii/ui/components/common/error-boundary";
export default function Page() {
  return (
    <ErrorBoundary>
      <CircuitPage />
    </ErrorBoundary>
  );
}
