"use client";

import { useQuery } from "@tanstack/react-query";
import { Loader2, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { ldrawCatalogSyncOptions, useStartLDrawCatalogSync } from "@chimii/core/build";
import { useWorkspaceId } from "@chimii/core/hooks";
import { Badge } from "@chimii/ui/components/ui/badge";
import { Button } from "@chimii/ui/components/ui/button";
import { Progress } from "@chimii/ui/components/ui/progress";
import { useT } from "../../i18n";
import {
  SettingsCard,
  SettingsRow,
  SettingsSection,
  SettingsTab,
} from "./settings-layout";

export function LabsTab() {
  const { t } = useT("settings");
  const workspaceId = useWorkspaceId();
  const statusQuery = useQuery(ldrawCatalogSyncOptions(workspaceId));
  const startSync = useStartLDrawCatalogSync();
  const status = statusQuery.data;
  const active = status?.status === "queued" || status?.status === "running";
  const target = status?.target_part_count ?? 0;
  const completed = status?.progress_part_count ?? 0;
  const progress = target > 0 ? Math.min(100, Math.round((completed / target) * 100)) : 0;

  async function handleSync() {
    try {
      await startSync.mutateAsync();
      toast.success(t(($) => $.labs.ldraw_sync_started));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t(($) => $.labs.ldraw_sync_failed));
    }
  }

  const statusLabel = status
    ? t(($) => $.labs.ldraw_status[status.status])
    : t(($) => $.labs.ldraw_status.loading);

  return (
    <SettingsTab
      title={t(($) => $.page.tabs.labs)}
      description={t(($) => $.labs.page_description)}
    >
      <SettingsSection
        title={t(($) => $.labs.ldraw_title)}
        description={t(($) => $.labs.ldraw_description)}
      >
        <SettingsCard>
          <SettingsRow
            label={t(($) => $.labs.ldraw_kit)}
            description={status?.catalog_version || t(($) => $.labs.ldraw_loading)}
          >
            <div className="flex items-center justify-end gap-2">
              <span className="text-xs tabular-nums text-muted-foreground">
                {status ? `${status.stored_part_count} / ${target}` : "—"}
              </span>
              <Badge variant="secondary">{statusLabel}</Badge>
            </div>
          </SettingsRow>

          <SettingsRow
            label={t(($) => $.labs.ldraw_progress)}
            description={t(($) => $.labs.ldraw_progress_description, {
              completed,
              total: target,
            })}
            size="text"
          >
            <Progress value={progress} aria-label={t(($) => $.labs.ldraw_progress)} />
          </SettingsRow>

          <SettingsRow
            label={t(($) => $.labs.ldraw_action)}
            description={
              status?.enabled
                ? status.can_manage
                  ? t(($) => $.labs.ldraw_action_description)
                  : t(($) => $.labs.ldraw_admin_only)
                : t(($) => $.labs.ldraw_operator_disabled)
            }
          >
            <Button
              size="sm"
              onClick={handleSync}
              disabled={!status?.enabled || !status.can_manage || active || startSync.isPending}
            >
              {active || startSync.isPending ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <RefreshCw className="size-4" />
              )}
              {active ? t(($) => $.labs.ldraw_syncing) : t(($) => $.labs.ldraw_sync)}
            </Button>
          </SettingsRow>

          {status?.status === "failed" && status.error ? (
            <div className="px-4 py-3 text-xs leading-5 text-destructive" role="alert">
              {status.error}
            </div>
          ) : null}
        </SettingsCard>
      </SettingsSection>
    </SettingsTab>
  );
}
