import { z } from "zod";
import { EMPTY_USER, UserSchema } from "../api/schemas";
import type { ChildMode, ChildProfile, ChildProfileList, EnterChildModeResponse, ExitChildModeResponse } from "./types";

export const ChildProfileSchema = z.looseObject({ id: z.string(), display_name: z.string(), avatar_seed: z.string() });
export const ChildModeSchema = z.looseObject({
  mode: z.enum(["parent", "child"]),
  profile: ChildProfileSchema.optional(),
  capabilities: z.array(z.string()),
});
export const ChildProfileListSchema = z.looseObject({ profiles: z.array(ChildProfileSchema) });
export const EnterChildModeResponseSchema = z.looseObject({ token: z.string().startsWith("mch_"), mode: ChildModeSchema });
export const ExitChildModeResponseSchema = z.looseObject({ token: z.string(), user: UserSchema });

export const PARENT_MODE: ChildMode = { mode: "parent", capabilities: ["*"] };
export const EMPTY_CHILD_PROFILE: ChildProfile = { id: "", display_name: "", avatar_seed: "" };
export const EMPTY_CHILD_PROFILES: ChildProfileList = { profiles: [] };
export const EMPTY_ENTER_CHILD_MODE: EnterChildModeResponse = { token: "", mode: PARENT_MODE };
export const EMPTY_EXIT_CHILD_MODE: ExitChildModeResponse = { token: "", user: EMPTY_USER };
