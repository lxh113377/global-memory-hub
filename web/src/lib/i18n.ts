import { createContext, useContext } from "react";

export type Lang = "zh" | "en";

type Dict = Record<string, string>;

const zh: Dict = {
  navHome: "总览",
  navSkills: "技能库",
  navDiag: "诊断",
  navSettings: "设置",
  appSub: "多端共享记忆 / 技能库控制台",
  connChecking: "检测中…",
  connOnline: "已连接",
  connOffline: "未检测到本地程序",
  needTokenBadge: "需要令牌",
  offlineTitle: "未检测到本地程序",
  offlineBody1:
    "请下载并运行焚诀 Global 本地伴随程序（监听 127.0.0.1:7799），保持其运行后点击重新探测即可恢复连接。",
  offlineBody2: "若浏览器拦截本地连接，请直接运行本地程序使用内置控制台。",
  tokenTitle: "需要握手令牌",
  tokenBody:
    "本地程序已运行但尚未携带令牌。粘贴 fenjue-agent serve 启动时打印的 token（64 位十六进制），或直接重新打开完整握手链接进入。",
  tokenPlaceholder: "粘贴 token（64 位十六进制）",
  tokenEnter: "进入控制台",
  statActive: "已接入端",
  statAbnormal: "异常端",
  memRoot: "记忆根：",
  skillRoot: "技能根：",
  stateErrorPrefix: "状态获取失败：",
  checkingText: "正在探测本地程序…",
  emptyText: "暂无平台数据，请确认本地程序已启动",
  batchEnable: "全部启用",
  batchDisable: "全部关闭",
  reprobe: "重新探测",
  batchDone: "批量{mode}完成：成功 {ok} 端，失败 {fail} 端",
  batchModeEnable: "启用",
  batchModeDisable: "关闭",
  cardOn: "挂载运行中",
  cardOff: "未挂载",
  cardEnable: "启用",
  cardDisable: "关闭",
  cardOk: "操作成功 · 备份 ID：{id} · 变更 {n} 项 · 可还原",
  cardOkNoBackup: "无",
  cardDetail: "挂载明细（{n} 项）",
  cardNone: "无挂载记录",
  cardInject: "注入项",
  cardSync: "手动同步",
  cardSyncMirrorOnly: "镜像端才会显示同步按钮",
  kindLink: "符号链接",
  kindMirror: "物理镜像，关闭=停止同步",
  kindPerSkill: "逐技能链接树",
  skillsTitle: "技能库",
  skillsSub: "统一库 skills 根下的全部技能（实时读取，来自 SKILL.md）。",
  skillsSearch: "搜索技能…",
  skillsCount: "{n} 个技能",
  skillsEmpty: "统一库还没有技能。启动一次 fenjue-agent 会自动写入通用技能包。",
  skillsUnavailable: "技能根不可用：{note}",
  thName: "名称",
  thDesc: "描述",
  thId: "ID",
  diagTitle: "诊断",
  diagSub: "对全部 9 端的挂载与注入点做四态体检（OK / BROKEN / MISMATCH / MISSING）。",
  diagRun: "运行诊断",
  diagRunning: "检测中…",
  diagSummary: "{total} 项检查：{ok} 正常 / {broken} 断链 / {mismatch} 失配 / {missing} 缺失",
  diagAllGood: "全部正常，未发现异常项。",
  diagIssues: "异常明细",
  setTitle: "设置",
  setSub: "库根、语言与程序信息。所有数据只存在你本机。",
  setMemRoot: "记忆根路径",
  setSkillRoot: "技能根路径",
  setSave: "保存",
  setSaving: "保存中…",
  setSaved: "已保存。重启 fenjue-agent 后生效。",
  setSaveErr: "保存失败：",
  setRootsHint: "留空表示不修改对应项。改动会写入 ~/.fenjue/state/roots.json。",
  setLang: "界面语言",
  setAbout: "关于",
  setVer: "程序版本",
  setSafety: "数据只存在你本机 ~/.fenjue/，不上传任何内容。",
  setGithub: "GitHub 仓库",
  fetchErr: "请求失败：",
  unknownErr: "未知错误",
  presetTitle: "命名预设",
  presetSub: "预设来自 platforms.json 的 presets 段，成员由你决定，程序不内置分组。",
  presetMembers: "成员 {n} 端",
  presetDryRun: "预演",
  presetEnable: "启用",
  presetDisable: "关闭",
  presetApplying: "执行中…",
  presetEmpty: "未配置预设。在 platforms.json 加一个 presets 段即可启用批量启停。",
  presetDryRunResult: "预演结果（未写入任何文件）：{n} 个成员将被{action}",
  presetOk: "{preset} 完成：{ok} 成功 / {fail} 失败",
  presetFail: "{preset} 有成员失败，逐端结果见下",
  presetSoftDisable: "批量关闭为软关闭（可一键还原）",
};

const en: Dict = {
  navHome: "Overview",
  navSkills: "Skills",
  navDiag: "Diagnostics",
  navSettings: "Settings",
  appSub: "Shared memory & skills console",
  connChecking: "Checking…",
  connOnline: "Connected",
  connOffline: "Local agent not detected",
  needTokenBadge: "Token required",
  offlineTitle: "Local agent not detected",
  offlineBody1:
    "Download and run the fenjue companion agent (listens on 127.0.0.1:7799), keep it running, then hit Re-probe.",
  offlineBody2:
    "If your browser blocks the local connection, use the built-in console printed by the agent itself.",
  tokenTitle: "Handshake token required",
  tokenBody:
    "The agent is running but no token is present. Paste the token printed by fenjue-agent serve (64 hex chars), or reopen the full handshake link.",
  tokenPlaceholder: "Paste token (64 hex chars)",
  tokenEnter: "Enter console",
  statActive: "Active",
  statAbnormal: "Abnormal",
  memRoot: "Memory root: ",
  skillRoot: "Skills root: ",
  stateErrorPrefix: "Failed to fetch state: ",
  checkingText: "Probing local agent…",
  emptyText: "No platform data; make sure the agent is running.",
  batchEnable: "Enable all",
  batchDisable: "Disable all",
  reprobe: "Re-probe",
  batchDone: "Batch {mode} done: {ok} ok, {fail} failed",
  batchModeEnable: "enable",
  batchModeDisable: "disable",
  cardOn: "Mounted & running",
  cardOff: "Not mounted",
  cardEnable: "Enable",
  cardDisable: "Disable",
  cardOk: "OK · backup {id} · {n} changes · restorable",
  cardOkNoBackup: "none",
  cardDetail: "Mount details ({n})",
  cardNone: "No mounts",
  cardInject: "Injection",
  cardSync: "Sync now",
  cardSyncMirrorOnly: "Sync appears on mirror platforms only",
  kindLink: "symlink/junction",
  kindMirror: "physical mirror; disable = stop syncing",
  kindPerSkill: "per-skill link tree",
  skillsTitle: "Skills",
  skillsSub: "All skills under the unified skills root (read live from SKILL.md files).",
  skillsSearch: "Search skills…",
  skillsCount: "{n} skills",
  skillsEmpty: "The library has no skills yet. Running fenjue-agent once seeds the base pack.",
  skillsUnavailable: "Skills root unavailable: {note}",
  thName: "Name",
  thDesc: "Description",
  thId: "ID",
  diagTitle: "Diagnostics",
  diagSub: "Four-state health check (OK / BROKEN / MISMATCH / MISSING) across all platforms.",
  diagRun: "Run diagnostics",
  diagRunning: "Checking…",
  diagSummary:
    "{total} checks: {ok} ok / {broken} broken / {mismatch} mismatch / {missing} missing",
  diagAllGood: "All good, no issues found.",
  diagIssues: "Issues",
  setTitle: "Settings",
  setSub: "Library roots, language and agent info. All data stays on this machine.",
  setMemRoot: "Memory root path",
  setSkillRoot: "Skills root path",
  setSave: "Save",
  setSaving: "Saving…",
  setSaved: "Saved. Restart fenjue-agent to take effect.",
  setSaveErr: "Save failed: ",
  setRootsHint: "Leave a field empty to keep it unchanged. Saved to ~/.fenjue/state/roots.json.",
  setLang: "UI language",
  setAbout: "About",
  setVer: "Agent version",
  setSafety: "All data stays local under ~/.fenjue/ — nothing is uploaded.",
  setGithub: "GitHub repository",
  fetchErr: "Request failed: ",
  unknownErr: "Unknown error",
  presetTitle: "Presets",
  presetSub: "Presets come from the presets section of platforms.json; members are yours to choose, the program hard-codes no grouping.",
  presetMembers: "{n} ends",
  presetDryRun: "Dry run",
  presetEnable: "Enable",
  presetDisable: "Disable",
  presetApplying: "Working…",
  presetEmpty: "No presets configured. Add a presets section to platforms.json to enable batch toggling.",
  presetDryRunResult: "Dry run (nothing was written): {n} member(s) would be {action}",
  presetOk: "{preset} done: {ok} ok / {fail} failed",
  presetFail: "{preset} had failing members, see per-end results below",
  presetSoftDisable: "Batch disable is a soft close (one-click restore)",
};

const dicts: Record<Lang, Dict> = { zh, en };

export type Key = keyof typeof zh;

export interface I18n {
  lang: Lang;
  t: (key: Key, vars?: Record<string, string | number>) => string;
}

export const I18nContext = createContext<I18n>({
  lang: "zh",
  t: (k) => zh[k],
});

export function useT(): I18n {
  return useContext(I18nContext);
}

export function makeI18n(lang: Lang): I18n {
  return {
    lang,
    t: (key, vars) => {
      let s = dicts[lang][key] ?? zh[key] ?? key;
      if (vars) {
        for (const [k, v] of Object.entries(vars)) {
          s = s.split(`{${k}}`).join(String(v));
        }
      }
      return s;
    },
  };
}

export function initialLang(): Lang {
  try {
    return window.localStorage.getItem("fenjue_lang") === "en" ? "en" : "zh";
  } catch {
    return "zh";
  }
}
