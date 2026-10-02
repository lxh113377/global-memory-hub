import { useCallback, useEffect, useState } from "react";
import PlatformCard from "./components/PlatformCard";
import Diagnostics from "./pages/Diagnostics";
import Settings from "./pages/Settings";
import Skills from "./pages/Skills";
import {
  ApiError,
  disablePlatform,
  enablePlatform,
  fetchHealth,
  fetchState,
  hasToken,
  initTokenFromHash,
  setToken,
} from "./lib/api";
import { I18nContext, initialLang, makeI18n, useT, type Lang } from "./lib/i18n";
import type {
  AppState,
  HealthInfo,
  MutationResult,
  PlatformInfo,
} from "./lib/types";

type ConnState = "checking" | "online" | "offline";
type BatchMode = "enable" | "disable";
type Route = "home" | "skills" | "diagnostics" | "settings";

const POLL_INTERVAL_MS = 5000;

function getRoute(): Route {
  const h = window.location.hash.replace(/^#\/?/, "");
  return h === "skills" || h === "diagnostics" || h === "settings" ? h : "home";
}

function describeError(e: unknown): string {
  if (e instanceof ApiError) return e.message;
  return "error";
}

function Stat({
  label,
  value,
  danger,
}: {
  label: string;
  value: number;
  danger?: boolean;
}) {
  return (
    <div className="flex items-baseline gap-2">
      <span
        className={`text-2xl font-bold tabular-nums ${
          danger ? "text-red-400" : "brand-gradient-text"
        }`}
      >
        {value}
      </span>
      <span className="text-xs text-zinc-500">{label}</span>
    </div>
  );
}

function Home({
  state,
  stateError,
  busy,
  results,
  actionErrors,
  onToggle,
}: {
  state: AppState | null;
  stateError: string | null;
  busy: Record<string, boolean>;
  results: Record<string, MutationResult>;
  actionErrors: Record<string, string>;
  onToggle: (p: PlatformInfo, next: boolean) => void;
}) {
  const { t } = useT();
  const platforms = state?.platforms ?? [];
  const activeCount = platforms.filter((p) => p.status === "OK").length;
  const brokenCount = platforms.filter(
    (p) => p.status === "BROKEN" || p.status === "MISMATCH" || p.status === "MISSING"
  ).length;

  return (
    <>
      <div className="glass-panel mb-5 flex flex-wrap items-center gap-x-10 gap-y-4 px-5 py-4">
        <Stat label={t("statActive")} value={activeCount} />
        <Stat label={t("statAbnormal")} value={brokenCount} danger={brokenCount > 0} />
        <div className="flex flex-col gap-0.5 text-xs text-zinc-500">
          <span>
            {t("memRoot")}
            <code className="text-amber-300/90">{state?.roots.memory ?? "--"}</code>
          </span>
          <span>
            {t("skillRoot")}
            <code className="text-amber-300/90">{state?.roots.skills ?? "--"}</code>
          </span>
        </div>
      </div>

      {stateError && (
        <p className="mb-4 rounded-lg border border-red-400/25 bg-red-400/[0.07] px-4 py-2.5 text-xs leading-relaxed text-red-300">
          {t("stateErrorPrefix")}
          {stateError}
        </p>
      )}

      {platforms.length === 0 && !stateError ? (
        <div className="glass-panel flex flex-col items-center gap-2 px-6 py-16 text-center">
          <span className="text-3xl">🔥</span>
          <p className="text-sm text-zinc-400">
            {state ? t("emptyText") : t("checkingText")}
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2 xl:grid-cols-3">
          {platforms.map((p) => (
            <PlatformCard
              key={p.id}
              platform={p}
              busy={Boolean(busy[p.id])}
              result={results[p.id] ?? null}
              error={actionErrors[p.id] ?? null}
              onToggle={onToggle}
            />
          ))}
        </div>
      )}
    </>
  );
}

function Shell() {
  const { t, lang } = useT();
  const [route, setRoute] = useState<Route>(getRoute);
  const [conn, setConn] = useState<ConnState>("checking");
  const [health, setHealth] = useState<HealthInfo | null>(null);
  const [state, setState] = useState<AppState | null>(null);
  const [stateError, setStateError] = useState<string | null>(null);
  const [tokenPresent, setTokenPresent] = useState(false);
  const [busy, setBusy] = useState<Record<string, boolean>>({});
  const [results, setResults] = useState<Record<string, MutationResult>>({});
  const [actionErrors, setActionErrors] = useState<Record<string, string>>({});
  const [batchBusy, setBatchBusy] = useState<BatchMode | null>(null);
  const [batchMsg, setBatchMsg] = useState<string | null>(null);
  const [tokenInput, setTokenInput] = useState("");

  useEffect(() => {
    const onHash = () => setRoute(getRoute());
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  const probeHealth = useCallback(async () => {
    try {
      const h = await fetchHealth();
      setHealth(h);
      setConn("online");
    } catch {
      setHealth(null);
      setConn("offline");
    }
  }, []);

  const refreshState = useCallback(async () => {
    try {
      const s = await fetchState();
      setState(s);
      setStateError(null);
      setConn("online");
    } catch (e) {
      setStateError(describeError(e));
      if (e instanceof ApiError && e.kind === "offline") {
        setConn("offline");
      }
    }
  }, []);

  useEffect(() => {
    initTokenFromHash();
    setTokenPresent(hasToken());
    void refreshState();
    void probeHealth();
    const timer = window.setInterval(() => void probeHealth(), POLL_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, [probeHealth, refreshState]);

  const handleToggle = useCallback(
    async (p: PlatformInfo, next: boolean) => {
      setBusy((prev) => ({ ...prev, [p.id]: true }));
      setActionErrors((prev) => {
        const copy = { ...prev };
        delete copy[p.id];
        return copy;
      });
      setResults((prev) => {
        const copy = { ...prev };
        delete copy[p.id];
        return copy;
      });
      try {
        const res = next ? await enablePlatform(p.id) : await disablePlatform(p.id);
        setResults((prev) => ({ ...prev, [p.id]: res }));
      } catch (e) {
        setActionErrors((prev) => ({ ...prev, [p.id]: describeError(e) }));
      } finally {
        setBusy((prev) => ({ ...prev, [p.id]: false }));
        await refreshState();
      }
    },
    [refreshState]
  );

  const runBatch = useCallback(
    async (mode: BatchMode) => {
      if (!state || batchBusy) return;
      setBatchBusy(mode);
      setBatchMsg(null);
      let okCount = 0;
      let failCount = 0;
      for (const p of state.platforms) {
        setBusy((prev) => ({ ...prev, [p.id]: true }));
        try {
          if (mode === "enable") {
            await enablePlatform(p.id);
          } else {
            await disablePlatform(p.id);
          }
          okCount += 1;
        } catch {
          failCount += 1;
        } finally {
          setBusy((prev) => ({ ...prev, [p.id]: false }));
        }
      }
      await refreshState();
      setBatchBusy(null);
      setBatchMsg(
        t("batchDone", {
          mode: mode === "enable" ? t("batchModeEnable") : t("batchModeDisable"),
          ok: okCount,
          fail: failCount,
        })
      );
    },
    [state, batchBusy, refreshState, t]
  );

  const anyBusy = batchBusy !== null;
  const anyInteraction = anyBusy || Object.values(busy).some(Boolean);

  const connLabel =
    conn === "checking"
      ? t("connChecking")
      : conn === "online"
        ? t("connOnline")
        : t("connOffline");
  const connColor =
    conn === "online" ? "text-teal-300" : conn === "checking" ? "text-zinc-400" : "text-red-300";
  const connDot =
    conn === "online"
      ? "bg-teal-400 text-teal-400"
      : conn === "checking"
        ? "bg-zinc-500 text-zinc-500"
        : "bg-red-400 text-red-400";

  const navItems: Array<{ route: Route; label: string; href: string }> = [
    { route: "home", label: t("navHome"), href: "#/" },
    { route: "skills", label: t("navSkills"), href: "#/skills" },
    { route: "diagnostics", label: t("navDiag"), href: "#/diagnostics" },
    { route: "settings", label: t("navSettings"), href: "#/settings" },
  ];

  return (
    <div className="flex min-h-screen flex-col">
      <header className="sticky top-0 z-20 border-b border-amber-500/15 bg-[#0b0d12]/80 backdrop-blur-md">
        <div className="mx-auto flex w-full max-w-7xl flex-wrap items-center justify-between gap-3 px-4 py-3 sm:px-6">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-gradient-to-br from-[#d4a017] to-[#ff6b1a] text-lg font-bold text-black shadow-glow">
              焚
            </div>
            <div>
              <h1 className="text-lg font-bold leading-tight brand-gradient-text">焚诀 Global</h1>
              <p className="text-[11px] leading-tight text-zinc-500">{t("appSub")}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {conn === "online" && health?.token_required && !tokenPresent && (
              <span className="hidden rounded-full border border-amber-400/40 bg-amber-400/10 px-2.5 py-1 text-[11px] text-amber-300 sm:inline">
                {t("needTokenBadge")}
              </span>
            )}
            <nav className="flex items-center gap-1 rounded-full border border-zinc-700/60 bg-black/30 p-1">
              {navItems.map((n) => (
                <a
                  key={n.route}
                  href={n.href}
                  className={`rounded-full px-3 py-1 text-xs transition-colors ${
                    route === n.route
                      ? "bg-amber-400/15 text-amber-300"
                      : "text-zinc-400 hover:text-zinc-200"
                  }`}
                >
                  {n.label}
                </a>
              ))}
            </nav>
            <div className="flex items-center gap-2 rounded-full border border-zinc-700/60 bg-black/30 px-3 py-1.5 text-xs">
              <span className={`h-2 w-2 rounded-full animate-breath ${connDot}`} />
              <span className={connColor}>{connLabel}</span>
              {conn === "online" && health && (
                <span className="text-zinc-500">
                  v{health.version} · {health.os}
                </span>
              )}
            </div>
          </div>
        </div>
      </header>

      {conn === "offline" && (
        <div className="border-b border-amber-500/15 bg-amber-500/[0.06] px-4 py-4 sm:px-6">
          <div className="mx-auto flex w-full max-w-7xl flex-col gap-1.5">
            <p className="text-sm font-semibold text-amber-300">{t("offlineTitle")}</p>
            <p className="text-xs leading-relaxed text-zinc-400">{t("offlineBody1")}</p>
            <p className="text-xs leading-relaxed text-zinc-500">{t("offlineBody2")}</p>
          </div>
        </div>
      )}

      {conn === "online" && health?.token_required && !tokenPresent && (
        <div className="border-b border-amber-500/15 bg-amber-500/[0.06] px-4 py-4 sm:px-6">
          <div className="mx-auto flex w-full max-w-7xl flex-col gap-2">
            <p className="text-sm font-semibold text-amber-300">{t("tokenTitle")}</p>
            <p className="text-xs leading-relaxed text-zinc-400">{t("tokenBody")}</p>
            <form
              className="flex flex-wrap items-center gap-2"
              onSubmit={(e) => {
                e.preventDefault();
                if (!tokenInput.trim()) return;
                setToken(tokenInput);
                setTokenPresent(true);
                void refreshState();
              }}
            >
              <input
                value={tokenInput}
                onChange={(e) => setTokenInput(e.target.value)}
                placeholder={t("tokenPlaceholder")}
                className="w-full max-w-md rounded-lg border border-zinc-700/60 bg-black/40 px-3 py-1.5 font-mono text-xs text-zinc-200 outline-none focus:border-amber-400/60"
              />
              <button type="submit" className="btn-primary text-xs">
                {t("tokenEnter")}
              </button>
            </form>
          </div>
        </div>
      )}

      <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-6 sm:px-6">
        {route === "home" && (
          <Home
            state={state}
            stateError={stateError}
            busy={busy}
            results={results}
            actionErrors={actionErrors}
            onToggle={handleToggle}
          />
        )}
        {route === "skills" && <Skills />}
        {route === "diagnostics" && <Diagnostics />}
        {route === "settings" && (
          <Settings
            state={state}
            healthVersion={conn === "online" && health ? health.version : null}
            lang={lang}
            onLangChange={(l: Lang) => {
              try {
                window.localStorage.setItem("fenjue_lang", l);
              } catch {
                // localStorage unavailable: language applies to this session only
              }
              window.location.reload();
            }}
          />
        )}
      </main>

      {route === "home" && (
        <footer className="sticky bottom-0 z-20 border-t border-amber-500/15 bg-[#0b0d12]/85 backdrop-blur-md">
          <div className="mx-auto flex w-full max-w-7xl flex-wrap items-center gap-3 px-4 py-3 sm:px-6">
            <button
              type="button"
              className="btn-primary"
              disabled={!state || anyInteraction}
              onClick={() => void runBatch("enable")}
            >
              {batchBusy === "enable" && (
                <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-black/40 border-t-transparent" />
              )}
              {t("batchEnable")}
            </button>
            <button
              type="button"
              className="btn-ghost"
              disabled={!state || anyInteraction}
              onClick={() => void runBatch("disable")}
            >
              {batchBusy === "disable" && (
                <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-amber-300/60 border-t-transparent" />
              )}
              {t("batchDisable")}
            </button>
            <button
              type="button"
              className="btn-ghost"
              disabled={anyInteraction}
              onClick={() => {
                void refreshState();
                void probeHealth();
              }}
            >
              {t("reprobe")}
            </button>
            {batchMsg && <span className="text-xs text-zinc-400">{batchMsg}</span>}
          </div>
        </footer>
      )}
    </div>
  );
}

export default function App() {
  const [lang] = useState<Lang>(initialLang);
  const i18n = makeI18n(lang);
  return (
    <I18nContext.Provider value={i18n}>
      <Shell />
    </I18nContext.Provider>
  );
}
