"use client";

import { BuildPage } from "@chimii/views/build";
import { ErrorBoundary } from "@chimii/ui/components/common/error-boundary";

export default function Page() {
  return <ErrorBoundary><BuildPage /></ErrorBoundary>;
}
