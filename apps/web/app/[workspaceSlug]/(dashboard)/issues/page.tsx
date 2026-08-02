"use client";

import { IssuesPage } from "@chimii/views/issues/components";
import { ErrorBoundary } from "@chimii/ui/components/common/error-boundary";

export default function Page() {
  return (
    <ErrorBoundary>
      <IssuesPage />
    </ErrorBoundary>
  );
}
