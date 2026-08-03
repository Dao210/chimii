import type { User } from "../types";

export interface ChildProfile {
  id: string;
  display_name: string;
  avatar_seed: string;
}

export interface ChildMode {
  mode: "parent" | "child";
  profile?: ChildProfile;
  capabilities: string[];
}

export interface ChildProfileList { profiles: ChildProfile[] }

export interface EnterChildModeResponse {
  token: string;
  mode: ChildMode;
}

export interface ExitChildModeResponse {
  token: string;
  user: User;
}
