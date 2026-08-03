import { githubUrl } from "../components/shared";
import { createEnDict } from "./en";
import type { LandingDict } from "./types";

export function createJaDict(allowSignup: boolean): LandingDict {
  const base = createEnDict(allowSignup);

  return {
    ...base,
    header: {
      cta: "始める",
      dashboard: "ワークベンチを開く",
      navigation: "メインナビゲーション",
      openMenu: "メニューを開く",
      closeMenu: "メニューを閉じる",
    },
    hero: {
      eyebrow: "B2B アウトバウンド · 中国 → グローバル",
      headlineLine1: "インバウンドを待つのをやめる。",
      headlineLine2: "海外バイヤーを自ら開拓する。",
      subheading:
        "広告なし、問い合わせ待ちなし — 自然言語でエージェントを指揮し、海外 B2B バイヤーを自ら特定して、一人ずつ開拓する。新規チームは足場作りに、熟練チームはプレイブックの再現に使う。",
      cta: "出海查 AI · 公開中",
      ctaSecondary: "完整な拓客フローを見る",
      downloadDesktop: "デスクトップ版をダウンロード",
      talkToSales: "営業に相談",
      worksWith: "導入済み",
  },
    stats: {
      items: [
        { value: "10+", label: "リーディング企業に導入" },
        { value: "200+", label: "深掘り調査レポート納品" },
        { value: "6 ステップ", label: "拓客フル自動ループ" },
      ],
      trustedBy:
        "スマートハードウェア · ソフトウェアサービス · AI グローバル展開 のリーディングチームで導入済み",
    },
    liveDemo: {
      badge: "LIVE DEMO",
      title: "デモ中",
      subtitle: "エージェントが海外バイヤーをどう一步步開拓するかを見る",
      browserBar: "meridian.app / 外連ワークベンチ LIVE",
      steps: [
        { id: "research", title: "独国チャネル掘り下げ research", detail: "オフライン家電チェーンを特定 · 首攻は NRW / バイエルン" },
        { id: "targets", title: "チェーンバイヤーを特定 targets", detail: "MediaHaus ほか3社、現在サプライヤー募集中" },
        { id: "outreach", title: "開拓メール作成 outreach", detail: "各社ごとにカスタマイズ、それぞれの調達シグナルを引用" },
        { id: "reply", title: "高意向インバウンド受信 reply", detail: "MediaHaus 調達担当から返信、受信箱へ整理" },
      ],
      standby: "全フロー完了 · 次の指示をお待ちしています",
      analyzing: "独国小売チャネル · ドリルダウン分析中…",
      analysisItems: [
        { label: "オフライン家電チェーン", percent: 52 },
        { label: "EC プラットフォーム小売", percent: 33 },
        { label: "地域ディストリビューター", percent: 15 },
      ],
      analysisNote:
        "◆ 首攻エリア **NRW / バイエルン** · 店舗密度最高、集購集中 · CleanMax 値上げ後の半額帯が空席",
      locked: "チェーンバイヤーを特定・検証済み",
      lockedTargets: [
        { initials: "MH", name: "MediaHaus", meta: "家電チェーン · NRW · サプライヤー募集中", tag: "✓ 高マッチ" },
        { initials: "EX", name: "Expert SE", meta: "家電チェーン · 全独 · 集購決済", tag: "✓ 高マッチ" },
        { initials: "EP", name: "EP:Group", meta: "家電チェーン · バイエルン · 店舗密集", tag: "✓ 高マッチ" },
      ],
      drafting: "EN · 生成済み",
      draftHeader: "TO: S. BRANDT · Buyer · MEDIAHAUS",
      draftBody:
        "Dear Ms. Brandt,\n\nSaw you're sourcing robot vacuums for the fall lineup — with CleanMax's price hike leaving the mid-tier open,\n\nwe ship from our DE warehouse in 12 days…",
      replyTitle: "高意向インバウンド · 受信箱返信処理",
      replyMeta: "S. Brandt · MediaHaus が返信しました",
      replyBody:
        "\"Interesting. Can you send your catalog and MOQ for the S9?\"",
      replyTag: "意向レベル: 高 · NRW 家電チェーン調達",
      replyFooter: "✓ 受信箱に整理 · 高意向をピン留め",
    },
    comparison: {
      label: "同じ仕事 · 二つの生き方",
      headlineLine1: "海外チャネルに売り込む —",
      headlineLine2: "かつては賭けだった。今は一言で。",
      pastLabel: "過去 · 高コスト・低速・リスク大",
      presentLabel: "現在 · 高速・低コスト・可逆",
      past: [
        { question: "始め方", answer: "現地へ飛び、ローカルチームを雇う" },
        { question: "成果までの時間", answer: "半年かけてようやく手探り" },
        { question: "試せる市場", answer: "予算は一つに賭けるのみ" },
        { question: "賭けに外れたら", answer: "数百万が水の泡" },
      ],
      present: [
        { question: "始め方", answer: "エージェントに一言" },
        { question: "成果までの時間", answer: "当日に最初のバイヤー" },
        { question: "試せる市場", answer: "好きなだけ" },
        { question: "合わなければ", answer: "別市場へ切り替えて再実行" },
      ],
    },
    features: {
      teammates: {
        label: "PRODUCT 01 · 公開中",
        title: "出海查 AI · WeChat ミニプログラム",
        description:
          "国内ビジネスでは企査査を使う。グローバル展開では出海查を使う。海外企業名を入力するだけで、登記情報、経営シグナル、AI 深掘りレポート — あなたの製品をどこから切入るべきかまで — が手に入る。",
        cards: [
          { title: "登記情報検証", description: "登記情報、取締役構成、存続状態 — 公式データベースから取得。" },
          { title: "AI 深掘り調査", description: "調達動向、チャネル構造、意思決定者 — 全ての結論に出典と信頼度を明記。" },
          { title: "切入路径の提案", description: "製品資料をアップロードすると、レポートが直接教えてくれる:この顧客とどう商談すべきか。" },
        ],
      },
      autonomous: {
        label: "グローバル企業",
        title: "一查便知。",
        description:
          "主要な海外市場の企業データベースをカバー。英国 Tesco から独国 MediaHaus まで、社名を入力すれば数秒で登記情報、調達規模、合作可行性が手に入る — AI が同時にあなたの製品を考慮して切入路径を提示。",
        cards: [
          { title: "公式データベース連携", description: "Companies House、Kantar、GLEIF などの権威ある情報源、追跡可能。" },
          { title: "あなたの業務に最適化", description: "製品資料をアップロードすると、レポートが自動的にあなたのカテゴリーを考慮した切入提案を生成。" },
          { title: "深掘りレポート · ワンクリック生成", description: "合作可行性 / 調達情報 / 切入路径 / リスク — 4つの次元を一度に提示。" },
        ],
      },
      skills: {
        label: "WeChat エントリー",
        title: "WeChat でスキャン、最初の調査を無料で",
        description:
          "アプリ不要、アカウント登録不要。WeChat で「出海查AI」を検索し、海外企業名を入力するだけで、数秒で最初の AI 深掘りレポートが手に入る。最初の1件は無料。",
        cards: [
          { title: "WeChat で「出海查AI」を検索", description: "WeChat を開き、ミニプログラム名を検索するだけで、最初の企業調査を開始。" },
          { title: "最初の調査は無料", description: "企業サイズ・地域は問わない。英国小売大手から独国チェーンバイヤーまで調査可能。" },
          { title: "レポート末尾からワンクリック", description: "レポート末尾のボタンが完全な6ステップ拓客フローに直通 — 一社の理解から市場全体の制覇へ。" },
        ],
      },
      runtimes: {
        label: "デモユースケース",
        title: "TESCO PLC · 公開情報デモ",
        description:
          "デモデータは TESCO PLC の公開情報に基づく。完全な出海查 深掘りレポートがどのようなものかを展示 — 登記情報、調達規模、あなたのカテゴリーを考慮した切入提案まで。",
        cards: [
          { title: "登記情報検証", description: "登録番号 00445790 · Active 存続 · Companies House · 市場シェア 27.4%。" },
          { title: "調達情報", description: "ベビー用品カテゴリー年間調達 £3–5 億 · サプライヤー要件 BSCI / UKCA。" },
          { title: "AI 切入路径の提案", description: "まず Own Label OEM で切入り、その後ブランド入場を商談 — BSCI/SEDEX + UKCA が必要。" },
        ],
      },
    },
    transition: {
      headlineLine1: "一社を理解するのは入り口に過ぎない。",
      headlineLine2: "次は、市場全体を*制覇する*。",
      body: "出海查 レポート末尾のボタンが AI 自動アウトリーチを起動 — 完全な6ステップ拓客フローへ ↓",
    },
    workflow: {
      label: "PRODUCT 02 · 内測排队中",
      headline: "拓客エンジン:市場調査から、成約一歩手前まで(内測)",
      subheading:
        "単なるメール一斉送信ツールではない。出海拓客の全行程をカバーするワークベンチ — 各ステップで AI が走り、判断し、あなたが承認して進める。",
      steps: [
        {
          id: "research_market",
          code: "STEP 01 / RESEARCH_MARKET",
          title: "01 市場調査",
          detail: "市場を特定後、攻めるべき地域とチャネルへ自動ドリルダウン",
          panel: {
            title: "3つの候補市場、ランキング済み",
            subtitle: "首攻市場のご確認をお待ちしています",
            body: "Lumo S9 ロボット掃除機 · 代理",
            bullets: [
              "DE 独国 · 87 · 成熟小売 · CR5 62% 集中 · 半額帯が空席",
              "FR 仏国 · 73 · チャネル分散 · ディストリビューション中心",
              "ND 北欧 · 69 · 高客単価 · 意思決定周期が長い",
            ],
          },
        },
        {
          id: "build_icp",
          code: "STEP 02 / BUILD_ICP",
          title: "02 ICP 構築",
          detail: "市場を具体的なバイヤーグループに分割し、規模と意思決定チェーンを算出",
          panel: {
            title: "独国市場、2つのバイヤーグループに分割",
            subtitle: "主力グループのご選定をお待ちしています",
            body: "根拠: 製品ポジショニング + 公開調達シグナル",
            bullets: [
              "グループ ① · 主力:小売チェーン · カテゴリー調達ディレクター · 規模 ~1,800 · チェーン: ディレクター → カテゴリー VP · 難易度 中",
              "グループ ② · 補充:独立家電ディーラー · 責任者 · 規模 ~3,200 · チェーン: 単点判断 · 難易度 低",
            ],
          },
        },
        {
          id: "find_contacts",
          code: "STEP 03 / FIND_CONTACTS",
          title: "03 連絡先検索",
          detail: "意思決定者を抽出、メールを検証、送信可能なリストを提供",
          panel: {
            title: "意思決定者 + 送達可能メール、検証済み",
            subtitle: "公開ソースから抽出 · 重複排除済み",
            body: "メール SMTP 検証合格",
            bullets: [
              "意思決定者の氏名 + 職位 + 部門",
              "メールは SMTP 検証済み、無効アドレスをフィルター",
              "情報源と更新日時を明記",
            ],
          },
        },
        {
          id: "draft_sequence",
          code: "STEP 04 / DRAFT_SEQUENCE",
          title: "04 シーケンス作成",
          detail: "バイヤーごとに専用の英語開拓メールを作成、調達シグナルを引用",
          panel: {
            title: "各バイヤー、専用のメール",
            subtitle: "初回 + 3ラウンドフォローアップ",
            body: "各バイヤーの調達シグナルから生成 · テンプレート一斉送信ではない",
            bullets: [
              "バイヤーごとにカスタマイズ、変数置換ではない",
              "初回メールは該当バイヤーの実際の調達シグナルを引用",
              "3ラウンドのフォローアップは意思決定周期に応じて自動編成",
            ],
          },
        },
        {
          id: "schedule_send",
          code: "STEP 05 / SCHEDULE_SEND",
          title: "05 配信・送信",
          detail: "相手のタイムゾーンの就業時間に配信、ドメインウォームアップとレート制限",
          panel: {
            title: "相手の朝9時に、時間通りに到着",
            subtitle: "タイムゾーン配信 · レート制限送信",
            body: "送信レピュテーション管理 · 深夜の迷惑なし",
            bullets: [
              "相手のローカル就業時間に配信",
              "ドメインウォームアップ曲線、ブラックリスト回避",
              "初回とフォローアップを最適間隔で編成",
            ],
          },
        },
        {
          id: "handle_reply",
          code: "STEP 06 / HANDLE_REPLY",
          title: "06 返信処理",
          detail: "返信を自動で分級して受信箱へ、高意向はピン留め通知",
          panel: {
            title: "受信箱自動分級、高意向をピン留め",
            subtitle: "返信を自動で意向分級",
            body: "あなたの出番の時だけ通知 · 本当に重要な数件だけを処理",
            bullets: [
              "意向分級: 高 / フォローアップ / 無効",
              "高意向はピン留め、引き継ぎを促す",
              "フォローアップ型は次ラウンドに自動キューイング",
            ],
          },
        },
      ],
      noteLabel: "根拠",
      note: "製品ポジショニング + 公開調達シグナル · 信頼度: 中 · グループはワークベンチ内で精調可能",
    },
    founder: {
      label: "なぜ私たちか",
      headlineLine1: "すべての中国チームが、",
      headlineLine2: "グローバル市場に打って出られるように。",
      paragraphs: [
        "私は複数のシリコンバレー注目スタートアップで B2B セールスチャンピオンを務め、Uber のような Fortune 500 で顧客成長を担当し、中国で20人近い出海チームを率いて、中国企業の製品を海外大手に直接売り込んできました。最良のプレイブックも、すべての落とし穴も知っています。",
        "しかし一つだけ明確なことがあります — 海外市場は、ローカルの営業ベテランを雇える大企業だけのものではあってはならない。AI が最も重い「調査 + 判断 + アウトリーチ」を引き受ける時、出海のハードルは完全に平準化されます — 外貿オーナー、初めて海外に挑む起業家、小規模チームの誰もが、ローカルベテランのようにバイヤーを一人ずつ開拓できるようになります。",
      ],
      punchline: "それが MeridianOS のやっていること — 出海を、誰にでもできるものにする。",
      name: "Zhou Yulin",
      role: "創業者 & CEO",
    },
    waitlist: {
      label: "内測席 · 毎月少量",
      headline: "拓客エンジン内測のウェイティングリストへ",
      body: "連絡先を残していただければ、ローンチ時に最初にお知らせします。既存の出海查ユーザーが優先されます。",
      cta: "始める",
      note: "既存の出海查ユーザーが優先されます。",
    },
    howItWorks: {
      label: "始める",
      headlineMain: "出海查 AI · WeChat で検索,",
      headlineFaded: "最初の調査レポートを取得。",
      steps: [
        {
          title: allowSignup ? "WeChat で「出海查AI」を検索" : "ワークベンチにログイン",
          description: allowSignup
            ? "WeChat でミニプログラム「出海查AI」を検索 — アプリ不要、アカウント不要、数秒で開始。"
            : "メールと認証コードでログインし、ワークベンチに入って最初の海外顧客調査を開始。",
        },
        {
          title: "海外企業名を入力",
          description:
            "英国 Tesco から独国 MediaHaus まで — 調査したい海外企業を入力。数秒で登記情報、経営シグナル、AI 深掘りレポートが手に入る。",
        },
        {
          title: "製品資料をアップロード",
          description:
            "アップロード後、レポートは自動的にあなたのカテゴリーを考慮した切入路径の提案を生成 — この顧客とどう商談すべきか、どの認証が必要か、どのラインを首攻にするか。",
        },
        {
          title: "ワンクリックで完全な6ステップ拓客フローへ",
          description:
            "レポート末尾のボタンが完全な6ステップ拓客エンジンに直通 — 一社の理解から海外市場全体の制覇へ。",
        },
      ],
      cta: "始める",
      ctaGithub: "GitHub で見る",
    },
    openSource: {
      label: "出海查 AI",
      headlineLine1: "WeChat でスキャン,",
      headlineLine2: "最初の調査を無料で。",
      description:
        "アプリ不要、登録不要。WeChat で「出海查AI」を検索し、海外企業名を入力するだけで、数秒で最初の AI 深掘りレポートが手に入る — あなたの製品をどこから切入るべきかまで。",
      cta: "WeChat で「出海查AI」を検索",
      highlights: [
        { title: "アプリ不要", description: "WeChat ミニプログラムで即起動。国内ネットワークで直接アクセス、VPN 不要、設定不要。" },
        { title: "最初の調査は無料", description: "企業サイズ・地域は問わない。英国小売大手から独国チェーンバイヤーまで調査可能。" },
        { title: "権威あるデータソース", description: "Companies House、Kantar、GLEIF などの権威あるデータベース。全ての結論に出典と信頼度を明記。" },
        { title: "あなたの業務に最適化", description: "製品資料をアップロードすると、レポートが自動的にあなたのカテゴリーを考慮した切入路径の提案を生成 — この顧客とどう商談すべきか。" },
      ],
    },
    faq: {
      label: "よくある質問",
      headline: "問と答。",
      items: [
        {
          question: "出海查 AI とは何ですか?拓客エンジンとはどう関係しますか?",
          answer:
            "出海查 AI は MeridianOS の WeChat ミニプログラムで、海外企業の調査を無料で提供します:登記情報、経営シグナル、AI 深掘りレポート。拓客エンジンは有料製品で、出海查 レポートの上に完全な6ステップ拓客フローを構築します — 市場調査、ICP 構築、連絡先検索から、開拓メール作成、配信送信、返信処理まで。",
        },
        {
          question: "VPN は必要ですか?アプリのダウンロードは?",
          answer:
            "不要です。出海查 AI は WeChat ミニプログラムで、国内ネットワークで直接開けます。拓客エンジン内測席はウェブワークベンチで利用、デスクトップ版は後日提供予定。",
        },
        {
          question: "LinkedIn や海外の企業名鑑と何が違いますか?",
          answer:
            "出海查 AI は単なる企業名片ではありません。権威あるデータベースと AI 深掘り調査を組み合わせ、その企業の調達規模、意思決定チェーン、合作可行性を教えてくれます。さらにあなたの製品カテゴリーを考慮した切入路径の提案を生成 — この顧客とどう商談すべきか、どの認証が必要か、どのラインを首攻にするか。",
        },
        {
          question: "拓客エンジンは今すぐ使えますか?どうやって申込みますか?",
          answer:
            "拓客エンジンは現在内部テスト中で、毎月少量の席です。連絡先を残していただければ、ローンチ時に最初にお知らせします。既存の出海查ユーザーが優先されます。",
        },
        {
          question: "製品資料は安全ですか?モデルの訓練に使われますか?",
          answer:
            "あなたの製品資料は、あなたの業務向けの調査レポートと切入路径の提案を生成するためだけに使われます。モデルの訓練には使われず、第三者と共有されることもありません。",
        },
        {
          question: "どの海外市場・業種に対応していますか?",
          answer:
            "スマートハードウェア、ソフトウェアサービス、AI グローバル展開などのリーディング企業に導入済みです。北米、欧州、東南アジア、中東などの主要海外市場をカバー。あなたのカテゴリーやターゲット市場がデモにない場合、営業までご相談ください。",
        },
      ],
    },
    footer: {
      tagline:
        "すべての中国チームが、海外のローカルベテランのように戦えるように。海外ビジネスを、誰にでもできるものに。",
      cta: "始める",
      groups: {
        product: {
          label: "製品",
          links: [
            { label: "出海查 AI", href: "#product" },
            { label: "拓客フロー", href: "#workflow" },
            { label: "なぜ私たちか", href: "#founder" },
            { label: "ダウンロード", href: "/download" },
          ],
        },
        resources: {
          label: "リソース",
          links: [
            { label: "API", href: githubUrl },
            { label: "X (Twitter)", href: "https://x.com/MeridianOSAI" },
          ],
        },
        company: {
          label: "会社",
          links: [
            { label: "私たちについて", href: "/about" },
            { label: "オープンソース", href: "#open-source" },
            { label: "営業へのご相談", href: "/contact-sales" },
            { label: "GitHub", href: githubUrl },
          ],
        },
      },
      copyright: "© {year} MeridianOS, Inc. All rights reserved.",
    },
    about: {
      title: "MeridianOS について",
      nameLine: {
        prefix: "MeridianOS — ",
        mul: "Meridian",
        tiplexed: "OS · ",
        i: "経",
        nformationAnd: "線",
        c: "が",
        omputing: "東西を",
        a: "つな",
        gent: "ぐネットワーク。",
      },
      paragraphs: [
        "MeridianOS(子午紀)の名前は「子午線」— 地球上の経線で、世界を東から西へと一つのネットワークで結ぶ線 — に由来します。中国チームの出海に欠けているのは決して製品力ではなく、海外バイヤーを一人ずつ見つけ出し、開拓していくネットワークです。",
        "過去、このネットワークを構築できたのは、ローカルの営業ベテランを雇える大企業だけでした。現地へ飛び、ローカルチームを雇い、単一市場に賭ける — 賭けに外れれば数百万が水の泡。MeridianOS がやっているのは、AI に最も重い「調査 + 判断 + アウトリーチ」を引き受けさせ、出海のハードルを完全に平準化することです。",
        "私たちは信じています — 海外市場は決して大企業だけのものではあってはならない。外貿オーナー、初めて海外に挑む起業家、小規模チームの誰もが、ローカルベテランのようにバイヤーを一人ずつ開拓できるべきです。",
        "それが MeridianOS — 出海拓客の全行程をカバーするワークベンチです。市場調査、ICP 構築、連絡先検索から、開拓メール作成、配信送信、返信処理まで、各ステップで AI が走り、判断し、あなたが承認して進める。",
        "MeridianOS は MeridianOS, Inc.(Delaware C-Corp)が運営しています。中国には20人近いチームがあり、創業者は複数のシリコンバレー注目スタートアップで B2B セールスチャンピオンを務め、Uber のような Fortune 500 で顧客成長を担当した経験を持ちます。",
      ],
      cta: "GitHub で見る",
    },
    download: {
      hero: {
        macArm64: {
          title: "MeridianOS for macOS",
          sub: "Apple Silicon · デーモン同梱、設定不要",
          primary: "ダウンロード (.dmg)",
          altZip: "または .zip",
        },
        macIntel: {
          title: "MeridianOS for macOS",
          sub: "Intel · デーモン同梱、設定不要",
          primary: "ダウンロード (.dmg)",
          altZip: "または .zip",
        },
        winX64: {
          title: "MeridianOS for Windows",
          sub: "デーモン同梱、設定不要",
          primary: "ダウンロード (.exe)",
        },
        winArm64: {
          title: "MeridianOS for Windows",
          sub: "ARM · デーモン同梱、設定不要",
          primary: "ダウンロード (.exe)",
        },
        linux: {
          title: "MeridianOS for Linux",
          sub: "デーモン同梱、設定不要",
          primary: "AppImage をダウンロード",
          altFormats: "または .deb / .rpm",
        },
        unknown: {
          title: "プラットフォームを選択",
          sub: "対応する全インストーラーは以下の通りです。",
        },
        safariMacHint: "Intel Mac の場合?下の Intel 版をお選びください。",
        archFallbackHint: "アーキテクチャが違う?全フォーマットは以下の通りです。",
      },
      allPlatforms: {
        title: "全プラットフォーム",
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
        unavailable: "利用不可",
      },
      cli: {
        title: "CLI をお使いですか?",
        sub: "サーバー、リモート開発機、ヘッドレス環境向け。Desktop と同じデーモンをターミナルからインストール。",
        installLabel: "インストール",
        startLabel: "デーモン起動",
        sshNote: "サーバー上ですか?SSH で同じコマンドを実行できます。",
        copyLabel: "コピー",
        copiedLabel: "コピー済み",
      },
      cloud: {
        title: "Cloud runtime(ウェイティングリスト)",
        sub: "runtime をホストします — 現在未公開。メールを残していただければ、公開時にお知らせします。",
      },
      footer: {
        releaseNotes: "v{version} リリースノート",
        allReleases: "全リリースを見る",
        currentVersion: "現在のバージョン:{version}",
        versionUnavailable: "バージョン取得失敗 — GitHub で確認してください",
      },
    },
    contactSales: {
      pageTitle: "営業へのお問い合わせ",
      pageDescription:
        "MeridianOS の AI 出海拓客ワークフローをあなたのチームでどう展開するかをご紹介します。",
      eyebrow: "営業へのお問い合わせ",
      title: "まずはご要望をお聞かせください",
      subtitle: "正式なご相談の前に、最適なプランをご提案します。",
      notice: {
        badge: "システムは企業メールドメインのみ認識します。",
        body: "個人メール(@gmail.com、@outlook.com など)からのリクエストは処理されません。",
      },
      fields: {
        firstName: "名",
        lastName: "姓",
        businessEmail: "企業メール",
        businessEmailHint: "フォローアップのため、実際の企業メールドメインをご使用ください。",
        companyName: "会社名",
        companySize: "会社規模",
        countryRegion: "国 / 地域",
        useCase: "MeridianOS をどのようにご使用予定ですか?",
        goals: "目標や課題",
        goalsHint:
          "MeridianOS で達成したいこと、直面している課題をお聞かせください。詳細なほど、より適切なサポートを提供できます。",
        selectPlaceholder: "選択してください",
        submit: "送信",
        submitting: "送信中…",
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
        { value: "evaluate", label: "チーム向けに MeridianOS を評価中" },
        { value: "adopt_team", label: "チーム / 社内で展開したい" },
        { value: "self_host", label: "自社インフラで自己ホストしたい" },
        { value: "integrate", label: "既存ツールと連携したい" },
        { value: "partner", label: "パートナーシップ / チャネル相談" },
        { value: "other", label: "その他" },
      ],
      countries: [
        "中国本土", "香港", "マカオ", "台湾", "シンガポール", "マレーシア",
        "インドネシア", "タイ", "ベトナム", "フィリピン", "日本", "韓国", "インド",
        "UAE", "サウジアラビア", "イスラエル", "トルコ", "アメリカ", "カナダ", "イギリス",
        "ドイツ", "フランス", "オランダ", "スウェーデン", "スイス", "スペイン", "イタリア", "アイルランド",
        "ノルウェー", "デンマーク", "フィンランド", "ベルギー", "ポルトガル", "オーストラリア",
        "ニュージーランド", "南アフリカ", "ブラジル", "メキシコ", "アルゼンチン", "チリ", "その他",
      ],
      consent: {
        intro:
          "MeridianOS, Inc. はあなたのプライバシーを尊重します。個人情報はアカウント管理とご要望の製品・サービスの提供にのみ使用します。製品アップデート、ベストプラクティス、業界インサイトを時折お送りします — 受け取る場合は以下にチェックしてください。",
        outreach:
          "MeridianOS, Inc. からの一対一のコミュニケーション(サービスアップデート、サポート問い合わせ、ビジネスフォローアップ)を受け取りたい。",
        updates: "MeridianOS の製品アップデート、インサイト、イベント招待を受け取りたい。",
        unsubscribe:
          "いつでも配信停止できます。データの取り扱いとプライバシーの権利については、",
        submitConsent:
          "「送信」をクリックすると、MeridianOS, Inc. がご要望のコンテンツを提供するため、送信された情報を保存・処理することに同意したことになります。",
        privacyLinkLabel: "プライバシーポリシー",
        privacyLinkHref: "/about",
      },
      success: {
        title: "受領しました — ありがとうございます!",
        message:
          "MeridianOS チームが3営業日以内に返信します。その間、ドキュメントをご覧いただくか、GitHub で Star をお願いします。",
        cta: "ホームに戻る",
      },
      errors: {
        generic: "送信失敗 — 後でもう一度お試しください。",
        rateLimit: "このメールは最近複数回送信されています — 後でもう一度お試しください。",
        freeEmail: "企業メールをご使用ください — 無料メール(gmail、outlook など)は受け付けません。",
        invalidEmail: "メールアドレスの形式が正しくありません。",
      },
    },
  };
}
