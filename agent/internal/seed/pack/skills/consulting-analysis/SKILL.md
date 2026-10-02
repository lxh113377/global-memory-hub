---
name: consulting-analysis
version: 1.16.0
description: 研究报告/咨询分析:市场分析、消费者洞察、品牌与财务分析报告。Use this skill when the user requests to generate, create, or write professional research reports including but not limited to market analysis, consumer insights, brand analysis, financial analysis, industry research, competitive intelligence, technical/engineering project benchmarking (技术/工程项目对标), investment due diligence, or any consulting-grade analytical report. This skill operates in two phases — (1) generating a structured analysis framework with chapter skeleton, data query requirements, and analysis logic, and (2) after data collection by other skills, producing the final consulting-grade report with structured narratives, embedded charts, and strategic insights.
---

# Professional Research Report Skill

## Overview

This skill produces professional, consulting-grade research reports in Markdown format, covering domains such as **market analysis, consumer insights, brand strategy, financial analysis, industry research, competitive intelligence, investment research, and macroeconomic analysis**. It operates across two distinct phases:

1. **Phase 1 — Analysis Framework Generation**: Given a research subject, produce a rigorous analysis framework including chapter skeleton, per-chapter data requirements, analysis logic, and visualization plan.
2. **Phase 2 — Report Generation**: After data has been collected by other skills, synthesize all inputs into a final polished report.

The output adheres to McKinsey/BCG consulting voice standards. The report language follows the `output_locale` setting (default: `zh_CN` for Chinese).

## Data Authenticity Protocol

**Strict Adherence Rule**: All data presented in the report and visualized in charts MUST be derived directly from the provided **Data Summary** or **External Search Findings**.
- **NO Hallucinations**: Do not invent, estimate, or simulate data. If data is missing, state "Data not available" rather than fabricating numbers.
- **Traceable Sources**: Every major claim and chart must be traceable back to the input data package.

## Core Capabilities

- **Design analysis frameworks** from scratch given only a research subject and scope
- Transform raw data into structured, high-depth research reports
- Follow the **"Visual Anchor → Data Contrast → Integrated Analysis"** flow per sub-chapter
- Produce insights following the **"Data → User Psychology → Strategy Implication"** chain
- Embed pre-generated charts and construct comparison tables
- Generate inline citations formatted per **GB/T 7714-2015** standards
- Output reports in the language specified by `output_locale` with professional consulting tone
- Adapt analytical depth and structure to domain (marketing, finance, industry, etc.)

## When to Use This Skill

**Always load this skill when:**

- User asks for a market analysis, consumer insight report, financial analysis, industry research, or any consulting-grade analytical report
- User asks for a **technical/engineering project benchmarking** (技术项目对标/开源项目对比分析): treat the subject system as the analyzed entity and peer projects as comparison targets — apply the Competitive Intelligence (Benchmarking) framework; typical chapter structure = 七维逐项对比 / 模块映射 / 差距汇总 / 可借鉴清单 / 优先级排序
- User provides a research subject and needs a structured analysis framework before data collection
- User provides data summaries, analysis frameworks, or chart files to be synthesized into a report
- User needs a professional consulting-style research report
- The task involves transforming research findings into structured strategic narratives

---

# Phase 1: Analysis Framework Generation

## Purpose

Given a **research subject** (e.g., "Gen-Z Skincare Market Analysis", "NEV Industry Competitive Landscape", "Brand X Consumer Profiling"), produce a complete **analysis framework** that serves as the blueprint for downstream data collection and final report generation.

## Phase 1 Inputs

| Input | Description | Required |
|-------|-------------|----------|
| **Research Subject** | The topic or question to be analyzed | Yes |
| **Scope / Constraints** | Geographic scope, time range, industry segment, target audience, etc. | Optional |
| **Specific Angles** | Any particular angles or hypotheses the user wants explored | Optional |
| **Domain** | The analytical domain: market, finance, industry, brand, consumer, investment, etc. | Inferred |

## Phase 1 Workflow

### Step 1.05: Metric Credibility Pre-Audit (度量可信度前置审计 · R280 强制)

**Trigger**: 任何「对标 / 竞品比较 / 自评分数 / 验收结论」类研究 —— 即报告将出现**数字**的场合。
**Purpose**: 根治「引用了过期或不同源的数字，导致整份报告结论错误却全部有据可查」这一类失效（焚诀 2026-09-24 第三轮对标实证，见 `焚诀/reports/2026-09-24_GitHub开源项目全景对标分析.md` §4）。

在写任何分析框架**之前**，先对待用数字逐条过下表判据（**条数以本表实际序号为准，禁手抄计数** —— M5 行实证过「六条问」这一写法落后三轮，被改前的本行写「四问」属同族）：任一不过即降级为「待核实」并重新取证：

| # | 前置判据 | 不合格形态（均实测发生过） |
|---|---|---|
| M1 | **当次实跑** | 数字来自历史快照产物而门禁只验 mtime（实测：派生评分 JSON 与正文差 7.4 分，两者 mtime 均在 7 天窗口内被判「新鲜」）|
| M2 | **同一份集** | 被引用的分数与流水线实际跑的输入不是同一份（实测：账面「盲测 93.1%/291 条」，CI 实跑是另一份 10 条集、Top-1 75.0%）|
| M3 | **口径同分母** | 分子分母来自不同刻度（实测：150 制档案分数被渲染成「x/200」并配「99.6%」）；或同一指标并存两个分母（167 vs 162）|
| M4 | **有判别力** | 阈值低到不可能失败（实测：门檻 50% 对 88~93% 实测）、或指标已饱和（Hit@5=1.0 ⇒ 题无区分度而非能力强）；**或退出码只分两挡**——把「非 0」一律读成「环境未验」，判据自身硬崩（实测 `rc=0xC0000409`、stdout 全空）就会混进 ENV 类拿通行证，红照报但**性质变了没人看得见**（实测：心屿对标 r35 自建两挡分类当天即被自己的输出证伪）。正解＝**三挡**：`1=判红 / 2=环境未验 / 其余=CRASH 点名（带 0x 十六进制）`，并要求合成四态套件双向验（PASS 不进红名单 / 崩溃不冒充环境 / 有红必 rc=1）。同族测量错：`python x.py | tail; echo $?` 取的是 `tail` 的 rc ⇒ 崩溃被读成 rc=0；判 rc 一律重定向到文件后取 `$?`|
| M5 | **粒度·等级·环境同构** | 逐条问（①-⑨；条数以本节实际序号为准，**不手抄计数**——"六条问"这个写法曾落后三轮）：① 断言的**粒度**配得上交付物吗——类别分是加权平均，单项彻底失败会被平均掉（实测：`color-contrast` 归零后 accessibility 仍 0.96，落在 0.95 error 门禁内，缺陷藏了六轮）；② 断言的**等级**真能拦吗——`warn` 级不改退出码，等于没有断言（实测：SEO warn 让 0.63 打日志绿了六轮）；③ **测量环境与目标用户一致**吗——主题/locale/设备差异会让整条代码路径从不被执行（实测：中文首访整页重排 CLS 0.262，只在 zh 浏览器下发生）；④ **判据命令自身有正样本吗**——"0 命中"要先证明这条命令能搜到已知存在的东西（实测：`grep` 在 `node_modules/` 下对任何已知串都返回 0，据此写下的"该功能不存在"是假的）。④ 还带一条反向教训：**注入的反例必须先单独证明「它真的会失败」**（2026-09-25 iCAN 对标第 17 轮实测：为验「迁移失败必须整体回滚」，注入 `ALTER TABLE t ADD COLUMN x NOT_A_REAL_TYPE X` 当失败语句，而 SQLite 不校验列类型名 ⇒ 那条 DDL 合法执行，测试拿到 `raised=False`，「回滚」这件事从头到尾根本没被测到——比没有判据更糟，它给出一个绿色的假安全感；换成真语法错 `INTEGER REFERENCES(;)` 才得到有效对照）。⇒ 写反例前先单独跑一次那条输入、确认它确实失败，并把「它为什么会失败」写进注释。同族形态：拿假异常 / 假错误码 / 不存在的文件名当反例，测的是夹具不是判据。；⑤ **能力判定来自观测还是来自环境字符串**——按 `location.protocol`/UA/版本号写死"不可用"，是拿一次幸运观察冒充普适规律（实测：本机 Chrome 开着文件访问权限，`file://` 下 `fetch` 与全文检索**都能成功**，协议猜测把上一轮刚交付的功能关掉了；正解=失败一次即闭锁，两种环境行为都对）。⑤ 还带一条反向教训：**"我改不动它"证明不了"它不可用"**——把主线程 `window.fetch` 整体替换成 reject 后 Pagefind 照常出结果、fetch 计数 0（它不走主线程 fetch），要证不可用得让它**真的缺文件**；⑥ **同一判断是否只有一处实现**——判据写成两份时，改宽一份不会让另一份跟上，于是"本地全绿、CI 判红"（实测：`test_build.py` 的词典孤儿规则已扩到全部脚本，CI 另跑的 `check-i18n.py` 仍只扫 `main.js`，把三个新键判成孤儿并把发版提交跑红；正解=规则收进一处、另一方 importlib 加载后断言两侧集合相同，并对"退回旧规则"做变异体）。⑦ **分母本身被证明过吗**——覆盖率与「全零 / 全覆盖」类指标必须先给出「有效分母 vs 总数」：取数失败或被截断的样本要**踢出分母并逐条点名原因**（豁免必须带理由），否则「0/16 全零」分不清是「对手都没有」还是「我没数到」。实测：GitHub trees API 对超大仓会 `truncated=true` 静默少数；心屿对标 r27 一手核 lobehub `main` 递归树（20,740 个对象、`truncated=false`、`sw.js|service-worker.js|sw.ts` 零命中）才判 `pwa_offline=0/16` 为真零，并把核查固化成判据 `coverage_hits` + 恒等式 `usable + blind == 总数`（不成立即退出码 1）。⑧ **同一把尺对双方是否同构**——横向对比的观测面必须两侧一致。若匹配器**只看文件路径**（参照仓只能静态取树），那么"能力写在文件内容里"的一侧会被**静默少算**；此时**禁止把少算的项并入同一列**（等于偷偷给一侧换尺，与 M5⑥"两处实现改宽一份"同族），正确形态是**单列「盲区点名」**：在自证行旁印出本机可 grep 的证据，横向计数保持不动。实测：心屿对标 r30 的 `CAP_RULES` 使 self 行少报 `streaming`（5 个文件含 `text/event-stream`，文件名不含 sse）与 `e2e_browser`（20 个脚本 `import playwright`，路径不含 e2e）；处置为 `blind_spot_caps()` 单列点名 + 三向自证（有证据→必须点名 / 无证据→必须为空 / 匹配器已看见→不得重复计数）+ 三变异体（恒空 / 恒报 / 不去重）均被 `SELFTEST-FAIL` 抓住，覆盖率矩阵与 peers 计数零变化。另记一条**读表纪律**：报告正文须写明"某列为观测下限而非能力上限"，否则读者会把 `2/16` 读成"只有两家做了"，而真因是"只有两家的文件名看得见"。⑨ **这条"全零/独有"结论有第二条独立通道吗**——单种观测法支撑的零结论不可信，尤其当同一种方法**已被证明会在另一侧假阴性**时；必须另起一条**互不依赖的通道**复核，并对每条命中**先归因语境再计数**（同一个词可能有三种含义），取数失败的样本要记 `unverified` 且此时**禁止**打印全零。实测：心屿对标 r31 发现 `pwa_offline=0/16` 只有"文件名法"一条腿（r27 仅手工抽查 1/16 仓），遂加内容法通道读 description+README 四类归因 `app_shell / local_models_offline / ml_training_offline / none`，结果 `app_shell=0`（两法一致）、`local_models_offline=1`（Open-LLM-VTuber「run completely offline using local models」= 桌面自托管形态，**不是**网页壳）、`ml_training_offline=1`（hello-diana/MASCOT 的 **offline DPO 训练** = 纯误报源，若不归因就会凭空给对手加一格离线能力）、`unverified=0`。措辞随之下调为"没人用浏览器 app-shell 做离线"，**不再**写"没人做离线"。判据自证：四类样本各一 + 两条反向（训练语境不得判成离线壳 / 真 SW 语境必须判成离线壳）+ 零输入判 `none`，两个变异体（恒判 app_shell、恒判 none）均 `rc=1`。⑩ **逐字节 / 体积 / 哈希类主张，其字节前提被钉住了吗**（可复算性）——这类主张隐含一个从未写明、也从未被测的前提：**比较两侧是同一份字节**。git 仓库里它由 checkout 配置决定：`core.autocrlf=true` + `* text=auto` 下工作树是 CRLF 而 blob 是 LF，于是"本机逐字节相等"在另一台 `autocrlf=input` 的机器上**整体反向**，而所有引用该主张的判据（SHA256 对账、体积预算、线上==权威源）都只在一台机器上成立。正解形态＝把前提**钉进版本库**（`.gitattributes` 声明 `eol=lf` + 显式 `binary` 名单）+ **常驻判据**（工作树字节 == 仓库侧 blob 字节 + 分母闭合 + 跨检出复验），并配套一条写盘纪律：**落仓库文本禁 `Path.write_text` 文本模式**（Windows 把 `\n` 翻成 `\r\n`，一次写入就毁掉归一），且"写后读回比对"必须按字节 —— `read_text` 的 universal newlines 会把 CR 读成 `\n`，**复验步骤自身正好掩盖该缺陷**。还须防一类假守卫："两侧都归一再比"看不见**内容损失**（归一化把二进制里的 `0D0A` 当行尾剥掉，PNG/MP4 少 1–2 字节，两侧同样破坏 ⇒ 差值为零），所以判据必须含"未声明 binary 却出现 `0D0A` 即点名"的形状规则，两侧各配夹具。⑪ **能力位来自"配置存在"还是来自"行为闭环"**（configuration-without-behavior）——对标台账里 `dependabot`、`ci_workflows`、`SECURITY.md`、`.gitattributes` 这类**文件存在性**指标最容易给出假优势：配了不等于在用。实测形态：心屿 SoulIsle r37 对 16 仓扫 dependabot，**只有 1/16 配了**，本仓是其中之一（账面看是"领先 15 家"）；而同一个 dependabot 产出的 4 条升级 PR **挂了 2 天无人处理**（GitHub `open_issues_count` 还把 PR 计成 issue，进一步掩盖成"有 4 个 issue 待办"）⇒ 真实状态是"配置在册、行为未闭环"，与没配的对手相比优势远小于账面。规定动作：每个配置类能力位必须配一条**行为回执**才允许计入优势 —— dependabot ⇒ 近期被处理（merge/close）的 PR 数；CI ⇒ 目标 commit 上真跑过的 run 结论；Release ⇒ 资产数与被下载计数；`.gitattributes` ⇒ 工作树字节==blob 的常驻判据。取不到回执就写成「配置在册·行为未证」，**禁止与有回执的对手并列计数**（同 M5⑧"一侧换尺"同族）。⑫ **一条"这维做不到 / 不可比"的边界结论，是不是被当成了该维的测量结果？**（disclaimer-as-measurement）—— 边界声明只说明"比不了"，不说明"这一维里发生了什么"；同句拿不出**取证口径**（比对口径 + 样本数/命令 + 差异定位）时，该维必须留在「未测」，**禁止以声明结案**。实测：心屿 SoulIsle 对标 r35 写下「各参照仓公开性能数字仍无可比口径 ⇒ 只讲机制有无」，此后 **r36–r39 连续四轮再没碰过性能维**（每轮都拿这句当"已处理"），直到 r40 真去量才有数：本机 p95 15.9–26.2ms / 吞吐 2026.9 rps，CI 三次采样 p95 3.3–7.1ms / 吞吐 976–1179 rps——**同一环境跨 run 就差 2.15×**，这些事实全都被一句免责声明挡在视野外四轮之久。规定动作：报告里 `不可比 / 无可比口径 / 受限于 / 仅保证 / 无法做到` 类句子必须同句带实证标记（实测 · 逐文件 · 命令路径 · run 号 · 计数+单位 · sha/CRC · ls-remote）；拿不出实证就**改写成诚实缺口**（`未实测` + 下一步取证路径），两种形态不得混写。执行器＝常驻判据 `_test/disclaimer_forensics_lint.py`（8 类桩 + 恒绿守卫：漏报侧 3 / 不误伤侧 3 / 边界 2；**不拦** `未实测`/`❌` 这类诚实标注），已接进 CI 阻断链（套件 49→51，r40c 同轮再加 `ci_watch` 判定桩 → 52）；首跑就在自己的报告里点名那句 r35 遗留，补上取证指向后才转绿。M1-M4 全过仍可能整批漏判，M5 就是补这一层 |
| M6 | **材料是否追上被描述对象**（交付一致性回扫）| 报告/方案/README/截图描述的是"当前构建"，但材料常停在旧版本：实测心屿 SoulIsle 对标 r20–r31 连十一轮改产品，`application-plan.html` 却停在 09-24，对 `sw.js`/`Service Worker`/`PWA`/`逐字`/`流式`/`设置面板` **命中数全为 0** ——三项已建成能力在评委唯一会读的文件里完全不可见。这是「文档写了 ≠ 磁盘有」的**反向形态**（磁盘有、材料没写），同样毁结论但更难发现，因为没有任何告警。正解：动笔前对交付物跑一次**能力关键词命中数矩阵**（每个能力名 grep 计数，0 命中即列为待补），补完后重渲染并**复算页数/图数**（页数贴官方上限时必须显式写"再加内容须先精简"） |
| M7 | **产品承诺的语言，每个可见面是否兑现**（locale-promise realization）| 双语/多语产品最常见的漏配是「正文翻译了、外壳没翻」：实测孝心联对标第 30 轮——切到中文，正文 `data-i18n` 全部翻译，但**浏览器标签页标题 `<title>`/`document.title` 仍是英文**（`<title>` 不在 i18n 体系内、`applyTranslations` 从不碰它）。对以该语言为主人群的产品（中文长者站），标签页/书签/历史都是英文＝承诺没在它的语言里兑现；且**只断言静态 HTML 看不出来**，必须真浏览器切换语言观测（呼应 M5③「测量环境=目标用户」）。正解：对被承诺的**每个语言**，逐个可见面（标题 / meta / 关键 UI 标签）核是否兑现；少数确有理由保留源语言的面须有**显式 allow-list + 理由**，否则=静默漏配。判据必须双向（R-ENUM）：漏报侧注入「正文译了标题没译」要命中；误报侧三样合法输入（各语言已全部兑现 / 显式 allow-list 例外页 / 单语产品）要不命中；空页面清单或无可判 i18n 证据记 UNVERIFIED，**不得静默 PASS**（R247）。 |

**输出要求**：报告须含「取证纪律」声明，逐条标注 `✅当次实测（附命令/路径/时刻）｜⚠️引用他处（注明原取证日）｜❌未核实（不得进入结论）`。
**禁令**：禁止把「文档写了」当「磁盘有」，也禁止反过来「磁盘有、交付材料没写」（M6 回扫）；禁止把「门禁 PASS」当「判据有效」（自校验装置同样会假通过，须抽样人工复验）；禁止用调参掩盖机制缺陷（阈值必须给出两侧边界值实测依据）。

### Step 1.1: Understand the Research Subject

- Parse the research subject to identify the **core entity** (market, brand, product, industry, consumer segment, financial instrument, etc.)
- Identify the **analytical domain** (marketing, finance, industry, competitive, consumer, investment, macro, etc.)
- Determine the **natural analytical dimensions** based on domain:

| Domain | Typical Dimensions |
|--------|--------------------|
| Market Analysis | Market size, growth trends, market segmentation, growth drivers, competitive landscape, consumer profiling |
| Brand Analysis | Brand positioning, market share, consumer perception, marketing strategy, competitor comparison |
| Consumer Insights | Demographic profiling, purchase behavior, decision journey, pain points, scenario analysis |
| Financial Analysis | Macro environment, industry trends, company fundamentals, financial metrics, valuation, risk assessment |
| Industry Research | Value chain analysis, market size, competitive landscape, policy environment, technology trends, entry barriers |
| Investment Due Diligence | Business model, financial health, management assessment, market opportunity, risk factors, exit pathways |
| Competitive Intelligence | Competitor identification, strategic comparison, SWOT analysis, differentiated positioning, market dynamics |

### Step 1.2: Select Analysis Frameworks & Models

Based on the identified domain and research subject, select **one or more** professional analysis frameworks to structure the reasoning in each chapter. The chosen frameworks guide the **Analysis Logic** in the chapter skeleton (Step 1.3).

#### Strategic & Environmental Analysis

| Framework | Description | Best For |
|-----------|-------------|----------|
| **SWOT Analysis** | Strengths, Weaknesses, Opportunities, Threats | Brand assessment, competitive positioning, strategic planning |
| **PEST / PESTEL Analysis** | Political, Economic, Social, Technological (+ Environmental, Legal) | Macro-environment scanning, market entry assessment, policy impact analysis |
| **Porter's Five Forces** | Supplier bargaining power, buyer bargaining power, threat of new entrants, threat of substitutes, industry rivalry | Industry competitive landscape, entry barrier assessment, profit margin analysis |
| **Porter's Diamond Model** | Factor conditions, demand conditions, related industries, firm strategy & structure | National/regional competitive advantage analysis |
| **VRIO Analysis** | Value, Rarity, Imitability, Organization | Core competency assessment, resource advantage analysis |

#### Market & Growth Analysis

| Framework | Description | Best For |
|-----------|-------------|----------|
| **STP Analysis** | Segmentation, Targeting, Positioning | Market segmentation, target market selection, brand positioning |
| **BCG Matrix (Growth-Share Matrix)** | Stars, Cash Cows, Question Marks, Dogs | Product portfolio management, resource allocation decisions |
| **Ansoff Matrix** | Market penetration, market development, product development, diversification | Growth strategy selection |
| **Product Life Cycle (PLC)** | Introduction, growth, maturity, decline | Product strategy formulation, market timing decisions |
| **TAM-SAM-SOM** | Total / Serviceable / Obtainable Market | Market sizing, opportunity quantification |
| **Technology Adoption Lifecycle** | Innovators → Early Adopters → Early Majority → Late Majority → Laggards | Emerging technology/category penetration analysis |

#### Consumer & Behavioral Analysis

| Framework | Description | Best For |
|-----------|-------------|----------|
| **Consumer Decision Journey** | Awareness → Consideration → Evaluation → Purchase → Loyalty | Consumer behavior path mapping, touchpoint optimization |
| **AARRR Funnel (Pirate Metrics)** | Acquisition, Activation, Retention, Revenue, Referral | User growth analysis, conversion rate optimization |
| **RFM Model** | Recency, Frequency, Monetary | Customer value segmentation, precision marketing |
| **Maslow's Hierarchy of Needs** | Physiological → Safety → Social → Esteem → Self-actualization | Consumer psychology analysis, product value proposition |
| **Jobs-to-be-Done (JTBD)** | The "job" a user needs to accomplish in a specific context | Demand insight, product innovation direction |

#### Financial & Valuation Analysis

| Framework | Description | Best For |
|-----------|-------------|----------|
| **DuPont Analysis** | ROE = Net Profit Margin × Asset Turnover × Equity Multiplier | Profitability decomposition, financial health diagnosis |
| **DCF (Discounted Cash Flow)** | Free cash flow discounting | Enterprise/project valuation |
| **Comparable Company Analysis** | PE, PB, PS, EV/EBITDA multiples comparison | Relative valuation, peer benchmarking |
| **EVA (Economic Value Added)** | After-tax operating profit - Cost of capital | Value creation capability assessment |

#### Competitive & Strategic Positioning

| Framework | Description | Best For |
|-----------|-------------|----------|
| **Benchmarking** | Key performance indicator item-by-item comparison | Competitor gap analysis, best practice identification |
| **Strategic Group Mapping** | Cluster competitors along two key dimensions | Competitive landscape visualization, white-space identification |
| **Value Chain Analysis** | Primary activities + support activities value decomposition | Cost advantage sources, differentiation opportunity identification |
| **Blue Ocean Strategy** | Value curve, four-action framework (Eliminate-Reduce-Raise-Create) | Differentiated innovation, new market space creation |
| **Perceptual Mapping** | Plot brand positions along two consumer-perceived dimensions | Brand positioning analysis, market gap discovery |

#### Industry & Supply Chain Analysis

| Framework | Description | Best For |
|-----------|-------------|----------|
| **Industry Value Chain** | Upstream → Midstream → Downstream decomposition | Industry structure understanding, profit distribution analysis |
| **Gartner Hype Cycle** | Technology Trigger → Peak of Inflated Expectations → Trough of Disillusionment → Slope of Enlightenment → Plateau of Productivity | Emerging technology maturity assessment |
| **GE-McKinsey Matrix** | Industry Attractiveness × Competitive Strength | Business portfolio prioritization, investment decisions |

#### Selection Principles

1. **Domain-First**: Based on the domain identified in Step 1.1, select **2-4** most relevant frameworks from the toolkit above
2. **Complementary**: Choose complementary rather than overlapping frameworks (e.g., macro-level with PESTEL + micro-level with Porter's Five Forces)
3. **Depth over Breadth**: Better to deeply apply 2 frameworks than superficially stack 6
4. **Data-Feasible**: Selected frameworks must be supportable by downstream data collection skills — if the data required by a framework cannot be reasonably obtained, downgrade or substitute
5. **Explicit Mapping**: In the chapter skeleton, explicitly annotate which framework each chapter uses and how it is applied

#### Framework Selection Output Format

```markdown
## Framework Selection

| Chapter | Selected Framework(s) | Application |
|---------|----------------------|-------------|
| Market Size & Growth Trends | TAM-SAM-SOM + Product Life Cycle | TAM-SAM-SOM to quantify market space, PLC to determine market stage |
| Competitive Landscape Assessment | Porter's Five Forces + Strategic Group Mapping | Five Forces to assess industry competition intensity, Group Mapping to visualize competitive positioning |
| Consumer Profiling | RFM + Consumer Decision Journey | RFM to segment customer value, Decision Journey to identify key conversion nodes |
| Brand Strategy Recommendations | SWOT + Blue Ocean Strategy | SWOT to summarize overall landscape, Blue Ocean to guide differentiation direction |
```

### Step 1.3: Design Chapter Skeleton

Produce a hierarchical chapter structure. Each chapter must include:

1. **Chapter Title** — Professional, concise, subject-based (follow titling constraints in Formatting section)
2. **Analysis Objective** — What this chapter aims to reveal
3. **Analysis Logic** — The reasoning chain or framework (must reference the frameworks selected in Step 1.2)
4. **Core Hypothesis** — Preliminary hypotheses to be validated or refuted by data

#### Chapter Skeleton Output Format

```markdown
## Analysis Framework

### Chapter 1: [Title]
- **Analysis Objective**: [This chapter aims to...]
- **Analysis Logic**: [Framework or reasoning chain used]
- **Core Hypothesis**: [Hypotheses to validate]
- **Data Requirements**: (see Step 1.4)
- **Visualization Plan**: (see Step 1.5)

### Chapter 2: [Title]
...
```

### Step 1.4: Define Data Query Requirements Per Chapter

For each chapter, specify **exactly what data needs to be collected**. This is the bridge to downstream data collection skills.

Each data requirement entry must include:

| Field | Description |
|-------|-------------|
| **Data Metric** | The specific metric or data point needed (e.g., "China skincare market size 2020-2025 (in billion CNY)") |
| **Data Type** | Quantitative, Qualitative, or Mixed |
| **Suggested Sources** | Suggested source categories: Industry reports, financial statements, government statistics, social media, e-commerce platforms, survey data, news |
| **Search Keywords** | Suggested search queries for data collection agents |
| **Priority** | P0 (Required) / P1 (Important) / P2 (Supplementary) |
| **Time Range** | The time period the data should cover |

#### Data Requirements Output Format (per chapter)

```markdown
#### Data Requirements

| # | Data Metric | Data Type | Suggested Sources | Search Keywords | Priority | Time Range |
|---|-------------|-----------|-------------------|-----------------|----------|------------|
| 1 | Market size (billion CNY) | Quantitative | Industry reports, government statistics | "China skincare market size 2024" | P0 | 2020-2025 |
| 2 | CAGR | Quantitative | Industry reports | "skincare CAGR growth rate" | P0 | 2020-2025 |
| 3 | Sub-category share | Quantitative | E-commerce platforms, industry reports | "skincare category share cream serum sunscreen" | P1 | Latest |
| 4 | Policy & regulatory updates | Qualitative | Government announcements, news | "cosmetics regulation 2024" | P2 | Past 1 year |
```

### Step 1.5: Define Visualization & Content Structure Per Chapter

For each chapter, specify the **planned visualization** and **content structure** for the final report:

| Field | Description |
|-------|-------------|
| **Visualization Type** | Chart type: Line chart, bar chart, pie chart, scatter plot, radar chart, heatmap, Sankey diagram, comparison table, etc. |
| **Visualization Title** | Descriptive title for the chart |
| **Visualization Data Mapping** | Which data indicators map to X/Y axes or segments |
| **Comparison Table Design** | Column headers and comparison dimensions for the data contrast table |
| **Argument Structure** | The planned "What → Why → So What" narrative outline |

#### Visualization Plan Output Format (per chapter)

```markdown
#### Visualization & Content Plan

**Chart 1**: [Type] — [Title]
- X-axis: [Dimension], Y-axis: [Metric]
- Data source: Corresponds to Data Requirement #1, #2

**Comparison Table**:
| Dimension | Item A | Item B | Item C |
|-----------|--------|--------|--------|

**Argument Structure**:
1. **Observation (What)**: [Surface phenomenon revealed by data]
2. **Attribution (Why)**: [Driving factors or underlying causes]
3. **Implication (So What)**: [Strategic implications or recommended actions]
```

### Step 1.6: Output Complete Analysis Framework

Assemble all outputs into a single, structured **Analysis Framework Document**:

```markdown
# [Research Subject] Analysis Framework

## Research Overview
- **Research Subject**: [...]
- **Scope**: [Geography, time range, industry segment]
- **Analysis Domain**: [Market / Finance / Industry / Brand / Consumer / ...]
- **Core Research Questions**: [1-3 key questions]

## Framework Selection

| Chapter | Selected Framework(s) | Application |
|---------|----------------------|-------------|
| ... | ... | ... |

## Chapter Skeleton

### 1. [Chapter Title]
- **Analysis Objective**: [...]
- **Analysis Logic**: [...]
- **Core Hypothesis**: [...]

#### Data Requirements
| # | Data Metric | Data Type | Suggested Sources | Search Keywords | Priority | Time Range |
|---|-------------|-----------|-------------------|-----------------|----------|------------|
| ... | ... | ... | ... | ... | ... | ... |

#### Visualization & Content Plan
[Chart plan + Comparison table design + Argument structure]

### 2. [Chapter Title]
...

### N. [Chapter Title]
...

## Data Collection Task List
[Consolidate all P0/P1 data requirements across chapters into a structured task list for downstream data collection skills to execute]
```

## Phase 1 Quality Checklist

- [ ] Analysis framework covers all natural dimensions for the identified domain
- [ ] 2-4 professional analysis frameworks are selected and explicitly mapped to chapters
- [ ] Selected frameworks are complementary (not overlapping) and data-feasible
- [ ] Each chapter has clear Analysis Objective, Analysis Logic (referencing chosen framework), and Core Hypothesis
- [ ] Data requirements are specific, measurable, and include search keywords
- [ ] Every chapter has at least one visualization plan
- [ ] Data priorities (P0/P1/P2) are assigned realistically
- [ ] The framework is actionable — a data collection agent can execute on the Search Keywords directly
- [ ] Data Collection Task List is comprehensive and deduplicated

---

# Phase 1→2 Handoff: Data Collection & Chart Generation

After the analysis framework is generated, it is handed off to **other data collection skills** (e.g., deep-research, data-analysis, web search agents) to:

1. Execute the **Search Keywords** from each chapter's data requirements
2. Collect quantitative data, qualitative insights, and source URLs
3. Generate charts based on the **Visualization & Content Plan**
4. Return a **Data Package** containing:
   - **Data Summary**: Raw numbers, metrics, and qualitative findings per chapter
   - **Chart Files**: Generated chart images with local file paths
   - **External Search Findings**: Source URLs and summaries for citations

> **This skill does NOT perform data collection.** It only produces the framework (Phase 1) and the final report (Phase 2).
>
> **Chart Generation**: If a visualization/charting skill is available (e.g., data-analysis, image-generation), chart generation can be deferred to the beginning of Phase 2 — see Step 2.3.

---

# Phase 2: Report Generation

## Purpose

Receive the completed **Analysis Framework** and **Data Package** from upstream, and synthesize them into a final consulting-grade report.

## Phase 2 Inputs

| Input | Description | Required |
|-------|-------------|----------|
| **Analysis Framework** | The framework document produced in Phase 1 | Yes |
| **Data Summary** | Collected data organized per chapter from the data collection phase | Yes |
| **Chart Files** | Local file paths for generated chart images. If not provided, will be generated in Step 2.3 using available visualization skills | Optional |
| **External Search Findings** | URLs and summaries for inline citations | Optional |

## Phase 2 Workflow

### Step 2.1: Receive and Validate Inputs

Verify that all required inputs are present:

1. **Analysis Framework** — Confirm it contains chapter skeleton, data requirements, and visualization plans
2. **Data Summary** — Confirm it contains data organized per chapter, cross-reference against P0 requirements
3. **Chart Files** — Confirm file paths are valid local paths

If any P0 data is missing, note it in the report and flag for the user.

### Step 2.2: Map Report Structure

Map the final report structure from the Analysis Framework:

1. **Abstract** — Executive summary with key takeaways
2. **Introduction** — Background, objectives, methodology
3. **Main Body Chapters (2...N)** — Mapped from the Framework's chapter skeleton
4. **Conclusion** — Pure, objective synthesis
5. **References** — GB/T 7714-2015 formatted references

### Step 2.3: Generate Chapter Charts (Pre-Report Visualization)

Before writing the report, generate all planned charts from the Analysis Framework's **Visualization & Content Plan**. This step ensures every sub-chapter has its "Visual Anchor" ready before narrative writing begins.

#### When to Execute This Step

- **Chart Files already provided**: Skip this step — proceed directly to Step 2.4.
- **Chart Files NOT provided but a visualization skill is available**: Execute this step to generate all charts first.
- **No Chart Files and no visualization skill available**: Skip this step — use comparison tables as the primary visual anchor in Step 2.4, and note the absence of charts.

#### Chart Generation Workflow

1. **Extract Chart Tasks**: Parse all `Visualization & Content Plan` entries from the Analysis Framework to build a chart generation task list:

| # | Chapter | Chart Type | Chart Title | Data Mapping | Data Source |
|---|---------|------------|-------------|--------------|-------------|
| 1 | 2.1 | Line chart | Market Size Trend 2020-2025 | X: Year, Y: Market Size (billion CNY) | Data Requirement #1, #2 |
| 2 | 3.1 | Pie chart | Consumer Age Distribution | Segments: Age groups, Values: Share % | Data Requirement #5 |
| ... | ... | ... | ... | ... | ... |

2. **Prepare Chart Data**: For each chart task, extract the corresponding data points from the **Data Summary**.
   > **CRITICAL**: Use ONLY the numbers provided in the Data Summary. Do NOT invent or "smooth" data to make charts look better. If data points are missing, the chart must reflect that reality (e.g., broken line or missing bar), or the chart type must be adjusted.

3. **Delegate to Visualization Skill**: Invoke the available visualization/charting skill (e.g., `data-analysis`) for each chart task with:
   - Chart type and title
   - Structured data
   - Axis labels and formatting preferences
   - Output file path convention: `charts/chapter_{N}_{chart_index}.png`

4. **Collect Chart File Paths**: Record all generated chart file paths for embedding in Step 2.4:

```markdown
## Generated Charts
| # | Chapter | Chart Title | File Path |
|---|---------|-------------|-----------|
| 1 | 2.1 | Market Size Trend 2020-2025 | charts/chapter_2_1.png |
| 2 | 3.1 | Consumer Age Distribution | charts/chapter_3_1.png |
```

5. **Validate**: Confirm all P0-priority charts have been generated. If any chart generation fails, note it and fall back to comparison tables for that sub-chapter.

> **Principle**: Complete ALL chart generation before starting report writing. This ensures a consistent visual narrative and avoids interleaving generation with writing.

### Step 2.4: Write the Report

For each sub-chapter, follow the **"Visual Anchor → Data Contrast → Integrated Analysis"** flow:

1. **Visual Evidence Block**: Embed charts using `![Image Description](Actual_File_Path)` — use the file paths collected in Step 2.3
2. **Data Contrast Table**: Create a Markdown comparison table for key metrics
   > **Source Rule**: Every number in the table must come from the Data Summary. No hallucinations.
3. **Integrated Narrative Analysis**: Write analytical text following "What → Why → So What"
   > **Narrative Rule**: Narrative must explain the *provided* data. Do not make claims unsupported by the inputs.

Each sub-chapter must end with a robust analytical paragraph (min. 200 words) that:
- Synthesizes conflicting or reinforcing data points
- Reveals the underlying user tension or opportunity
- Optionally ends with a punchy "One-Liner Truth" in a blockquote (`>`)

### Step 2.5: Final Structure Self-Check

Before outputting, confirm the report contains **all sections in order**:

```
Abstract → 1. Introduction → 2...N. Body Chapters → N+1. Conclusion → N+2. References
```

Additionally verify:
- All charts generated in Step 2.3 are embedded in the correct sub-chapters
- Chart file paths in `![](path)` references are valid
- Sub-chapters without charts have comparison tables as visual anchors

The report **MUST NOT** stop after the Conclusion — it **MUST** include References as the final section.

## Formatting & Tone Standards

### Consulting Voice
- **Tone**: McKinsey/BCG — Authoritative, Objective, Professional
- **Language**: All headings and content in the language specified by `output_locale`
- **Number Formatting**: Use English commas for thousands separators (`1,000` not `1，000`)
- **Data emphasis**: **Bold** important viewpoints and key numbers

### Titling Constraints
- **Numbering**: Use standard numbering (`1.`, `1.1`) directly followed by the title
- **Forbidden Prefixes**: Do NOT use "Chapter", "Part", "Section" as prefixes
- **Allowed Tone Words**: Analysis, Profiling, Overview, Insights, Assessment
- **Forbidden Words**: "Decoding", "DNA", "Secrets", "Mindscape", "Solar System", "Unlocking"

### Sub-Chapter Conclusions
- **Requirement**: End each sub-chapter with a robust analytical paragraph (min. 200 words).
- **Narrative Flow**: This paragraph must look like a natural continuation of the text. It must synthesize the section's findings into a strategic judgment.
- **Content Logic**:
    1.  Synthesize the conflicting or reinforcing data points above.
    2.  Reveal the *underlying* user tension or opportunity.
    3.  Key Insight: **Optional**: Only if you have a concise, punchy "One-Liner Truth", place it at the very end using a **Blockquote** (`>`) to anchor the section.

### Insight Depth (The "So What" Chain)

Every insight must connect **Data → User Psychology → Strategy Implication**:

```
❌ Bad: "Females are 60%. Strategy: Target females."

✅ Good: "Females constitute 60% with a high TGI of 180. **This suggests**
   the purchase decision is driven by aesthetic and social validation
   rather than pure utility. **Consequently**, media spend should pivot
   towards visual-heavy platforms (e.g., RED/Instagram) to maximize CTR,
   treating male audiences only as a secondary gift-giving segment."
```

### References
- **Inline**: Use markdown links for sources (e.g. `[Source Title](URL)`) when using External Search Findings
- **References section**: Formatted strictly per **GB/T 7714-2015**

### Markdown Rules
- **Immediate Start**: Begin directly with `# Report Title` — no introductory text
- **No Separators**: Do NOT use horizontal rules (`---`)

## Report Structure Template

```markdown
# [Report Title]

## Abstract
[Executive summary with key takeaways]

## 1. Introduction
[Background, objectives, methodology]

## 2. [Body Chapter Title]
### 2.1 [Sub-chapter Title]
![Chart Description](chart_file_path)

| Metric | Brand A | Brand B |
|--------|---------|--------|
| ... | ... | ... |

[Integrated narrative analysis: What → Why → So What, min. 200 words]

> [Optional: One-liner strategic truth]

### 2.2 [Sub-chapter Title]
...

## N+1. Conclusion
[Pure objective synthesis, NO bullet points, neutral tone]
[Para 1: The fundamental nature of the group/market]
[Para 2: Core tension or behavior pattern]
[Final: One or two sentences stating the objective truth]

## N+2. References
[1] Author. Title[EB/OL]. URL, Date.
[2] ...
```

## Complete Example

### Phase 1 Example: Framework Generation

User provides: Research subject "Gen-Z Skincare Market Analysis"

**Phase 1 output (Analysis Framework):**

```markdown
# Gen-Z Skincare Market Analysis Framework

## Research Overview
- **Research Subject**: Gen-Z Skincare Market Deep Analysis
- **Scope**: China market, 2020-2025, consumers aged 18-27
- **Analysis Domain**: Market Analysis + Consumer Insights
- **Core Research Questions**:
  1. What is the size and growth momentum of the Gen-Z skincare market?
  2. What is unique about Gen-Z consumer skincare behavior patterns?
  3. How can brands effectively reach and convert Gen-Z consumers?

## Chapter Skeleton

### 1. Market Size & Growth Trends
- **Analysis Objective**: Quantify Gen-Z skincare market size and identify growth drivers
- **Analysis Logic**: Total market → Segmentation → Growth rate → Driver decomposition
- **Core Hypothesis**: Gen-Z is becoming the core engine of skincare consumption growth

#### Data Requirements
| # | Data Metric | Data Type | Suggested Sources | Search Keywords | Priority | Time Range |
|---|-------------|-----------|-------------------|-----------------|----------|------------|
| 1 | China skincare market total size | Quantitative | Industry reports | "China skincare market size 2024 2025" | P0 | 2020-2025 |
| 2 | Gen-Z skincare spending share | Quantitative | Industry reports, e-commerce platforms | "Gen-Z skincare spending share youth" | P0 | Latest |

#### Visualization & Content Plan
**Chart 1**: Line chart — China Skincare Market Size Trend 2020-2025
**Argument Structure**:
1. What: Quantified status of market size and Gen-Z share
2. Why: Consumption upgrade, ingredient-conscious consumers, social media driven
3. So What: Brands should prioritize building youth-oriented product lines

### 2. Consumer Profiling & Behavioral Insights
...

## Data Collection Task List
[Consolidated P0/P1 tasks]
```

### Phase 2 Example: Report Generation

After data collection, user provides: Analysis Framework + Data Summary with brand metrics + chart file paths.

**Phase 2 output (Final Report) follows this flow:**

1. Start with `# Gen-Z Skincare Market Deep Analysis Report`
2. Abstract — 3-5 key takeaways in executive summary form
3. 1. Introduction — Market context, research scope, data sources
4. 2. Market Size & Growth Trend Analysis — Embed trend charts, comparison tables, strategic narrative
5. 3. Consumer Profiling & Behavioral Insights — Demographics, purchase drivers, "So What" analysis
6. 4. Brand Competitive Landscape Assessment — Brand positioning, share analysis, competitive dynamics
7. 5. Marketing Strategy & Channel Insights — Channel effectiveness, content strategy implications
8. 6. Conclusion — Objective synthesis in flowing prose (no bullets)
9. 7. References — GB/T 7714-2015 formatted list

---

## Quality Checklists

### Phase 1 Quality Checklist (Analysis Framework)

- [ ] Framework covers all natural analytical dimensions for the identified domain
- [ ] Each chapter has clear Analysis Objective, Analysis Logic, and Core Hypothesis
- [ ] Data requirements are specific, measurable, and include actionable Search Keywords
- [ ] Every chapter has at least one visualization plan with chart type and data mapping
- [ ] Data priorities (P0/P1/P2) are assigned — P0 items are essential for core arguments
- [ ] Data Collection Task List is comprehensive, deduplicated, and ready for downstream execution
- [ ] Framework adapts to the correct domain (market/finance/industry/consumer/etc.)

### Phase 2 Quality Checklist (Final Report)

- [ ] **NO HALLUCINATION**: All numbers and charts are verified against the input Data Summary
- [ ] All planned charts generated before report writing (Step 2.3 completed first)
- [ ] All sections present in correct order (Abstract → Introduction → Body → Conclusion → References)
- [ ] Every sub-chapter follows "Visual Anchor → Data Contrast → Integrated Analysis"
- [ ] Every sub-chapter ends with a min. 200-word analytical paragraph
- [ ] All insights follow the "Data → User Psychology → Strategy Implication" chain
- [ ] All headings use proper numbering (no "Chapter/Part/Section" prefixes)
- [ ] Charts are embedded with `![Description](path)` syntax
- [ ] Numbers use English commas for thousands separators
- [ ] Inline references use markdown links where applicable
- [ ] References section follows GB/T 7714-2015
- [ ] No horizontal rules (`---`) in the document
- [ ] Conclusion uses flowing prose — no bullet points
- [ ] Report starts directly with `#` title — no preamble
- [ ] Missing P0 data is explicitly flagged in the report

## Output Format

- **Phase 1**: Output the complete Analysis Framework in **Markdown** format
- **Phase 2**: Output the complete Report in **Markdown** format

## Settings

```
output_locale = zh_CN  # configurable per user request
reasoning_locale = en
```

## Notes

- This skill operates in **two phases** of a multi-step agentic workflow:
  - **Phase 1** produces the analysis framework and data collection requirements
  - **Data collection** is performed by other skills (deep-research, data-analysis, etc.)
  - **Phase 2** receives the collected data and produces the final report
- Dynamic titling: **Rewrite** topics from the Framework into professional, concise subject-based headers
- The Conclusion section must contain **NO** detailed recommendations — those belong in the preceding body chapters
- **ZERO HALLUCINATION POLICY**: Each statement, chart, and number in the report must be supported by data points from the input Data Summary. If data is missing, admit it.
- **Traceability**: If requested, you must be able to point to the specific line in the Data Summary or External Search Findings that supports a claim.
- The framework should adapt its analytical dimensions and depth to the specific domain (financial analysis uses different frameworks than consumer insights)
- When the research subject is ambiguous, default to the broadest reasonable scope and note assumptions

## 版本历史

- V1.16.0 (2026-09-26): **M5 增第⑫问「一条"这维不可比/做不到"的边界声明，是不是被当成了该维的测量结果？」**（disclaimer-as-measurement；心屿 SoulIsle 对标轮 r40b/r40c 一手实证）。根因：r35 报告写下「各参照仓公开性能数字仍无可比口径 ⇒ 只讲机制有无」后，**r36–r39 连续四轮都没再碰性能维**（每轮都拿这句当"已处理"），直到 r40 被用户追问同一条命令时才发现该维从未真正测量；真去量才有数——本机 p95 15.9–26.2ms / 吞吐 2026.9 rps，CI 三次采样 p95 3.3–7.1ms / 吞吐 976–1179 rps（**同一环境跨 run 就差 2.15×**），这些事实被一句免责声明挡在视野外四轮。规定动作：`不可比 / 无可比口径 / 受限于 / 仅保证 / 无法做到` 类句子必须同句带实证标记（实测 · 逐文件 · 命令路径 · run 号 · 计数+单位 · sha/CRC · ls-remote）；拿不出实证就**改写成诚实缺口**（`未实测` + 下一步取证路径），两种形态不得混写；**`未实测`/`❌` 不在本判据拦截面内**（那是"知道没测"，不是"宣称做不了"）。对照=**已取两侧**（漏报侧：判据首跑即在 r35 报告里点名该句并 `rc=1`，补上取证指向 §14.1 后转 `CLEAN 0 处`；误伤侧：8 类桩含"同句带 run 号"/"带计数+单位"/"诚实缺口标注"三条 **期望 0 命中且实得 0**，另有恒绿守卫——最裸的边界句若不命中即 selftest 直接 FAIL。**并且首跑就自抓一处判据自身缺陷**：`口径` 一词同时出现在被拦词表与实证白名单里，句子会自己洗白自己（漏报侧桩期望 1 实得 0 当场暴露）⇒ 已从白名单移除）。执行器＝项目侧常驻判据 `_test/disclaimer_forensics_lint.py`，已接进电池 CI 阻断链（套件 **49→51→52**：本条 +2，同轮 `ci_watch` 判定桩 +1；G4/G10 恒等式当前为 **实跑 49 + 豁免 3 == 52**）。并发提示：本版采集期间该文件被并行会话整卷扩写（13,379B → 69,150B），故本版**只用行级 Edit**（锚点命中数==1），落账后 `git diff --numstat` 读数 **2/2** ⇒ 未覆盖他人内容；这条自证本身就是⑫要的"取回执不取假设"。本版**只加一问，不改 M1-M7 任何阈值与 ①-⑪ 任何子问**。效果回归: 变好（当轮即把四轮空转的一维变成有数、有预算、有常驻判据的一格）。
- V1.15.0 (2026-09-26): **M5 增第⑪问「能力位来自配置存在还是行为闭环」**（心屿 SoulIsle 对标轮 r37 一手实证）。根因：本轮新开"工程治理"面扫 16 仓 dependabot，实测**只有 1/16 配了**且就是我方，账面结论是"领先 15 家"；同日清队列时发现自己 4 条 dependabot PR **挂了 2 天无人处理**（`open_issues=4` 全是被计成 issue 的 PR）⇒ 真实形态"配置在册、行为未闭环"。规定动作：配置类能力位（dependabot / CI / SECURITY.md / `.gitattributes` / Release）**必须带一条行为回执**才计入优势——被处理过的 PR 数、目标 commit 上真跑过的 run 结论、资产数与下载计数、工作树字节==blob 的常驻判据；取不到回执写「配置在册·行为未证」，禁止与有回执的一方并列计数。对照=已取（正例：本轮清空 3/4 队列（#1 #3 #4 合并，三次 push 的 CI 逐项 `success`：run `36236421340`/`36236407756`/`36236336824`）后"dependabot 有效"才第一次有回执，并把"每轮 savepoint 顺带 `gh pr list --author app/dependabot`"登记为节奏项；反例：合并前同一份台账印出的 `deps_autoupdate=2/16 我方领先` 是纯配置存在性结论，与真实维护行为方向相反）。项目侧落点 `陪聊/memory/07-next-steps.part26.md` P2「依赖队列 SLA」与 `陪聊/交付物/对标分析报告-2026-09-26.md` §11.2。并发提示：本版采集期间该文件被另一会话 bump 到 V1.14.0，故锚点按**现读版本行**重锚（先 `grep -m1 ^version:` 再拼补丁，否则 `--dry-run` 第 1 项未命中、全量不生效——这恰是本问要求的"取回执而非取假设"）。本版**只加一问，不改 M1-M7 任何阈值与 ①-⑩ 任何子问**。效果回归: 变好（当轮即把我方一个账面"领先项"降级为"待回执项"并当场补出回执）。
- V1.14.0 (2026-09-26): **Step 1.05 引言改为「条数以本表实际序号为准，禁手抄计数」**（本技能自证：V1.12.0 落 M7 后本行仍写「逐条过四问」，而表已 7 行 —— 与 M5 行自己记下的「六条问落后三轮」同族，且这次落在**教别人别犯此错的那句话**上）。修法不写死新数字（写 7 会再落后一次），改为声明计数来源并点名旧形态。同轮二次自证：收口时先试 `replace --patch … --desc …` 得 `rc=2 unrecognized arguments` —— `--desc` 只属于 `commit`，批量模式必须「replace --no-commit → commit --desc」两步；本轮未把该号直接 bump 成 V1.15.0，因失败调用未写盘（判据：只记**改变了被描述对象状态**的动作，否则版本历史会积累未发生的变更）。对照=已取（① 漏报侧：`grep -c '^| M[0-9] |'` 实跑 **7**，与改前文案「四问」不等 ⇒ 腐化可复算、非臆断；② 误报侧：改后文案不含任何具体条数，未来新增 M 行不再使其失真；③ 附带核验 M7 行仍为 1 处、V1.13.0 条目完好未被覆盖）。本版改动共 3 处：Step 1.05 引言计数口径 + 本条目 + frontmatter 版本行；**不改 M1-M7 任何阈值与子问**。效果回归: 变好（判据表加行不再顺带腐化其引言）。
- V1.13.0 (2026-09-26): **M5 增第⑩问「逐字节/体积/哈希类主张的字节前提是否被钉住」**（心屿 SoulIsle 对标轮 r36 一手实证）。根因：本仓连做六轮"SHA256 双向对账 / 首屏 831,152B 预算 / 线上==权威源"主张，全部隐含"两侧同一份字节"，而 `core.autocrlf=true` + `* text=auto` 让工作树是 CRLF、blob 是 LF —— 换机器 checkout 同一提交字节就变，主张只在开发机上成立。修法三层：`.gitattributes` 钉 `* text=auto eol=lf` + 20 类扩展名显式 `binary`；108 文本文件 `git add --renormalize`（"零内容改动"由 `git diff --name-only` == `--ignore-cr-at-eol --name-only` 证明）；常驻判据 `eol_parity`（E1 属性表 / E2 工作树==blob / E3 binary 形状 / E4 分母闭合 + 8 类自证）。**归一化本身当场伤过一次**：首轮把 20 个二进制里出现的 `0D0A` 当行尾剥掉（PNG/MP4 少 1–2 字节），而"两侧都归一再比"的守卫**看不见内容损失**（两侧同样破坏），全部 `git show HEAD:` 还原后把该形状固化为 E3。同一轮判据接上即抓到同类写盘口：`Path.write_text` 文本模式在 Windows 产出 `b'a\r\nb\r\n'`，台账一次重写就被打回 CRLF；按"修一类不修一例"枚举三处出口（台账 / **补丁器** / 视频流水线产物）改 `write_bytes`，并把补丁器的"写后读回"也改按字节（`read_text` 会掩盖 CR）。对照=已取（正例：归一后两份不同 `autocrlf`（true / input）的 clone 文本侧字节一致、`EOL-PARITY-PASS text=205 binary=20 total=225`、公网重部署后 `live=9288 == local=9288` 不再走"仅行尾差异"兜底分支；反例：归一前同一提交在两侧字节不同（差异 100% 落在 `0D0A`），且"两侧归一再比"对二进制损伤给出零差值——已由 E3 + 夹具钉住，8 类样本自证含"未声明 binary 的 `0D0A` 判红"与"声明 binary 不要求含 `0D0A`"两个方向）。项目侧落点 `陪聊/_test/eol_parity_check.py`、`陪聊/memory/06-constraints.md`「行尾红线/二进制红线」。本版**只加一问，不改 M1-M7 任何阈值与 ①-⑨ 任何子问**。效果回归: 变好（当轮即把"本机绿"升级为"任何 clone 可复算"，并当场抓到同类写盘口三处）。
- V1.12.0 (2026-09-26): **Step 1.05 增 M7「产品承诺的语言，每个可见面是否兑现」**（孝心联对标第 30 轮一手实证，用户选中上轮 footer「未做对照前不落规则」→ 本轮先补对照再落）。根因：孝心联双语长者站切到中文后正文全翻译、但浏览器标签页标题仍英文——`<title>` 不在 `data-i18n` 体系、`applyTranslations` 从不碰 `document.title`；这类"正文翻了、外壳没翻"的漏配对**静态 HTML 断言完全隐形**，必须真浏览器切语言观测（是 M5③ 的下游：不止测量环境，测量**语言**也得是目标语言）。修法以**零英文回归 + 零新字典键**：能派生的页用「已译 h1 + 已双语存在的品牌键」重算，源语言专有的少数页标 manual 进显式 allow-list、不强制派生否则改英文=回归。对照=已取（证据：先写隔离桩再落规则——桩含漏报侧 1 + 误报侧 3 （已兑现 / allow-list 例外 / 单语）+ 边界 2（无 i18n 证据→UNVERIFIED、空分母→不得 PASS），首跑即抓到谓词自身漏洞「多语承诺但页面清单为空却返回 PASS」，按 `R247` 改成 UNVERIFIED 后 6/6 绿；项目侧 t_page_title 5 条变异逐条转红、链 2034→2040，提交 filialconnect `8c4310f`/v1.9.0）。本版**只加一行判据 M7，不改 M1-M6 任何阈值与 ①-⑨ 任何子问**。效果回归: 变好（当轮即把「文案要在承诺语言里兑现」从一句口头变成带双向对照的可跑判据）。
- V1.11.0 (2026-09-26): **M4「有判别力」增第二种失效形态：退出码只分两挡会把判据自身崩溃并入"环境未验"**（心屿 SoulIsle 对标轮 r35 一手实证）。根因：本轮给电池收口行写"rc==1 判红 / 其余 ENV-UNVERIFIED"两挡分类，随后 `voice` 与 `voice_selftest` 两条套件硬崩（`rc=0xC0000409`、stdout 一个字节都没有）被判成"环境未验"——红照报，但**性质变了没人看得见**；与本轮开头那条"账单配额已恢复却仍按旧归因挂着"同源。规定正解＝三挡 `1=判红 / 2=环境未验 / 其余=CRASH 点名（带 0x 十六进制）`，并要求合成四态套件双向验（PASS 不进红名单 / 崩溃不冒充环境 / 有红必 rc=1）。同轮另记一条测量错：`python x.py | tail; echo $?` 取的是 `tail` 的退出码 ⇒ 崩溃的判据被读成 rc=0；判 rc 必须重定向到文件后取 `$?`。对照=已取（证据=正例：改三挡后本机两段电池 21/21 与 20/20 全绿、远端四条 job success（run 36220200506）；反例：合成套件里 `os.abort()` 那条被点名成 `CRASH(...)=0xc0000409`，两挡旧写法同一输入判成 ENV）。项目侧落点 `陪聊/_test/run_all_suites.py` 收口行。本版**只在 M4 一格内追加一种形态与一句正解，不改 M1-M3、M5、M6 任何阈值与子问**。效果回归: 变好（当轮即把自己造的两挡分类改判三挡并留双向验）。
- V1.10.0 (2026-09-25): **Step 1.05 增 M6「材料是否追上被描述对象」（交付一致性回扫）**（心屿 SoulIsle 对标轮 r32 实证）。根因：连十一轮都在改产品，而评委唯一会读的 `application-plan.html`停在 09-24 —— grep 实测 `sw.js`/`Service Worker`/`PWA`/`逐字`/`流式`/`设置面板` **命中数全为 0**，离线壳与 SSE 流式两项差异能力在交付材料里不可见；这是「文档写了 ≠ 磁盘有」的**反向形态**（磁盘有、材料没写），因无任何告警而更难发现。规定动作：动笔前跑能力关键词命中数矩阵（0 命中即待补）→ 补节后重渲染 → **复算页数/图数**，页数贴上限须写明「再加内容必须先精简」。同轮把禁令扩到双向（原只禁「文档写了当磁盘有」）。对照=已取（证据=本轮按该动作补 6.7/7.8 两节 + 实拍图 8/9，PDF 由 18 页 8 图重渲为 20 页 9 图，页数正好贴官方 ≤20 上限并已写入交付清单；`python _test/pdf_leak_scan.py` 复扫 CLEAN，图数期望改从 HTML 现读 `<figure>` 不再手抄；项目侧提交 3d5543c）。本版**只加一行判据与一句禁令扩写，不改 M1-M5 任何阈值与 ①-⑨ 任何子问**。效果回归: 变好（当轮即让两项已建成能力进入交付材料）。
- V1.9.0 (2026-09-25): **M5 ④ 增反向教训「注入的反例自身须单独证伪」**——为验 DDL 原子性注入的"失败语句" `NOT_A_REAL_TYPE X` 其实被 SQLite 合法接受（列类型名不校验），于是测试拿到 raised=False，"回滚"从未被测到却显示绿色通过。对照=已取（正例：换成真语法错 INTEGER REFERENCES(;) 后判据立刻抓到未回滚的第一条 ALTER；反例：本轮首版夹具即被自己的假反例蒙过，是 L4 断言把它拽出来的）。落在 consulting-analysis/SKILL.md M5 ④ 段。本版只加一问的一半，不动 M1-M4 与 ①-⑨ 任何阈值。效果回归: 变好（同类"测的是夹具不是判据"的形态从此有点名判据）。
- V1.8.0 (2026-09-25): **M5 增第⑨问「这条全零/独有结论有第二条独立通道吗」**（心屿 SoulIsle 对标轮 r31 实证）。根因是自纠链条的下一环：r30 刚因「文件名匹配器假阴性」给自己装了盲区点名，而同一把尺也量着 16 个参照仓 ——卖点句 `pwa_offline=0/16` 只有一条腿（r27 仅手工抽查 lobehub 一仓）。本版规定：**单种观测法支撑的零结论不可信**，须另起互不依赖的第二通道复核，且对每条命中**先归因语境再计数**（同一个词可能有三种含义），取数失败的样本记 `unverified` 且此时**禁止**打印全零。一手实证：内容法（读 description+README，四类 `app_shell/local_models_offline/ml_training_offline/none`）跑出 `app_shell=0`（两法一致）、`local_models_offline=1`（Open-LLM-VTuber 本地模型完全离线＝桌面形态，不是网页壳）、`ml_training_offline=1`（hello-diana/MASCOT 的 offline DPO＝纯误报源，不归因就会凭空给对手加一格能力）、`unverified=0`；据此把措辞从「没人做离线」下调为「没人用浏览器 app-shell 做离线」。对照=已取（证据=四类样本 + 两条反向断言 + 零输入判 none 全通过；两个变异体（恒判 app_shell、恒判 none）均 `SELFTEST-FAIL rc=1`、还原 `rc=0`；项目侧红线已入 `陪聊/memory/06-constraints.md`「全零/差异结论须两条独立观测通道」）。同时修掉本节正文里手抄的「六条问」计数（实际已到九问）——改为「条数以本节现读为准，不手抄」，与 M5⑥ 同一判据精神。本版**只加一问与一处计数口径修正，不改 M1-M4 与 M5 ①-⑧ 任何阈值**。效果回归: 变好（当轮即把本项目的差异卖点从单法升级为双法互证）。
- V1.7.0 (2026-09-25): **M5 增第⑧问「同一把尺对双方是否同构」**（心屿 SoulIsle 对标轮 r30 实证）。根因：横向对表的 `CAP_RULES` 只读文件路径（参照仓只能静态取树），于是**我方能力写在文件内容里就看不见** —— self 行少报 `streaming`（5 个文件含 `text/event-stream`，文件名不含 sse）与 `e2e_browser`（20 个脚本 `import playwright`，路径不含 e2e）。两难：并进 `caps` 就是给一侧换尺（与 M5⑥"两处实现改宽一份"同族，且把台账变广告牌）；不管则读者读成"没做"。本版规定的正确形态：**计数不动，另立「盲区点名」单列**（`blind_spot_caps()` 读内容取证据、只印在自证行），并要求正文写明"该列是观测下限而非能力上限"。对照=已取（证据=三向自证通过：有证据→点名 / 无证据→空 / 已看见→不重复计数；三变异体（恒空 / 恒报 / 不去重）均被 `SELFTEST-FAIL` 抓住；peers 覆盖率矩阵与 `usable+blind==总数` 恒等式零变化；项目侧红线已入 `陪聊/memory/06-constraints.md`「对标台账禁手动加能力位」）。本版**只加一问，不改 M1-M4 与 M5 ①-⑦ 任何阈值**。效果回归: 变好（当轮即用于台账 self 行，且新增红线进 06 由复算命令背书）。
- V1.6.0 (2026-09-25): **M5 增第⑦问「分母本身被证明过吗」**（心屿 SoulIsle 对标轮 r27/r28 实证）。根因：能力矩阵长期打印 `pwa_offline=0/16` 这类全零值，而全零有两种成因——真没有 / 我没数到（树截断、取数失败）；旧 M5 的④只覆盖「命令恒 0 命中被读成确实没有」，未覆盖「**分母被静默缩小**」。本轮一手核 lobehub 树（20,740 对象、`truncated=false`）判为真零，并固化为判据：盲区仓踢出分母 + 逐条点名原因 + 恒等式 `usable+blind==总数`，配负控制（把判据换回「全部计入分母」的旧写法，selftest 立刻 rc=1）。同轮另把该问用于新交付的离线壳判据：R8「PRECACHE 清单 17 项必须真入缓存」与 A10「页面引用的每个 js/css/data 都必须在壳里」，堵的是同一类洞——**加了模块忘了进壳，离线就静默破**。对照=已取（正例：真实快照有效分母 16/16、13 类注入反例全抓；反例：旧写法 rc=1；报告见 `陪聊/交付物/对标分析报告-2026-09-25.md` §15.3 与 §16）。本版只加一问，不改 M1-M4 与 M5 ①-⑥ 任何阈值。效果回归: 变好（当轮即落到新判据的分母断言上）。
- V1.5.0 (2026-09-25): **M5 增第⑥问「同一判断是否只有一处实现」**（孝心联第十三轮实证）。根因：词典孤儿键判据有两处实现，上一轮只把 `test_build.py` 那份从「只扫 main.js」扩到全部脚本，CI 另跑的 `tools/check-i18n.py` 仍是旧规则 → 发版提交被判红并报回三个「孤儿键」；同族形态是「判据读仓外路径」（本地能跑、CI 检出里没有）。解法形态：规则收到一处，另一处用 importlib 加载并断言两侧结果集合相同（外加正样本防空转、变异体证明会红）。对照=已取（正例：合并后 1,492 全绿；反例：把合并后的函数退回「只扫 main.js」，两条对表断言同时红 2/1490 且工具自身报回那三个键；报告见 `孝心联/_internal/reports/2026-09-25_GitHub开源项目对标分析报告_第十三轮.md` §3；提交 filialconnect `5c21bba`）。本版只加判据，不改 M1-M4 与 M5 ①-⑤ 任何阈值。效果回归: 变好（当轮即拦下第二次漂移）。
- V1.4.0 (2026-09-25): **M5 增第⑤问「能力判定来自观测还是来自环境字符串」**（孝心联第十二轮对标实证）。根因：第十一轮为兜离线 ZIP 写下 `window.location.protocol === 'file:'` 即判定检索不可用，第十二轮本机实测**证伪**——测试浏览器开着文件访问权限，`file://` 下 `fetch` 与检索都能成功（`pagefind-entry.json` 200/172 B、索引分片 200/2213 B），协议猜测等于在一台能用的设备上把功能关掉；正解改为**首次加载失败即闭锁**，两种浏览器行为都对。⑤ 另含反向教训：**"我改不动它"不能当作"它不可用"的证据**（替换 `window.fetch` 为 reject 后 Pagefind 照常命中、fetch 计数 0，其分片加载不走主线程 fetch；要证不可用得让服务真的缺文件）。对照=已取（正例：缺索引环境实测 `ready→unavailable` 且闭锁后零重试；反例：`file://` 索引随包时实测命中真实页锚点，被原判据误关；报告见 `孝心联/_internal/reports/2026-09-25_GitHub开源项目对标分析报告_第十二轮.md` §3、§5；提交 filialconnect `7e494b9`）。本版只加判据，不改 M1-M4 与 M5 ①-④ 任何阈值。效果回归: 变好（该问当轮即拦下一次同类误判）。
- V1.3.0 (2026-09-25): **Step 1.05 增 M5「粒度·等级·环境同构」四问** —— 根因（孝心联第七轮对标实证）：该轮 **M1-M4 四条全部通过**，却仍漏掉三处真实缺陷，说明四问覆盖的是"数字本身可不可信"，未覆盖"**判据够不够细、拦不拦得住、跑在谁的设备上、判据命令自己有没有正样本**"。四条实测形态：① 类别阈值加权平均吞掉单项失败（axe `color-contrast` 归零 → accessibility 仍 0.96 > 0.95 error 门禁，藏六轮）；② `warn` 级断言不改退出码（SEO 实测 0.63 打日志绿了六轮）；③ 测量环境与用户不一致使整条代码路径从不执行（中文首访整页重排 CLS 0.262，仅 zh 浏览器发生）；④ **判据命令恒 0 命中被读成"确实没有"**（`grep` 在 `node_modules/` 下对任何已知串都返回 0，据此写进报告的"Lighthouse 无 prefers-color-scheme 仿真"是假结论，后被 A/B 推翻）。对照=已取（证据=正例：M5 四条判据各对应本会话一处实测缺陷；反例：M1-M4 全过的那一轮同一批数据仍漏掉这三处，报告见 `孝心联/_internal/reports/2026-09-25_GitHub开源项目对标分析报告_第七轮.md` §5「归因订正」；提交 filialconnect `ab3d25d`）。本版只加判据，不改 M1-M4 任何阈值。

- V1.2.0 (2026-09-24): **新增 Step 1.05「度量可信度前置审计」（R280，对标/竞品/自评类任务强制）** —— 在框架设计之前先对待用数字过 M1 当次实跑 / M2 同一份集 / M3 口径同分母 / M4 有判别力 四问，并要求报告带 `✅当次实测｜⚠️引用他处｜❌未核实` 三级取证标注。根因（焚诀第三轮对标实证）：一轮报告里 18 项缺陷有 11 项属「度量装置自己说谎」，且其中两条正是 M1（派生评分 JSON 与正文差 7.4 分、mtime 门禁放行）与 M2（账面盲测 93.1%/291 条 vs CI 实跑另一份 10 条集 75.0%）—— 报告结论会被这些数字带着走，而每个数字都「有出处」。对照=已取（证据=焚诀提交 205ed81 / 8a00233 与报告 §4 D-1~D-18）。
- V1.1.0 (2026-09-24): When to Use 增补「技术/工程项目对标」场景 + description 同步增补关键词（P2-12 闭环：unified_router 将「对标分析/改进建议」类查询直连本技能，但原场景面未覆盖技术对标，执行需自行推断；增补后语义对齐）。对照=已取：改前 When to Use 无任何「对标」字样（grep 0 命中），改后命中新增行；生效路径=2026-09-23 焚诀 GitHub 全景对标任务按其 Benchmarking 框架成功交付 reports/2026-09-23_GitHub开源项目全景对标分析.md。
- V1.0.0 (2026-09-22): 初始版（技能库入库时的咨询分析技能）。
