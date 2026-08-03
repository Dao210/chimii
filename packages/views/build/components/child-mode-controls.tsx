"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Baby, KeyRound, LoaderCircle, ShieldCheck, UserRoundPlus } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@chimii/ui/components/ui/button";
import { Input } from "@chimii/ui/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@chimii/ui/components/ui/dialog";
import { useWorkspaceId } from "@chimii/core/hooks";
import {
  childModeOptions,
  childProfilesOptions,
  useCreateChildProfile,
  useEnterChildMode,
  useExitChildMode,
} from "@chimii/core/child-mode";
import { useT } from "../../i18n";

export function ChildModeLauncher() {
  const { t } = useT("build");
  const workspaceId = useWorkspaceId();
  const { data: mode } = useQuery(childModeOptions());
  const { data: profiles = [] } = useQuery({
    ...childProfilesOptions(workspaceId),
    select: (data) => data.profiles,
    enabled: mode?.mode === "parent",
  });
  const createProfile = useCreateChildProfile();
  const enterMode = useEnterChildMode();
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [pin, setPIN] = useState("");

  if (mode?.mode !== "parent") return null;

  const addProfile = async () => {
    try {
      const profile = await createProfile.mutateAsync({ display_name: name.trim(), pin });
      setName("");
      setPIN("");
      if (profile.id) await enterMode.mutateAsync(profile.id);
      window.location.reload();
    } catch {
      toast.error(t($ => $.child_create_error));
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline" className="rounded-full border-2 border-[#1d241f] bg-[#fffdf7] font-bold" />}>
        <Baby className="size-4" /> {t($ => $.child_mode)}
      </DialogTrigger>
      <DialogContent className="rounded-[1.8rem] border-2 border-[#1d241f] bg-[#fffdf7] sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="text-2xl font-black">{t($ => $.child_dialog_title)}</DialogTitle>
          <DialogDescription>{t($ => $.child_dialog_description)}</DialogDescription>
        </DialogHeader>
        {profiles.length > 0 && (
          <div className="grid gap-2 py-2">
            {profiles.map((profile) => (
              <button
                key={profile.id}
                type="button"
                disabled={enterMode.isPending}
                onClick={async () => {
                  try { await enterMode.mutateAsync(profile.id); window.location.reload(); }
                  catch { toast.error(t($ => $.child_enter_error)); }
                }}
                className="flex min-h-14 items-center gap-3 rounded-2xl border-2 border-[#1d241f] bg-[#e7f4e5] px-4 text-left font-black transition hover:bg-[#ffd85a] disabled:opacity-50"
              >
                <span className="flex size-9 items-center justify-center rounded-xl bg-[#a8dfc2]"><Baby className="size-5" /></span>
                {profile.display_name}
                {enterMode.isPending && <LoaderCircle className="ml-auto size-4 animate-spin" />}
              </button>
            ))}
          </div>
        )}
        <div className="mt-2 rounded-2xl border border-[#d3c7ae] bg-[#f6efdf] p-4">
          <p className="mb-3 flex items-center gap-2 text-sm font-black"><UserRoundPlus className="size-4" /> {profiles.length ? t($ => $.child_add_another) : t($ => $.child_create_profile)}</p>
          <div className="grid gap-3">
            <Input value={name} onChange={(event) => setName(event.target.value)} maxLength={24} placeholder={t($ => $.child_name_placeholder)} className="h-11 rounded-xl border-[#9c9079] bg-white" />
            <Input value={pin} onChange={(event) => setPIN(event.target.value.replace(/\D/g, "").slice(0, 4))} inputMode="numeric" type="password" maxLength={4} placeholder={t($ => $.child_pin_placeholder)} className="h-11 rounded-xl border-[#9c9079] bg-white tracking-[.4em]" />
            <Button onClick={() => void addProfile()} disabled={!name.trim() || pin.length !== 4 || createProfile.isPending || enterMode.isPending} className="h-11 rounded-xl bg-[#1d241f] font-bold">
              {createProfile.isPending || enterMode.isPending ? <LoaderCircle className="size-4 animate-spin" /> : <ShieldCheck className="size-4" />} {t($ => $.child_create_enter)}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

export function ParentUnlockButton() {
  const { t } = useT("build");
  const exitMode = useExitChildMode();
  const [open, setOpen] = useState(false);
  const [pin, setPIN] = useState("");
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline" className="w-full justify-start" />}>
        <KeyRound className="size-4" /> {t($ => $.parent_unlock)}
      </DialogTrigger>
      <DialogContent className="rounded-[1.6rem] sm:max-w-sm">
        <DialogHeader><DialogTitle>{t($ => $.parent_unlock)}</DialogTitle><DialogDescription>{t($ => $.parent_unlock_description)}</DialogDescription></DialogHeader>
        <Input autoFocus value={pin} onChange={(event) => setPIN(event.target.value.replace(/\D/g, "").slice(0, 4))} onKeyDown={(event) => { if (event.key === "Enter" && pin.length === 4) void exitMode.mutateAsync(pin).then(() => window.location.reload()).catch(() => toast.error(t($ => $.parent_pin_error))); }} inputMode="numeric" type="password" maxLength={4} className="h-14 text-center text-xl tracking-[.6em]" />
        <Button disabled={pin.length !== 4 || exitMode.isPending} onClick={() => void exitMode.mutateAsync(pin).then(() => window.location.reload()).catch(() => toast.error(t($ => $.parent_pin_error)))}>
          {exitMode.isPending && <LoaderCircle className="size-4 animate-spin" />} {t($ => $.parent_unlock_action)}
        </Button>
      </DialogContent>
    </Dialog>
  );
}
