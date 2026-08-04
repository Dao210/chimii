"use client";

import { useEffect, useId, useState } from "react";
import type { FormEvent } from "react";
import { CloudCog, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { useConfigStore } from "@chimii/core/config";
import { useWorkspaceId } from "@chimii/core/hooks";
import { useCreateCloudRuntime } from "@chimii/core/runtimes";
import { Button } from "@chimii/ui/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@chimii/ui/components/ui/dialog";
import { Input } from "@chimii/ui/components/ui/input";
import { Label } from "@chimii/ui/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@chimii/ui/components/ui/select";
import { useT } from "../../i18n";

type CloudProvider = "anthropic" | "openai";

export function ManagedCloudRuntimeDialog({ onClose }: { onClose: () => void }) {
  const { t } = useT("runtimes");
  const wsId = useWorkspaceId();
  const providers = useConfigStore((state) => state.cloudRuntimeProviders).filter(
    (provider): provider is CloudProvider =>
      provider === "anthropic" || provider === "openai",
  );
  const [provider, setProvider] = useState<CloudProvider>(providers[0] ?? "anthropic");
  const [name, setName] = useState("");
  const [visibility, setVisibility] = useState<"private" | "public">("private");
  const createRuntime = useCreateCloudRuntime(wsId);
  const idPrefix = `managed-cloud-runtime-${useId().replace(/:/g, "")}`;

  useEffect(() => {
    if (!providers.includes(provider) && providers[0]) {
      setProvider(providers[0]);
    }
  }, [provider, providers]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    try {
      await createRuntime.mutateAsync({
        provider,
        name: name.trim() || undefined,
        visibility,
      });
      toast.success(t(($) => $.managed_cloud_runtime.toast_created));
      onClose();
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t(($) => $.managed_cloud_runtime.toast_create_failed),
      );
    }
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-base">
            <CloudCog className="size-4 text-muted-foreground" />
            {t(($) => $.managed_cloud_runtime.title)}
          </DialogTitle>
          <DialogDescription>
            {t(($) => $.managed_cloud_runtime.description)}
          </DialogDescription>
        </DialogHeader>

        <form id={`${idPrefix}-form`} onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor={`${idPrefix}-provider`}>
              {t(($) => $.managed_cloud_runtime.provider)}
            </Label>
            <Select
              items={providers.map((value) => ({ value, label: value }))}
              value={provider}
              onValueChange={(value) => value && setProvider(value as CloudProvider)}
            >
              <SelectTrigger id={`${idPrefix}-provider`} className="w-full">
                <SelectValue>{() => provider}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                {providers.map((value) => (
                  <SelectItem key={value} value={value}>
                    {value === "anthropic" ? "Anthropic" : "OpenAI"}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor={`${idPrefix}-name`}>
              {t(($) => $.managed_cloud_runtime.name)}
            </Label>
            <Input
              id={`${idPrefix}-name`}
              value={name}
              onChange={(event) => setName(event.target.value)}
              maxLength={100}
              placeholder={t(($) => $.managed_cloud_runtime.name_placeholder)}
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor={`${idPrefix}-visibility`}>
              {t(($) => $.managed_cloud_runtime.visibility)}
            </Label>
            <Select
              items={[
                { value: "private", label: t(($) => $.managed_cloud_runtime.private) },
                { value: "public", label: t(($) => $.managed_cloud_runtime.public) },
              ]}
              value={visibility}
              onValueChange={(value) =>
                value && setVisibility(value as "private" | "public")
              }
            >
              <SelectTrigger id={`${idPrefix}-visibility`} className="w-full">
                <SelectValue>
                  {() =>
                    visibility === "private"
                      ? t(($) => $.managed_cloud_runtime.private)
                      : t(($) => $.managed_cloud_runtime.public)
                  }
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="private">
                  {t(($) => $.managed_cloud_runtime.private)}
                </SelectItem>
                <SelectItem value="public">
                  {t(($) => $.managed_cloud_runtime.public)}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </form>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t(($) => $.managed_cloud_runtime.cancel)}
          </Button>
          <Button
            type="submit"
            form={`${idPrefix}-form`}
            disabled={createRuntime.isPending || providers.length === 0}
          >
            {createRuntime.isPending && <Loader2 className="size-4 animate-spin" />}
            {t(($) => $.managed_cloud_runtime.create)}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
