import type { PresetInfo, PresetResult } from "../lib/types";
import { useT } from "../lib/i18n";

interface PresetCardProps {
  preset: PresetInfo;
  busy: boolean;
  result: PresetResult | null;
  error: string | null;
  onDryRun: (preset: PresetInfo) => void;
  onApply: (preset: PresetInfo, action: "enable" | "disable") => void;
}

/**
 * One configured preset.
 *
 * The dry-run button is deliberately the first one: a batch touches several ends at
 * once, so the console must let you see which ends would change before you commit.
 */
export default function PresetCard({
  preset,
  busy,
  result,
  error,
  onDryRun,
  onApply,
}: PresetCardProps) {
  const { t } = useT();
  const failed = result ? result.members.filter((m) => !m.ok) : [];

  return (
    <section className="glass-panel flex flex-col gap-3 p-5 transition-all duration-200 hover:-translate-y-1 hover:border-amber-400/40">
      <div className="flex items-start justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="text-base font-semibold text-zinc-100">{preset.label}</h2>
          <span className="rounded bg-zinc-800/80 px-1.5 py-0.5 text-[10px] uppercase tracking-wider text-zinc-400">
            {preset.name}
          </span>
        </div>
        <span className="text-[11px] text-zinc-500">{t("presetMembers", { n: preset.platforms.length })}</span>
      </div>

      {preset.note && <p className="text-xs leading-relaxed text-zinc-400">{preset.note}</p>}

      <div className="flex flex-wrap gap-1.5">
        {preset.platforms.map((id) => (
          <span
            key={id}
            className="rounded bg-zinc-800/70 px-1.5 py-0.5 text-[10px] tracking-wide text-zinc-400"
          >
            {id}
          </span>
        ))}
      </div>

      <div className="flex flex-wrap gap-2 pt-1">
        <button
          type="button"
          disabled={busy}
          onClick={() => onDryRun(preset)}
          className="rounded-lg border border-zinc-700 px-3 py-1.5 text-xs font-medium text-zinc-300 transition-colors hover:border-amber-400/50 hover:text-amber-200 disabled:opacity-50"
        >
          {t("presetDryRun")}
        </button>
        <button
          type="button"
          disabled={busy}
          onClick={() => onApply(preset, "enable")}
          className="btn-primary rounded-lg px-3 py-1.5 text-xs font-semibold disabled:opacity-50"
        >
          {busy ? t("presetApplying") : t("presetEnable")}
        </button>
        <button
          type="button"
          disabled={busy}
          onClick={() => onApply(preset, "disable")}
          title={t("presetSoftDisable")}
          className="btn-ghost rounded-lg px-3 py-1.5 text-xs font-medium disabled:opacity-50"
        >
          {t("presetDisable")}
        </button>
      </div>

      {error && (
        <p className="rounded-lg border border-red-400/25 bg-red-400/[0.07] px-3 py-2 text-xs text-red-300">
          {error}
        </p>
      )}

      {result && (
        <div className="rounded-lg border border-zinc-700/60 bg-zinc-900/40 px-3 py-2 text-xs">
          {result.dryRun ? (
            <p className="text-zinc-300">
              {t("presetDryRunResult", {
                n: result.members.length,
                action: result.action === "enable" ? t("presetEnable") : t("presetDisable"),
              })}
            </p>
          ) : (
            <p className={result.ok ? "text-teal-300" : "text-amber-300"}>
              {t(result.ok ? "presetOk" : "presetFail", {
                preset: result.label,
                ok: result.members.length - failed.length,
                fail: failed.length,
              })}
            </p>
          )}
          {failed.length > 0 && (
            <ul className="mt-2 flex flex-col gap-1">
              {failed.map((m) => (
                <li key={m.id} className="text-red-300">
                  <span className="text-zinc-400">{m.id}</span>
                  {m.error ? ` · ${m.error}` : ""}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </section>
  );
}