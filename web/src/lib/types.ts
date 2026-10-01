/**
 * Types mirroring the companion program API (127.0.0.1:7799).
 */

/** Mount / item status. */
export type ItemStatus = "OK" | "BROKEN" | "MISMATCH" | "MISSING" | "SKIP";

/** How a mount is realized on disk. */
export type MountKind = "link" | "mirror" | "per-skill";

/** Platform support level. */
export type SupportLevel = "ga" | "beta";

export interface MountEntry {
  from: string;
  to: string;
  kind: MountKind;
  status: ItemStatus;
}

export interface InjectEntry {
  path: string;
  status: ItemStatus;
}

export interface PlatformInfo {
  id: string;
  label: string;
  support: SupportLevel;
  status: ItemStatus;
  mounts: MountEntry[];
  inject: InjectEntry[];
}

/** GET /api/state */
export interface AppState {
  roots: {
    memory: string;
    skills: string;
  };
  platforms: PlatformInfo[];
}

/** GET /api/health */
export interface HealthInfo {
  ok: boolean;
  version: string;
  os: string;
  token_required: boolean;
}

/** One entry of enable/disable changes[]. Shape is backend-defined; keep permissive. */
export interface ChangeEntry {
  action?: string;
  path?: string;
  detail?: string;
  [key: string]: unknown;
}

/** POST /api/platforms/{id}/enable | disable */
export interface MutationResult {
  ok: boolean;
  backupId: string;
  changes: ChangeEntry[];
}
