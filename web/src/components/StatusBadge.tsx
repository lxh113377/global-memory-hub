import type { ItemStatus } from "../lib/types";

interface StatusMeta {
  label: string;
  text: string;
  ring: string;
  dot: string;
}

const STATUS_META: Record<ItemStatus, StatusMeta> = {
  OK: {
    label: "正常",
    text: "text-teal-300",
    ring: "border-teal-400/30 bg-teal-400/10",
    dot: "bg-teal-400 text-teal-400",
  },
  BROKEN: {
    label: "已损坏",
    text: "text-amber-300",
    ring: "border-amber-400/30 bg-amber-400/10",
    dot: "bg-amber-400 text-amber-400",
  },
  MISMATCH: {
    label: "不一致",
    text: "text-red-300",
    ring: "border-red-400/30 bg-red-400/10",
    dot: "bg-red-400 text-red-400",
  },
  MISSING: {
    label: "缺失",
    text: "text-red-300",
    ring: "border-red-400/30 bg-red-400/10",
    dot: "bg-red-400 text-red-400",
  },
  SKIP: {
    label: "未安装",
    text: "text-zinc-400",
    ring: "border-zinc-500/30 bg-zinc-500/10",
    dot: "bg-zinc-500 text-zinc-500",
  },
};

export default function StatusBadge({ status }: { status: ItemStatus }) {
  const meta = STATUS_META[status];
  return (
    <span
      className={`inline-flex shrink-0 items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium ${meta.ring} ${meta.text}`}
    >
      <span className={`h-1.5 w-1.5 rounded-full animate-breath ${meta.dot}`} />
      {meta.label}
    </span>
  );
}
