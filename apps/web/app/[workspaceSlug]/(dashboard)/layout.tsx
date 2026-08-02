"use client";

import { DashboardLayout } from "@chimii/views/layout";
import { ChimiiIcon } from "@chimii/ui/components/common/chimii-icon";
import { SearchCommand, SearchTrigger } from "@chimii/views/search";
import { FloatingChat } from "@chimii/views/chat";
import { WebNotificationBridge } from "@/components/web-notification-bridge";

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <DashboardLayout
      loadingIndicator={<ChimiiIcon className="size-6" />}
      searchSlot={<SearchTrigger />}
      extra={
        <>
          <SearchCommand />
          <WebNotificationBridge />
          <FloatingChat />
        </>
      }
    >
      {children}
    </DashboardLayout>
  );
}
