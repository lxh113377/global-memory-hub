import { useState } from "react";
import { runVerify, errText } from "../lib/api";
import type { VerifyReport } from "../lib/types";
import { useT } from "../lib/i18n";

export default function Diagnostics() {
  const { t } = useT();
  const [report, setReport] = useState<VerifyReport | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const run = async () => {
    setBusy(true);
    setErr(null);
    try {
      setReport(await runVerify());
    } catch (e) {
      setErr(errText(e));
    } finally {
      setBusy(false);
    }
  };

  const issueCount =
    (report?.broken.length ?? 0) +
    (report?.mismatch.length ?? 0) +
    (report?.missing.length ?? 0);

  return (
    <div className="flex flex-col gap-4">
      <div className="glass-panel px-5 py-4">
        <h2 className="text-lg font-bold brand-gradient-text">{t("diagTitle")}</h2>
        <p className="mt-1 text-xs leading-relaxed text-zinc-400">{t("diagSub")}</p>
      </div>

      <button type="button" className="btn-primary w-fit" disabled={busy} onClick={() => void run()}>
        {busy ? t("diagRunning") : t("diagRun")}
      </button>

      {err && (
        <p className="rounded-lg border border-red-400/25 bg-red-400/[0.07] px-4 py-2.5 text-xs text-red-300">
          {t("fetchErr")}
          {err}
        </p>
      )}

      {report && (
        <div className="flex flex-col gap-3">
          <div className="glass-panel px-5 py-4 text-sm text-zinc-300">
            {t("diagSummary", {
              total: report.total,
              ok: report.ok,
              broken: report.broken.length,
              mismatch: report.mismatch.length,
              missing: report.missing.length,
            })}
          </div>

          {issueCount === 0 ? (
            <p className="glass-panel px-5 py-6 text-center text-sm text-teal-300">
              {t("diagAllGood")}
            </p>
          ) : (
            <div className="glass-panel flex flex-col gap-2 px-5 py-4">
              <p className="text-sm font-semibold text-amber-300">{t("diagIssues")}</p>
              {(["broken", "mismatch", "missing"] as const).map((k) =>
                report[k].map((line) => (
                  <p key={`${k}-${line}`} className="break-all font-mono text-xs text-zinc-400">
                    [{k}] {line}
                  </p>
                ))
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
