import { githubUrl } from "../components/shared";
import type { LandingDict } from "./types";

export function createZhDict(allowSignup: boolean): LandingDict {
  return {
  header: {
    cta: "立即开始",
    dashboard: "进入工作台",
    navigation: "主导航",
    openMenu: "打开导航菜单",
    closeMenu: "关闭导航菜单",
  },

  hero: {
    eyebrow: "B2B OUTBOUND · 中国 → 全球",
    headlineLine1: "不等询盘上门,",
    headlineLine2: "主动把海外客户谈进来。",
    subheading:
      "不投广告、不等询盘——用自然语言指挥 Agent,主动锁定海外 B 端买家、逐个谈进去。从零起步的团队用它搭线,已有打法的团队用它复刻。",
    cta: "出海查 AI · 已上线",
    ctaSecondary: "看完整拓客流程",
    downloadDesktop: "下载桌面端",
    talkToSales: "联系商务",
    worksWith: "已服务",
  },

  stats: {
    items: [
      { value: "10+", label: "已服务头部企业" },
      { value: "200+", label: "深度调研已交付" },
      { value: "6 步", label: "拓客全自动闭环" },
    ],
    trustedBy:
      "已服务 智能硬件 · 软件服务 · AI 出海 等领域的头部企业",
  },

  liveDemo: {
    badge: "LIVE DEMO",
    title: "演示中",
    subtitle: "看 Agent 如何把海外客户一步步谈进来",
    browserBar: "meridian.app / 外联工作台 LIVE",
    steps: [
      { id: "research", title: "德国渠道下钻 research", detail: "锁定线下家电连锁 · 首打 NRW / 巴伐利亚" },
      { id: "targets", title: "锁定连锁买家 targets", detail: "MediaHaus 等三家连锁,正招募供应商" },
      { id: "outreach", title: "撰写开发信 outreach", detail: "逐家定制,引用各自采购信号" },
      { id: "reply", title: "收到高意向询盘 reply", detail: "MediaHaus 采购回复,已归入收件箱" },
    ],
    standby: "全流程跑通 · 等待你的下一条指令",
    analyzing: "德国零售渠道 · 下钻分析中…",
    analysisItems: [
      { label: "线下家电连锁", percent: 52 },
      { label: "电商平台零售", percent: 33 },
      { label: "分销区域分销", percent: 15 },
    ],
    analysisNote:
      "◆ 首打区域 **NRW / 巴伐利亚** · 门店密度最高、集采集中 · CleanMax 提价后半价档真空",
    locked: "锁定连锁买家已校验",
    lockedTargets: [
      { initials: "MH", name: "MediaHaus", meta: "家电连锁 · NRW · 正招募供应商", tag: "✓ 高匹配" },
      { initials: "EX", name: "Expert SE", meta: "家电连锁 · 全德 · 集采决策", tag: "✓ 高匹配" },
      { initials: "EP", name: "EP:Group", meta: "家电连锁 · 巴伐利亚 · 门店密", tag: "✓ 高匹配" },
    ],
    drafting: "EN · 已生成",
    draftHeader: "TO: S. BRANDT · Buyer · MEDIAHAUS",
    draftBody:
      "Dear Ms. Brandt,\n\nSaw you're sourcing robot vacuums for the fall lineup — with CleanMax's price hike leaving the mid-tier open,\n\nwe ship from our DE warehouse in 12 days…",
    replyTitle: "高意向询盘 · 收件箱回复处理",
    replyMeta: "S. Brandt · MediaHaus 回复了",
    replyBody:
      "\"Interesting. Can you send your catalog and MOQ for the S9?\"",
    replyTag: "意向等级 高 · NRW 家电连锁采购",
    replyFooter: "✓ 已归入收件箱 · 高意向已置顶",
  },

  comparison: {
    label: "同样一件事 · 两种活法",
    headlineLine1: "把货卖进海外渠道,",
    headlineLine2: "过去要拿命赌,现在动动嘴。",
    pastLabel: "过去 · 又贵又慢又赌命",
    presentLabel: "现在 · 又快又省又可退",
    past: [
      { question: "怎么起步", answer: "飞去当地、雇一支本地队" },
      { question: "多久见效", answer: "大半年才摸清门道" },
      { question: "能试几个市场", answer: "预算只够押一个" },
      { question: "押错了", answer: "几百万打水漂" },
    ],
    present: [
      { question: "怎么起步", answer: "打一句话给 Agent" },
      { question: "多久见效", answer: "当天出第一批客户" },
      { question: "能试几个市场", answer: "想试几个试几个" },
      { question: "不合适", answer: "换个市场重跑一遍" },
    ],
  },

  features: {
    teammates: {
      label: "PRODUCT 01 · 已上线",
      title: "出海查 AI · 微信小程序",
      description:
        "国内做生意,你先查企查查。出海做生意,先用出海查。输入一家海外公司,拿到工商信息、经营信号,和一份 AI 深度报告——包括你的产品该从哪里切入。",
      cards: [
        { title: "工商信息核验", description: "注册信息、董事结构、存续状态,对接官方数据库。" },
        { title: "AI 深度调研", description: "采购动向、渠道结构、决策人——每条结论标注来源与置信度。" },
        { title: "切入路径建议", description: "上传你的产品资料,报告直接告诉你:这家客户该怎么谈。" },
      ],
    },
    autonomous: {
      label: "全球公司",
      title: "一查便知。",
      description:
        "覆盖主要海外市场公司数据库。从英国 Tesco 到德国 MediaHaus,输入公司名,几秒拿到工商信息、采购规模、合作可行性——AI 同时结合你的产品,告诉你切入路径。",
      cards: [
        { title: "官方数据库对接", description: "Companies House、Kantar、GLEIF 等权威来源,信息可追溯。" },
        { title: "结合你的业务", description: "上传产品资料后,报告自动结合你的品类给出切入建议。" },
        { title: "深度报告 · 一键生成", description: "合作可行性 / 采购情报 / 切入路径 / 风险,四个维度一次性给齐。" },
      ],
    },
    skills: {
      label: "微信入口",
      title: "微信扫一扫,免费开始第一次调研",
      description:
        "无需下载 App、无需注册账号。微信搜索「出海查AI」,输入一家海外公司,几秒拿到第一份 AI 深度报告。第一次免费。",
      cards: [
        { title: "微信搜索「出海查AI」", description: "在国内打开微信,搜索小程序名称,即可开始第一次公司调研。" },
        { title: "首次调研免费", description: "不限制公司大小、不限地区。从英国零售巨头到德国渠道连锁都能查。" },
        { title: "报告结尾一键触达", description: "报告结尾按钮直通完整六步拓客流程——从看懂一家公司,到把整个市场打下来。" },
      ],
    },
    runtimes: {
      label: "演示用例",
      title: "TESCO PLC · 公开信息演示",
      description:
        "演示数据基于 TESCO PLC 的公开信息。展示一份完整的出海查深度报告长什么样:从工商信息、采购规模,到结合你品类的切入建议。",
      cards: [
        { title: "工商信息核验", description: "注册编号 00445790 · Active 存续 · Companies House · 市占 27.4%。" },
        { title: "采购情报", description: "母婴品类年采购 £3–5 亿 · 供应商准入 BSCI / UKCA。" },
        { title: "AI 切入路径建议", description: "先通过自有品牌(Own Label)代工切入,再谈品牌入驻——需 BSCI/SEDEX + UKCA。" },
      ],
    },
  },

  transition: {
    headlineLine1: "看懂一家公司只是起点。",
    headlineLine2: "下一步,是把整个市场*打下来*。",
    body: "出海查报告结尾的那个按钮 AI 自动触达,通往完整的六步拓客流程 ↓",
  },

  workflow: {
    label: "PRODUCT 02 · 内测排队中",
    headline: "拓客引擎:从市场调研,到成交前一步内测",
    subheading:
      "不是又一个发邮件工具。是一个覆盖出海拓客全程的工作台——每一步都有 AI 跑腿、给判断,再由你确认放行。",
    steps: [
      {
        id: "research_market",
        code: "STEP 01 / RESEARCH_MARKET",
        title: "01 市场调研",
        detail: "锁定市场后,自动下钻到该打的区域和渠道",
        panel: {
          title: "三个候选市场,已排好序",
          subtitle: "待你确认首打市场",
          body: "Lumo S9 扫地机器人 · 代理",
          bullets: [
            "DE 德国 · 87 · 成熟零售渠道 · CR5 62% 集中 · 半价档真空",
            "FR 法国 · 73 · 渠道分散 · 分销为主",
            "ND 北欧 · 69 · 客单价高 · 决策周期长",
          ],
        },
      },
      {
        id: "build_icp",
        code: "STEP 02 / BUILD_ICP",
        title: "02 构建 ICP",
        detail: "把市场拆成具体买家群体,算清规模与决策链",
        panel: {
          title: "德国市场,拆成两个买家群体",
          subtitle: "待你选定主攻群体",
          body: "依据 产品定位 + 公开采购信号",
          bullets: [
            "群体 ① · 主力:零售连锁 · 品类采购总监 · 预估规模 ~1,800 · 决策链 采购总监 → 品类 VP · 触达难度 中",
            "群体 ② · 补充:独立家电经销商 · 负责人 · 预估规模 ~3,200 · 决策链 单点决策 · 触达难度 低",
          ],
        },
      },
      {
        id: "find_contacts",
        code: "STEP 03 / FIND_CONTACTS",
        title: "03 查找联系人",
        detail: "扒出决策人、校验邮箱,给你能直接发的名单",
        panel: {
          title: "决策人 + 可送达邮箱,已校验",
          subtitle: "从公开源提取 · 逐个去重",
          body: "邮箱 SMTP 校验通过",
          bullets: [
            "决策人姓名 + 职位 + 部门",
            "邮箱经 SMTP 校验,过滤失效地址",
            "标注信息来源与更新时间",
          ],
        },
      },
      {
        id: "draft_sequence",
        code: "STEP 04 / DRAFT_SEQUENCE",
        title: "04 撰写序列",
        detail: "每个客户一封专属英文开发信,引用它的采购信号",
        panel: {
          title: "每个客户,一封专属的信",
          subtitle: "首封 + 3 轮跟进",
          body: "基于该客户采购信号生成 · 不是模板群发一万人",
          bullets: [
            "逐客户定制,不是变量替换",
            "首封信引用该客户的真实采购信号",
            "3 轮跟进节奏按决策周期自动编排",
          ],
        },
      },
      {
        id: "schedule_send",
        code: "STEP 05 / SCHEDULE_SEND",
        title: "05 排期发送",
        detail: "按对方时区在工作时间投递,域名预热限流",
        panel: {
          title: "在对方的早上九点,准时抵达",
          subtitle: "按时区排期 · 限流投递",
          body: "发信信誉度托管 · 一封都不在半夜打扰",
          bullets: [
            "对方本地工作时间送达",
            "域名预热曲线,规避拉黑风险",
            "首封与跟进按最优间隔编排",
          ],
        },
      },
      {
        id: "handle_reply",
        code: "STEP 06 / HANDLE_REPLY",
        title: "06 回复处理",
        detail: "回信自动分级收进收件箱,高意向置顶提醒",
        panel: {
          title: "收件箱自动分级,高意向置顶",
          subtitle: "回信自动意向分级",
          body: "该你出场时才提醒 · 你只需要处理真正重要的那几封",
          bullets: [
            "意向分级:高 / 待跟进 / 无效",
            "高意向置顶,并提醒你接手",
            "跟进型自动排入下一轮节奏",
          ],
        },
      },
    ],
    noteLabel: "依据",
    note: "产品定位 + 公开采购信号 · 置信度 中 · 群体可在工作台内继续精调",
  },

  founder: {
    label: "为什么是我们",
    headlineLine1: "让每一个中国团队,",
    headlineLine2: "都能打进全球市场。",
    paragraphs: [
      "我在多个硅谷明星创业公司做过 B2B 销售冠军,也在 Uber 这样的 500 强负责过客户增长,还在中国带过一支近 20 人的出海团队,亲手把中国公司的产品卖给了海外大客户。我见过最好的打法,也踩过所有的坑。",
      "但我更清楚一件事:海外市场从来不该只属于那些请得起本地销售老兵的大公司。当 AI 能接手最重的「调研 + 判断 + 触达」,出海的门槛会被彻底拉平——外贸老板、第一次做海外的创业者、几个人的小团队,都能像本地老兵一样把客户一个个谈进来。",
    ],
    punchline: "这就是子午纪在做的事:让出海,变成一件全民都能做成的事。",
    name: "周玉林",
    role: "创始人 & CEO",
  },

  waitlist: {
    label: "内测席位 · 每月少量",
    headline: "排队进入拓客引擎内测",
    body: "留下联系方式,上线时第一批通知你。现有出海查用户优先。",
    cta: "立即开始",
    note: "现有出海查用户优先。",
  },

  howItWorks: {
    label: "开始使用",
    headlineMain: "出海查 AI · 微信搜一搜,",
    headlineFaded: "就能拿到第一份调研报告。",
    steps: [
      {
        title: allowSignup ? "微信搜索「出海查AI」" : "登录工作台",
        description: allowSignup
          ? "在微信里搜索小程序「出海查AI」,无需下载 App,无需注册账号,几秒打开。"
          : "输入邮箱并验证码登录,即可进入工作台,开始第一次海外客户调研。",
      },
      {
        title: "输入一家海外公司名",
        description:
          "从英国 Tesco 到德国 MediaHaus,输入你想查的海外公司,几秒拿到工商信息、经营信号和 AI 深度报告。",
      },
      {
        title: "上传你的产品资料",
        description:
          "上传产品资料后,报告自动结合你的品类给出切入路径建议——这家客户该怎么谈、需要哪些认证、首打哪条线。",
      },
      {
        title: "一键触达完整六步拓客流程",
        description:
          "报告结尾的按钮,直通完整六步拓客引擎——从看懂一家公司,到把整个海外市场打下来。",
      },
    ],
    cta: "立即开始",
    ctaGithub: "在 GitHub 上查看",
  },

  openSource: {
    label: "出海查 AI",
    headlineLine1: "微信扫一扫,",
    headlineLine2: "免费开始第一次调研。",
    description:
      "无需下载、无需注册。微信搜索「出海查AI」,输入一家海外公司,几秒拿到第一份 AI 深度报告——包括你的产品该从哪里切入。",
    cta: "微信搜索「出海查AI」",
    highlights: [
      { title: "无需下载 App", description: "微信小程序即开即用。国内网络直接访问,无需翻墙、无需配置。" },
      { title: "首次调研免费", description: "不限制公司大小、不限地区。从英国零售巨头到德国渠道连锁都能查。" },
      { title: "权威数据来源", description: "对接 Companies House、Kantar、GLEIF 等权威数据库。每条结论标注来源与置信度。" },
      { title: "结合你的业务", description: "上传产品资料后,报告自动结合你的品类,给出切入路径建议——这家客户该怎么谈。" },
    ],
  },

  faq: {
    label: "常见问题",
    headline: "问与答。",
    items: [
      {
        question: "出海查 AI 是什么?和拓客引擎是什么关系?",
        answer:
          "出海查 AI 是子午纪的微信小程序,免费提供海外公司调研:工商信息、经营信号、AI 深度报告。拓客引擎是付费产品,在出海查报告基础上,把整个六步拓客流程跑通——从市场调研、构建 ICP、查找联系人,到撰写开发信、排期发送、回复处理。",
      },
      {
        question: "需要翻墙吗?需要下载 App 吗?",
        answer:
          "不需要。出海查 AI 是微信小程序,国内网络直接打开,即开即用。拓客引擎内测席位通过网页工作台使用,后续会有桌面端。",
      },
      {
        question: "和直接用领英、用海外黄页查公司有什么区别?",
        answer:
          "出海查 AI 不只是给你一张公司名片。它结合权威数据库 + AI 深度调研,告诉你这家公司的采购规模、决策链、合作可行性,并结合你的产品品类给出切入路径建议——这家客户该怎么谈、需要哪些认证、首打哪条线。",
      },
      {
        question: "拓客引擎现在能用吗?怎么排队?",
        answer:
          "拓客引擎目前内测排队中,每月少量席位。留下联系方式,上线时第一批通知你。现有出海查用户优先。",
      },
      {
        question: "我的产品资料安全吗?会不会被用来训练模型?",
        answer:
          "你的产品资料仅用于生成针对你业务的调研报告与切入路径建议,不会用于训练模型,也不会分享给第三方。",
      },
      {
        question: "支持哪些海外市场?哪些行业?",
        answer:
          "已服务智能硬件、软件服务、AI 出海等领域的头部企业。覆盖北美、欧洲、东南亚、中东等主要海外市场。如果你的品类或目标市场不在演示用例里,欢迎联系商务沟通定制方案。",
      },
    ],
  },

  footer: {
    tagline:
      "让每一个中国团队,跟海外本地老兵一样能打。让天下没有难做的海外生意。",
    cta: "立即开始",
    groups: {
      product: {
        label: "产品",
        links: [
          { label: "出海查 AI", href: "#product" },
          { label: "拓客流程", href: "#workflow" },
          { label: "为什么是我们", href: "#founder" },
          { label: "下载", href: "/download" },
        ],
      },
      resources: {
        label: "资源",
        links: [
          { label: "API", href: githubUrl },
          { label: "X (Twitter)", href: "https://x.com/MeridianOSAI" },
        ],
      },
      company: {
        label: "关于",
        links: [
          { label: "关于我们", href: "/about" },
          { label: "开源", href: "#open-source" },
          { label: "联系商务", href: "/contact-sales" },
          { label: "GitHub", href: githubUrl },
        ],
      },
    },
    copyright: "© {year} MeridianOS, Inc. 保留所有权利。",
  },

  about: {
    title: "关于子午纪",
    nameLine: {
      prefix: "MeridianOS ——",
      mul: "Meridian",
      tiplexed: "OS · ",
      i: "子",
      nformationAnd: "午",
      c: "纪",
      omputing: " · ",
      a: "出",
      gent: "海拓客操作系统。",
    },
    paragraphs: [
      "子午纪(MeridianOS)的名字取自「子午线」——地球上的经线,把世界从东到西连成一张网。中国团队出海,缺的从来不是产品力,而是一张能把海外买家一个个找出来、谈进去的网。",
      "过去,这张网只有请得起本地销售老兵的大公司才铺得起。飞去当地、雇本地队、押注单一市场——押错了几百万打水漂。子午纪在做的事,是用 AI 接手最重的「调研 + 判断 + 触达」,把出海的门槛彻底拉平。",
      "我们相信,海外市场从来不该只属于大公司。外贸老板、第一次做海外的创业者、几个人的小团队,都应该能像本地老兵一样,把海外客户一个个谈进来。",
      "这就是子午纪——一个覆盖出海拓客全程的工作台。从市场调研、构建 ICP、查找联系人,到撰写开发信、排期发送、回复处理,每一步都有 AI 跑腿、给判断,再由你确认放行。",
      "子午纪由 MeridianOS, Inc.(Delaware C-Corp)运营。我们在中国有一支近 20 人的团队,创始人曾在多个硅谷明星创业公司做过 B2B 销售冠军,也在 Uber 这样的 500 强负责过客户增长。",
    ],
    cta: "在 GitHub 上查看",
  },

  download: {
    hero: {
      macArm64: {
        title: "MeridianOS for macOS",
        sub: "Apple Silicon · 内置 daemon,无需配置",
        primary: "下载 (.dmg)",
        altZip: "或下载 .zip",
      },
      macIntel: {
        title: "MeridianOS for macOS",
        sub: "Intel · 内置守护进程,无需配置",
        primary: "下载 (.dmg)",
        altZip: "或下载 .zip",
      },
      winX64: {
        title: "MeridianOS for Windows",
        sub: "内置 daemon,无需配置",
        primary: "下载 (.exe)",
      },
      winArm64: {
        title: "MeridianOS for Windows",
        sub: "ARM · 内置 daemon,无需配置",
        primary: "下载 (.exe)",
      },
      linux: {
        title: "MeridianOS for Linux",
        sub: "内置 daemon,无需配置",
        primary: "下载 AppImage",
        altFormats: "或 .deb / .rpm",
      },
      unknown: {
        title: "选择你的平台",
        sub: "下方是所有支持的安装包。",
      },
      safariMacHint: "在 Intel Mac 上?请在下方选择 Intel 版本。",
      archFallbackHint: "架构不对?下方是所有可选格式。",
    },
    allPlatforms: {
      title: "所有平台",
      macArm64Label: "macOS · Apple Silicon",
      macX64Label: "macOS · Intel",
      winX64Label: "Windows · x64",
      winArm64Label: "Windows · ARM64",
      linuxX64Label: "Linux · x64",
      linuxArm64Label: "Linux · ARM64",
      formatDmg: ".dmg",
      formatZip: ".zip",
      formatExe: ".exe",
      formatAppImage: ".AppImage",
      formatDeb: ".deb",
      formatRpm: ".rpm",
      unavailable: "暂不可用",
    },
    cli: {
      title: "想用 CLI?",
      sub: "适合服务器、远程开发机、无图形界面环境。底层 daemon 与 Desktop 相同,通过终端安装。",
      installLabel: "安装",
      startLabel: "启动 daemon",
      sshNote: "已经在服务器上?通过 SSH 执行同样的命令即可。",
      copyLabel: "复制",
      copiedLabel: "已复制",
    },
    cloud: {
      title: "Cloud runtime(等待名单)",
      sub: "我们将为你托管 runtime,目前尚未上线——留下邮箱,上线后通知你。",
    },
    footer: {
      releaseNotes: "v{version} 更新内容",
      allReleases: "查看所有版本",
      currentVersion: "当前版本:{version}",
      versionUnavailable: "版本获取失败——请前往 GitHub 查看",
    },
  },
  contactSales: {
    pageTitle: "联系商务",
    pageDescription:
      "了解如何在你的团队中落地子午纪的 AI 出海拓客工作流。",
    eyebrow: "联系商务",
    title: "先了解你的需求",
    subtitle: "在正式沟通之前,让我们为你定制最合适的方案。",
    notice: {
      badge: "系统仅识别企业邮箱域名。",
      body: "来自个人邮箱(例如 @gmail.com、@outlook.com)的请求不会被处理。",
    },
    fields: {
      firstName: "名",
      lastName: "姓",
      businessEmail: "企业邮箱",
      businessEmailHint: "请使用真实的企业邮箱域名,方便我们后续与你联系。",
      companyName: "公司名称",
      companySize: "公司规模",
      countryRegion: "国家 / 地区",
      useCase: "你打算如何使用子午纪或与我们合作?",
      goals: "你的目标或挑战",
      goalsHint:
        "告诉我们你希望借助子午纪达成什么目标,或正在面临的挑战。信息越详细,我们越能给到合适的支持。",
      selectPlaceholder: "请选择",
      submit: "提交",
      submitting: "正在提交…",
    },
    companySizes: [
      { value: "1-10", label: "1 – 10 人" },
      { value: "11-50", label: "11 – 50 人" },
      { value: "51-200", label: "51 – 200 人" },
      { value: "201-500", label: "201 – 500 人" },
      { value: "501-1000", label: "501 – 1,000 人" },
      { value: "1000+", label: "1,000 人以上" },
    ],
    useCases: [
      { value: "evaluate", label: "正在为团队评估子午纪" },
      { value: "adopt_team", label: "希望在团队 / 公司内推广使用" },
      { value: "self_host", label: "需要在自有基础设施上自托管" },
      { value: "integrate", label: "希望与现有工具集成" },
      { value: "partner", label: "合作 / 渠道合作咨询" },
      { value: "other", label: "其他" },
    ],
    countries: [
      "中国大陆", "中国香港", "中国澳门", "中国台湾", "新加坡", "马来西亚",
      "印度尼西亚", "泰国", "越南", "菲律宾", "日本", "韩国", "印度",
      "阿联酋", "沙特阿拉伯", "以色列", "土耳其", "美国", "加拿大", "英国",
      "德国", "法国", "荷兰", "瑞典", "瑞士", "西班牙", "意大利", "爱尔兰",
      "挪威", "丹麦", "芬兰", "比利时", "葡萄牙", "澳大利亚", "新西兰",
      "南非", "巴西", "墨西哥", "阿根廷", "智利", "其他",
    ],
    consent: {
      intro:
        "MeridianOS, Inc. 尊重你的隐私。我们仅会将你的个人信息用于管理账户,以及提供你所请求的产品或服务。我们偶尔也希望与你分享产品更新、最佳实践或行业洞察,如果你愿意接收,请在下方勾选。",
      outreach:
        "我希望接收来自 MeridianOS, Inc. 的一对一沟通,包括服务更新、支持咨询以及业务相关的跟进。",
      updates: "我希望接收子午纪的产品更新、洞察以及活动邀请。",
      unsubscribe:
        "你可以随时取消订阅我们的邮件。关于我们如何处理你的数据以及隐私权利,请参阅",
      submitConsent:
        "点击「提交」即表示你同意 MeridianOS, Inc. 存储并处理你提交的信息,以便交付你请求的内容。",
      privacyLinkLabel: "隐私政策。",
      privacyLinkHref: "/about",
    },
    success: {
      title: "已收到,谢谢!",
      message:
        "子午纪团队会在三个工作日内回复你。在此期间,欢迎查看我们的文档,或在 GitHub 上为我们点个 Star。",
      cta: "返回首页",
    },
    errors: {
      generic: "提交失败,请稍后再试。",
      rateLimit: "该邮箱近期已提交多次,请稍后再试。",
      freeEmail: "请使用企业邮箱——免费邮箱(gmail、outlook 等)暂不接受。",
      invalidEmail: "邮箱地址格式不正确。",
    },
  },
  };
}
