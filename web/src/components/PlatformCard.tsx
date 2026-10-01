import { useState } from "react";
import type { MountEntry, MutationResult, PlatformInfo } from "../lib/types";
import StatusBadge from "./StatusBadge";

const KIND_NOTE: Record<MountEntry["kind"], string> = {
  link: "符号链接",
  mirror: "物理镜像，关闭=停止同步",
  "per-skill": "逐技能链接树",
};

interface PlatformCardProps {
  platform: PlatformInfo;
  busy: boolean;
  result: MutationResult | null;
  error: string | null;
  onToggle: (platform: PlatformInfo, next: boolean) => void;
}

export default function PlatformCard({
  platform,
  busy,
  result,
  error,
  onToggle,
}: PlatformCardProps) {
  const [detailOpen, setDetailOpen] = useState(false);
  const isOn = platform.status === "OK";
  const isBeta = platform.support === "beta";

  return (
    <section className="glass-panel flex flex-col gap-3 p-5 transition-all duration-200 hover:-translate-y-1 hover:border-amber-400/40 hover:shadow-[0_10px_32px_rgba(255,107,26,0.18)]">
      <div className="flex items-start justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="text-base font-semibold text-zinc-100">{platform.label}</h2>
          <span className="rounded bg-zinc-800/80 px-1.5 py-0.5 text-[10px] uppercase tracking-wider text-zinc-400">
            {platform.id}
          </span>
          {isBeta && (
            <span className="rounded-full border border-amber-400/50 bg-amber-400/10 px-2 py-0.5 text-[10px] font-semibold text-amber-300">
              BETA
            </span>
          )}
        </div>
        <StatusBadge status={platform.status} />
      </div>

      <div className="flex items-center justify-between gap-3">
        <span className="text-xs text-zinc-500">
          {isOn ? "挂载运行中" : "未挂载"}
        </span>
        <button
          type="button"
          role="switch"
          aria-checked={isOn}
          aria-label={`${isOn ? "关闭" : "启用"} ${platform.label}`}
          disabled={busy}
          onClick={() => onToggle(platform, !isOn)}
          className={`relative h-8 w-14 shrink-0 rounded-full border transition-colors duration-300 ${
            isOn
              ? "border-amber-400/50 bg-gradient-to-r from-[#d4a017] to-[#ff6b1a] shadow-[0_0_14px_rgba(255,107,26,0.4)]"
              : "border-zinc-600/60 bg-zinc-800"
          } ${busy ? "cursor-not-allowed opacity-60" : "cursor-pointer"}`}
        >
          <span
            className={`absolute top-1/2 flex h-6 w-6 -translate-y-1/2 items-center justify-center rounded-full bg-white shadow-md transition-all duration-300 ${
              isOn
                ? "left-[calc(100%-1.75rem)] shadow-[0_0_10px_rgba(212,160,23,0.9)]"
                : "left-1"
            }`}
          >
            {busy && (
              <span className="h-3 w-3 animate-spin rounded-full border-2 border-zinc-400 border-t-transparent" />
            )}
          </span>
        </button>
      </div>

      {error && (
        <p className="rounded-lg border border-red-400/30 bg-red-400/10 px-3 py-2 text-xs leading-relaxed text-red-300">
          {error}
        </p>
      )}

      {result && result.ok && (
        <p className="rounded-lg border border-teal-400/25 bg-teal-400/10 px-3 py-2 text-xs leading-relaxed text-teal-200">
          操作成功 · 备份 ID：{result.backupId || "无"} · 变更 {result.changes.length} 项 · 可还原
        </p>
      )}

      <button
        type="button"
        onClick={() => setDetailOpen((v) => !v)}
        className="flex items-center gap-1.5 self-start text-xs text-zinc-400 transition-colors hover:text-amber-300"
        aria-expanded={detailOpen}
      >
        <span
          className={`inline-block text-[10px] transition-transform duration-200 ${
            detailOpen ? "rotate-90" : ""
          }`}
        >
          ▶
        </span>
        挂载明细（{platform.mounts.length} 项）
      </button>

      {detailOpen && (
        <div className="flex flex-col gap-2 rounded-lg border border-zinc-700/50 bg-black/25 p-3">
          {platform.mounts.length === 0 && (
            <p className="text-xs text-zinc-500">无挂载记录</p>
          )}
          {platform.mounts.map((m, i) => (
            <div key={`${m.from}-${m.to}-${i}`} className="flex flex-col gap-0.5">
              <div className="flex flex-wrap items-center gap-1.5 text-[11px] leading-relaxed">
                <span className="text-zinc-300">{m.from}</span>
                <span className="text-amber-400">→</span>
                <span className="break-all text-zinc-400">{m.to}</span>
                <StatusBadge status={m.status} />
              </div>
              <span className="text-[10px] text-zinc-500">[{KIND_NOTE[m.kind]}]</span>
            </div>
          ))}
          {platform.inject.length > 0 && (
            <div className="mt-1 flex flex-col gap-1.5 border-t border-zinc-700/50 pt-2">
              <p className="text-[11px] font-medium text-zinc-400">注入项</p>
              {platform.inject.map((inj) => (
                <div
                  key={inj.path}
                  className="flex flex-wrap items-center gap-2 text-[11px]"
                >
                  <span className="break-all text-zinc-400">{inj.path}</span>
                  <StatusBadge status={inj.status} />
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </section>
  );
}
