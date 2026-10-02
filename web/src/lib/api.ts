import type {
  AppState,
  HealthInfo,
  MutationResult,
  PresetResult,
  PresetsResponse,
  RootsSetResult,
  SkillsResponse,
  VerifyReport,
} from "./types";

const API_BASE = "http://127.0.0.1:7799";
const TOKEN_KEY = "fenjue_token";

export type ApiErrorKind = "offline" | "forbidden" | "unauthorized" | "network";

/** Normalized API failure: offline (program not running), 403 (token), other network/HTTP errors. */
export class ApiError extends Error {
  readonly kind: ApiErrorKind;
  readonly status?: number;

  constructor(kind: ApiErrorKind, message: string, status?: number) {
    super(message);
    this.name = "ApiError";
    this.kind = kind;
    this.status = status;
  }
}

/** Read #token=xxx from location.hash into sessionStorage, then wipe the fragment. */
export function initTokenFromHash(): void {
  if (typeof window === "undefined") return;
  const raw = window.location.hash.replace(/^#/, "");
  if (!raw) return;
  const params = new URLSearchParams(raw);
  const token = params.get("token");
  if (!token) return;
  try {
    window.sessionStorage.setItem(TOKEN_KEY, token);
  } catch {
    // sessionStorage unavailable (e.g. privacy mode): token simply not persisted
  }
  window.history.replaceState(null, "", window.location.pathname + window.location.search);
}

/** True when a token has been stored in sessionStorage. */
export function hasToken(): boolean {
  try {
    return Boolean(window.sessionStorage.getItem(TOKEN_KEY));
  } catch {
    return false;
  }
}

function getToken(): string {
  try {
    return window.sessionStorage.getItem(TOKEN_KEY) ?? "";
  } catch {
    return "";
  }
}

/** Persist a manually pasted token into sessionStorage. */
export function setToken(token: string): void {
  try {
    window.sessionStorage.setItem(TOKEN_KEY, token.trim());
  } catch {
    // sessionStorage unavailable: token kept in memory only for this session scope
  }
}

async function request<T>(path: string, method = "GET", body?: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  const token = getToken();
  if (token) headers["X-Fenjue-Token"] = token;
  if (body !== undefined) headers["Content-Type"] = "application/json";

  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError("offline", "未检测到本地程序，请确认本地伴随程序已启动");
  }

  if (res.status === 401) {
    throw new ApiError("unauthorized", "未带令牌（401）：请粘贴本地程序启动时打印的令牌，或通过握手链接进入");
  }
  if (res.status === 403) {
    throw new ApiError("forbidden", "令牌校验失败（403），请通过带令牌的入口链接重新进入");
  }
  if (!res.ok) {
    throw new ApiError("network", `本地程序通信异常（HTTP ${res.status}）`, res.status);
  }
  try {
    return (await res.json()) as T;
  } catch {
    throw new ApiError("network", "本地程序响应格式异常");
  }
}

/** Probe the companion program. */
export function fetchHealth(): Promise<HealthInfo> {
  return request<HealthInfo>("/api/health");
}

/** Full platform state. */
export function fetchState(): Promise<AppState> {
  return request<AppState>("/api/state");
}

/** Enable a platform. roots is optional; omit to let the backend use its configured roots. */
export function enablePlatform(
  id: string,
  roots?: { memory: string; skills: string }
): Promise<MutationResult> {
  return request<MutationResult>(
    `/api/platforms/${encodeURIComponent(id)}/enable`,
    "POST",
    roots ? { roots } : {}
  );
}

/** Disable a platform (soft mode). */
export function disablePlatform(id: string): Promise<MutationResult> {
  return request<MutationResult>(
    `/api/platforms/${encodeURIComponent(id)}/disable`,
    "POST",
    { soft: true }
  );
}

/** List skills in the unified skills root (live disk read). */
export function fetchSkills(): Promise<SkillsResponse> {
  return request<SkillsResponse>("/api/skills");
}

/** Run the four-state verify report. */
export function runVerify(): Promise<VerifyReport> {
  return request<VerifyReport>("/api/verify", "POST");
}

/** Manually re-sync mirror mounts of a platform. */
export function syncPlatform(id: string): Promise<MutationResult> {
  return request<MutationResult>(
    `/api/platforms/${encodeURIComponent(id)}/sync`,
    "POST",
    {}
  );
}

/** List configured presets (empty array when platforms.json has no presets section). */
export function fetchPresets(): Promise<PresetsResponse> {
  return request<PresetsResponse>("/api/presets");
}

/**
 * Apply a preset to every member. dryRun=true resolves members and reports without
 * touching a single file, which is what the console shows before the real button.
 */
export function applyPreset(
  name: string,
  action: "enable" | "disable",
  dryRun = false
): Promise<PresetResult> {
  return request<PresetResult>(
    `/api/presets/${encodeURIComponent(name)}`,
    "POST",
    { action, dry_run: dryRun }
  );
}

/** Persist custom library roots (takes effect after agent restart). */
export function setRoots(
  memory: string,
  skills: string
): Promise<RootsSetResult> {
  return request<RootsSetResult>("/api/roots", "POST", { memory, skills });
}

/** Best-effort error text from an unknown catch value. */
export function errText(e: unknown): string {
  if (e instanceof Error) return e.message;
  return String(e);
}
