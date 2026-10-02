import { useEffect, useMemo, useState } from "react";
import { fetchSkills, errText } from "../lib/api";
import type { SkillsResponse } from "../lib/types";
import { useT } from "../lib/i18n";

export default function Skills() {
  const { t } = useT();
  const [data, setData] = useState<SkillsResponse | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [q, setQ] = useState("");

  useEffect(() => {
    fetchSkills()
      .then(setData)
      .catch((e) => setErr(errText(e)));
  }, []);

  const list = useMemo(() => {
    const all = data?.skills ?? [];
    const needle = q.trim().toLowerCase();
    if (!needle) return all;
    return all.filter((s) =>
      `${s.id} ${s.name} ${s.description}`.toLowerCase().includes(needle)
    );
  }, [data, q]);

  return (
    <div className="flex flex-col gap-4">
      <div className="glass-panel px-5 py-4">
        <h2 className="text-lg font-bold brand-gradient-text">{t("skillsTitle")}</h2>
        <p className="mt-1 text-xs leading-relaxed text-zinc-400">{t("skillsSub")}</p>
      </div>

      {err && (
        <p className="rounded-lg border border-red-400/25 bg-red-400/[0.07] px-4 py-2.5 text-xs text-red-300">
          {t("fetchErr")}
          {err}
        </p>
      )}

      <div className="flex flex-wrap items-center gap-3">
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder={t("skillsSearch")}
          className="w-full max-w-sm rounded-lg border border-zinc-700/60 bg-black/40 px-3 py-2 text-sm text-zinc-200 outline-none focus:border-amber-400/60"
        />
        {data && (
          <span className="text-xs text-zinc-500">
            {t("skillsCount", { n: list.length })}
          </span>
        )}
      </div>

      {data?.note && (
        <p className="rounded-lg border border-amber-400/25 bg-amber-400/[0.07] px-4 py-2.5 text-xs text-amber-300">
          {t("skillsUnavailable", { note: data.note })}
        </p>
      )}

      {data && list.length === 0 && !data.note && (
        <div className="glass-panel px-6 py-12 text-center text-sm text-zinc-400">
          {t("skillsEmpty")}
        </div>
      )}

      {list.length > 0 && (
        <div className="glass-panel overflow-x-auto px-2 py-2">
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="text-left text-xs text-zinc-500">
                <th className="px-3 py-2 font-medium">{t("thName")}</th>
                <th className="px-3 py-2 font-medium">{t("thId")}</th>
                <th className="px-3 py-2 font-medium">{t("thDesc")}</th>
              </tr>
            </thead>
            <tbody>
              {list.map((s) => (
                <tr key={s.id} className="border-t border-zinc-800/70 align-top">
                  <td className="px-3 py-2.5 font-medium text-zinc-200">{s.name}</td>
                  <td className="px-3 py-2.5 font-mono text-xs text-zinc-500">{s.id}</td>
                  <td className="px-3 py-2.5 text-xs leading-relaxed text-zinc-400">
                    {s.description || "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
