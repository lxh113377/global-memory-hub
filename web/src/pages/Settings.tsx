import { useEffect, useState } from "react";
import { setRoots, errText } from "../lib/api";
import { useT } from "../lib/i18n";
import type { AppState } from "../lib/types";

interface SettingsProps {
  state: AppState | null;
  healthVersion: string | null;
  lang: "zh" | "en";
  onLangChange: (lang: "zh" | "en") => void;
}

export default function Settings({ state, healthVersion, lang, onLangChange }: SettingsProps) {
  const { t } = useT();
  const [mem, setMem] = useState("");
  const [skl, setSkl] = useState("");
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);

  useEffect(() => {
    setMem("");
    setSkl("");
  }, [state?.roots.memory, state?.roots.skills]);

  const save = async () => {
    setSaving(true);
    setMsg(null);
    try {
      const res = await setRoots(mem.trim(), skl.trim());
      setMsg({ ok: true, text: t("setSaved") });
      setMem("");
      setSkl("");
      void res;
    } catch (e) {
      setMsg({ ok: false, text: t("setSaveErr") + errText(e) });
    } finally {
      setSaving(false);
    }
  };

  const canSave = (mem.trim() !== "" || skl.trim() !== "") && !saving;

  return (
    <div className="flex flex-col gap-4">
      <div className="glass-panel px-5 py-4">
        <h2 className="text-lg font-bold brand-gradient-text">{t("setTitle")}</h2>
        <p className="mt-1 text-xs leading-relaxed text-zinc-400">{t("setSub")}</p>
      </div>

      <div className="glass-panel flex flex-col gap-3 px-5 py-4">
        <label className="flex flex-col gap-1.5">
          <span className="text-xs text-zinc-400">
            {t("setMemRoot")} <span className="text-zinc-600">({state?.roots.memory ?? "--"})</span>
          </span>
          <input
            value={mem}
            onChange={(e) => setMem(e.target.value)}
            placeholder={state?.roots.memory ?? ""}
            className="rounded-lg border border-zinc-700/60 bg-black/40 px-3 py-2 font-mono text-xs text-zinc-200 outline-none focus:border-amber-400/60"
          />
        </label>
        <label className="flex flex-col gap-1.5">
          <span className="text-xs text-zinc-400">
            {t("setSkillRoot")} <span className="text-zinc-600">({state?.roots.skills ?? "--"})</span>
          </span>
          <input
            value={skl}
            onChange={(e) => setSkl(e.target.value)}
            placeholder={state?.roots.skills ?? ""}
            className="rounded-lg border border-zinc-700/60 bg-black/40 px-3 py-2 font-mono text-xs text-zinc-200 outline-none focus:border-amber-400/60"
          />
        </label>
        <p className="text-[11px] text-zinc-600">{t("setRootsHint")}</p>
        <div className="flex items-center gap-3">
          <button type="button" className="btn-primary" disabled={!canSave} onClick={() => void save()}>
            {saving ? t("setSaving") : t("setSave")}
          </button>
          {msg && (
            <span className={`text-xs ${msg.ok ? "text-teal-300" : "text-red-300"}`}>{msg.text}</span>
          )}
        </div>
      </div>

      <div className="glass-panel flex flex-col gap-3 px-5 py-4">
        <p className="text-sm font-semibold text-zinc-200">{t("setLang")}</p>
        <div className="flex gap-2">
          {(["zh", "en"] as const).map((l) => (
            <button
              key={l}
              type="button"
              onClick={() => onLangChange(l)}
              className={`rounded-lg border px-4 py-1.5 text-xs transition-colors ${
                lang === l
                  ? "border-amber-400/60 bg-amber-400/10 text-amber-300"
                  : "border-zinc-700/60 text-zinc-400 hover:text-zinc-200"
              }`}
            >
              {l === "zh" ? "中文" : "English"}
            </button>
          ))}
        </div>
      </div>

      <div className="glass-panel flex flex-col gap-2 px-5 py-4">
        <p className="text-sm font-semibold text-zinc-200">{t("setAbout")}</p>
        <p className="text-xs text-zinc-500">
          {t("setVer")}: <span className="font-mono text-amber-300/90">{healthVersion ?? "--"}</span>
        </p>
        <p className="text-xs leading-relaxed text-zinc-500">{t("setSafety")}</p>
        <a
          href="https://github.com/lxh113377/global-memory-hub"
          target="_blank"
          rel="noopener noreferrer"
          className="w-fit text-xs text-amber-300 underline decoration-amber-400/40 hover:decoration-amber-300"
        >
          {t("setGithub")}
        </a>
      </div>
    </div>
  );
}
