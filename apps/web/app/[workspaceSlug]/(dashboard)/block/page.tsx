"use client";

import { BlockPage } from "@chimii/views/build";
import { ErrorBoundary } from "@chimii/ui/components/common/error-boundary";

export default function Page() {
  return <ErrorBoundary><BlockPage /></ErrorBoundary>;
}
