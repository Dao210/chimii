"use client";

import { CreationsPage } from "@chimii/views/build";
import { ErrorBoundary } from "@chimii/ui/components/common/error-boundary";

export default function Page() {
  return <ErrorBoundary><CreationsPage /></ErrorBoundary>;
}
