"use client";

import { useEffect, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { childModeOptions } from "@chimii/core/child-mode";
import { useWorkspacePaths } from "@chimii/core/paths";
import { useNavigation } from "../navigation";
import { SidebarProvider, SidebarInset } from "@chimii/ui/components/ui/sidebar";
import { ModalRegistry } from "../modals/registry";
import { SourceBackfillModal } from "../onboarding";
import { AppSidebar } from "./app-sidebar";
import { DashboardGuard } from "./dashboard-guard";
import { NavigationProgress } from "./navigation-progress";
import { WorkspacePresencePrefetch } from "./workspace-presence-prefetch";
import { GlobalShortcuts } from "./global-shortcuts";

interface DashboardLayoutProps {
  children: ReactNode;
  /** Rendered inside SidebarInset (e.g. ChatWindow, ChatFab — absolute-positioned overlays) */
  extra?: ReactNode;
  /** Rendered inside sidebar header as a search trigger */
  searchSlot?: ReactNode;
  /** Loading indicator */
  loadingIndicator?: ReactNode;
}

export function DashboardLayout({
  children,
  extra,
  searchSlot,
  loadingIndicator,
}: DashboardLayoutProps) {
  const { data: childMode } = useQuery(childModeOptions());
  const isChildMode = childMode?.mode === "child";
  const parentModeReady = childMode?.mode === "parent";
  const workspacePaths = useWorkspacePaths();
  const { pathname, replace } = useNavigation();
  useEffect(() => {
    if (!isChildMode) return;
    const allowed = pathname === workspacePaths.build() || pathname.startsWith(`${workspacePaths.creations()}/`) || pathname === workspacePaths.creations();
    if (!allowed) replace(workspacePaths.build());
  }, [isChildMode, pathname, replace, workspacePaths]);
  return (
    <DashboardGuard
      loadingFallback={
        <div className="flex h-svh items-center justify-center">
          {loadingIndicator}
        </div>
      }
    >
      <SidebarProvider className="h-svh bg-app-shell">
        {parentModeReady && <GlobalShortcuts />}
        {parentModeReady && <WorkspacePresencePrefetch />}
        <AppSidebar searchSlot={searchSlot} />
        <SidebarInset className="relative overflow-hidden">
          <NavigationProgress />
          {children}
          {parentModeReady && <ModalRegistry />}
          {parentModeReady && <SourceBackfillModal />}
          {parentModeReady && extra}
        </SidebarInset>
      </SidebarProvider>
    </DashboardGuard>
  );
}
