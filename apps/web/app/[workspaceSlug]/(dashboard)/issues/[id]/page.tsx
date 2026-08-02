"use client";

import { use } from "react";
import { IssueDetail } from "@chimii/views/issues/components";
import { ErrorBoundary } from "@chimii/ui/components/common/error-boundary";

export default function IssueDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  return (
    <ErrorBoundary resetKeys={[id]}>
      <IssueDetail issueId={id} />
    </ErrorBoundary>
  );
}
