import type { Locale } from "../i18n";

type InventionCopy = {
  navigation: {
    how: string;
    make: string;
    parents: string;
    aria: string;
  };
  header: {
    cta: string;
    dashboard: string;
    backHome: string;
    backHomeShort: string;
    language: string;
  };
  hero: {
    eyebrow: string;
    title: string;
    body: string;
    cta: string;
    secondaryCta: string;
    note: string;
    studioLabel: string;
    ideaLabel: string;
    ideaPrompt: string;
    planLabel: string;
    planNote: string;
    creationLabel: string;
    creationNote: string;
  };
  flow: {
    eyebrow: string;
    title: string;
    body: string;
    steps: {
      kicker: string;
      title: string;
      body: string;
    }[];
  };
  possibilities: {
    eyebrow: string;
    title: string;
    body: string;
    items: {
      title: string;
      note: string;
    }[];
  };
  parents: {
    eyebrow: string;
    title: string;
    body: string;
    values: {
      title: string;
      note: string;
    }[];
    screenTitle: string;
    screenBody: string;
  };
  ai: {
    eyebrow: string;
    title: string;
    body: string;
    childRole: string;
    aiRole: string;
  };
  cta: {
    eyebrow: string;
    title: string;
    body: string;
    button: string;
  };
  footer: {
    tagline: string;
    copyright: string;
  };
};

const en: InventionCopy = {
  navigation: {
    how: "How it works",
    make: "What kids make",
    parents: "For parents",
    aria: "Main navigation",
  },
  header: {
    cta: "Start inventing",
    dashboard: "Open CHIMII",
    backHome: "Back to home",
    backHomeShort: "Home",
    language: "Choose language",
  },
  hero: {
    eyebrow: "CHIMII · AI INVENTION KIT",
    title: "Imagine it. Build it. Bring it to life.",
    body: "An AI invention kit that helps kids turn ideas into real moving creations.",
    cta: "Start inventing",
    secondaryCta: "See how it works",
    note: "Start with any idea · Use the parts you already have · Build with your own hands",
    studioLabel: "FROM IDEA TO REALITY",
    ideaLabel: "A child imagines",
    ideaPrompt: "A little guard dog that wags its tail when I come home!",
    planLabel: "CHIMII makes a buildable plan",
    planNote: "Checks available parts · Plans stable steps",
    creationLabel: "The child brings it to life",
    creationNote: "Built, moving, and ready to be reinvented",
  },
  flow: {
    eyebrow: "ONE IDEA. THREE STEPS.",
    title: "AI does the translating. Kids do the inventing.",
    body: "CHIMII understands an idea, works with the parts at hand, and stays beside the child from first brick to first movement.",
    steps: [
      {
        kicker: "01 · IMAGINE IT",
        title: "Say it or sketch it.",
        body: "A machine pet, a secret door, a wonderfully weird racer—every invention starts in the child's own words.",
      },
      {
        kicker: "02 · BUILD IT",
        title: "Get a plan that can really be built.",
        body: "CHIMII understands the parts already available and creates clear, stable, step-by-step guidance.",
      },
      {
        kicker: "03 · BRING IT TO LIFE",
        title: "Make it move, react, and change.",
        body: "Add motion or expression, then keep testing and rebuilding until the invention feels truly theirs.",
      },
    ],
  },
  possibilities: {
    eyebrow: "WHAT WILL THEY MAKE?",
    title: "Not another thing to watch. Something to build.",
    body: "Children choose the story. CHIMII helps them find the structure.",
    items: [
      { title: "Robot pets", note: "that move and react" },
      { title: "Secret bases", note: "with doors and hiding places" },
      { title: "Clever traps", note: "powered by gears and gravity" },
      { title: "Self-driving racers", note: "built to test and tune" },
      { title: "Expressive dolls", note: "with feelings on their face" },
    ],
  },
  parents: {
    eyebrow: "BUILT FOR GROWING MINDS",
    title: "The skills behind every wild invention.",
    body: "The finished creation is exciting. The thinking that happens along the way matters even more.",
    values: [
      { title: "Creativity", note: "Turn an open-ended idea into a personal design." },
      { title: "Problem solving", note: "Try, notice what failed, and find another way." },
      { title: "Engineering thinking", note: "Explore structure, motion, balance, and cause." },
      { title: "Responsible AI literacy", note: "Use AI as a tool—never as a substitute for thinking." },
      { title: "Confidence to create", note: "See an idea become something real through their own effort." },
    ],
    screenTitle: "More hands-on making. Less passive screen time.",
    screenBody: "CHIMII gives the next useful cue, then guides attention back to the bricks, mechanisms, and real-world problem in front of the child.",
  },
  ai: {
    eyebrow: "A HEALTHIER ROLE FOR AI",
    title: "AI is the guide. Your child is the inventor.",
    body: "CHIMII offers plans, questions, and encouragement. The child chooses, builds, tests, fixes, and gives the creation its character.",
    childRole: "Chooses · Builds · Tests · Reinvents",
    aiRole: "Understands · Plans · Guides · Encourages",
  },
  cta: {
    eyebrow: "THE NEXT IDEA IS ALREADY WAITING",
    title: "What will they bring to life first?",
    body: "Give imagination a way out of the screen and into the real world.",
    button: "Start with an idea",
  },
  footer: {
    tagline: "Helping kids turn imagination into real-world invention.",
    copyright: "© {year} CHIMII. All rights reserved.",
  },
};

const zh: InventionCopy = {
  navigation: {
    how: "如何创造",
    make: "孩子能做什么",
    parents: "家长收获",
    aria: "主导航",
  },
  header: {
    cta: "开始创造",
    dashboard: "打开 CHIMII",
    backHome: "返回首页",
    backHomeShort: "首页",
    language: "选择语言",
  },
  hero: {
    eyebrow: "奇觅发明家 · AI 实体创造套件",
    title: "让孩子从 AI 内容消费者，变成真实世界的创造者。",
    body: "孩子从任意想法出发，AI 理解现有零件，生成可执行方案，并陪伴孩子完成和改造。",
    cta: "开始创造",
    secondaryCta: "看看如何实现",
    note: "从任意想法出发 · 使用家中现有零件 · 亲手把它造出来",
    studioLabel: "从想法到真实世界",
    ideaLabel: "孩子提出想法",
    ideaPrompt: "我想做一只回家时会摇尾巴的机械小狗！",
    planLabel: "奇觅生成可执行方案",
    planNote: "理解现有零件 · 规划稳固步骤",
    creationLabel: "孩子亲手赋予生命",
    creationNote: "造出来、动起来，还能继续改造",
  },
  flow: {
    eyebrow: "一个想法 · 三步成真",
    title: "AI 负责理解和引导，孩子负责真正创造。",
    body: "奇觅理解孩子的想法和手边的零件，从第一块积木到第一次运动，全程陪伴但不替孩子动手。",
    steps: [
      {
        kicker: "01 · IMAGINE IT",
        title: "说出来，或画下来。",
        body: "机器宠物、秘密基地、古怪赛车——每个发明，都从孩子自己的表达开始。",
      },
      {
        kicker: "02 · BUILD IT",
        title: "得到真正能造出来的方案。",
        body: "奇觅理解家中已有零件，生成清楚、稳固、可以一步步执行的搭建指引。",
      },
      {
        kicker: "03 · BRING IT TO LIFE",
        title: "让它会动、会回应、还能改变。",
        body: "加入动力或表情，反复测试和改造，直到它真正成为孩子自己的发明。",
      },
    ],
  },
  possibilities: {
    eyebrow: "孩子会造出什么？",
    title: "不是再看一个内容，而是亲手造出一个东西。",
    body: "孩子决定故事，奇觅帮他们找到实现它的结构。",
    items: [
      { title: "机器宠物", note: "会移动，也会回应" },
      { title: "秘密基地", note: "有舱门和藏宝空间" },
      { title: "陷阱机关", note: "用齿轮和重力触发" },
      { title: "自动赛车", note: "边跑边测试改进" },
      { title: "情绪玩偶", note: "把心情写在脸上" },
    ],
  },
  parents: {
    eyebrow: "在创造中自然成长",
    title: "每一个脑洞背后，都是受用长久的能力。",
    body: "成品让孩子兴奋，而过程中发生的思考更重要。",
    values: [
      { title: "创造力", note: "把开放的想法，变成独一无二的设计。" },
      { title: "解决问题", note: "试一试，看见失败，再找到另一种办法。" },
      { title: "工程思维", note: "理解结构、运动、平衡和因果。" },
      { title: "负责任的 AI 素养", note: "把 AI 当作工具，而不是思考的替代品。" },
      { title: "创造的自信", note: "亲眼看见想法通过自己的努力成为现实。" },
    ],
    screenTitle: "更多真实动手，更少被动屏幕时间。",
    screenBody: "奇觅只在需要时给出下一条有效提示，然后把孩子的注意力带回零件、机械结构和眼前的真实问题。",
  },
  ai: {
    eyebrow: "让 AI 回到更健康的位置",
    title: "AI 是向导，孩子才是发明家。",
    body: "奇觅提供方案、提问和鼓励；孩子做选择、亲手搭建、测试修复，并赋予作品真正的个性。",
    childRole: "选择 · 搭建 · 测试 · 改造",
    aiRole: "理解 · 规划 · 引导 · 鼓励",
  },
  cta: {
    eyebrow: "下一个想法已经在等着了",
    title: "孩子最想先让什么活起来？",
    body: "给想象力一条从屏幕通往真实世界的路。",
    button: "从一个想法开始",
  },
  footer: {
    tagline: "帮助孩子把想象力变成真实世界的发明。",
    copyright: "© {year} CHIMII 奇觅。保留所有权利。",
  },
};

const ja: InventionCopy = {
  navigation: {
    how: "つくり方",
    make: "つくれるもの",
    parents: "保護者の方へ",
    aria: "メインナビゲーション",
  },
  header: {
    cta: "発明を始める",
    dashboard: "CHIMII を開く",
    backHome: "ホームに戻る",
    backHomeShort: "ホーム",
    language: "言語を選択",
  },
  hero: {
    eyebrow: "CHIMII · AI 発明キット",
    title: "想像する。つくる。命を吹き込む。",
    body: "子どものアイデアを、本当に動く作品へ変える AI 発明キット。",
    cta: "発明を始める",
    secondaryCta: "しくみを見る",
    note: "どんなアイデアからでも · 手元のパーツで · 自分の手でつくる",
    studioLabel: "アイデアから現実へ",
    ideaLabel: "子どもが想像する",
    ideaPrompt: "帰ってきたらしっぽを振る、小さな番犬をつくりたい！",
    planLabel: "CHIMII が組み立てられる設計に",
    planNote: "パーツを確認 · 安定した手順を設計",
    creationLabel: "子どもが命を吹き込む",
    creationNote: "つくって、動かして、また改造",
  },
  flow: {
    eyebrow: "ひとつのアイデア · 3つのステップ",
    title: "AI は翻訳役。発明するのは子ども。",
    body: "CHIMII はアイデアと手元のパーツを理解し、最初の一個から動き出す瞬間まで寄り添います。",
    steps: [
      { kicker: "01 · IMAGINE IT", title: "話す、または描く。", body: "ロボットペット、秘密基地、不思議なレーサー。すべては子ども自身の言葉から始まります。" },
      { kicker: "02 · BUILD IT", title: "本当につくれる設計へ。", body: "手元にあるパーツを理解し、わかりやすく安定した手順をつくります。" },
      { kicker: "03 · BRING IT TO LIFE", title: "動かし、反応させ、つくり変える。", body: "動きや表情を加え、試して直して、自分だけの発明に育てます。" },
    ],
  },
  possibilities: {
    eyebrow: "なにをつくる？",
    title: "見るためのコンテンツではなく、手でつくる何かを。",
    body: "物語を決めるのは子ども。CHIMII は形にする構造を見つけます。",
    items: [
      { title: "ロボットペット", note: "動いて反応する" },
      { title: "秘密基地", note: "扉と隠し場所つき" },
      { title: "からくり罠", note: "歯車と重力で動く" },
      { title: "自動レーサー", note: "走らせながら改良" },
      { title: "表情のある人形", note: "気持ちが顔に出る" },
    ],
  },
  parents: {
    eyebrow: "考える力を育てる",
    title: "自由な発明の中に、一生ものの力がある。",
    body: "完成した作品の喜びも、その途中で生まれる思考も大切にします。",
    values: [
      { title: "創造力", note: "自由な発想を自分だけの設計に。" },
      { title: "問題解決力", note: "試し、失敗に気づき、別の方法を探す。" },
      { title: "エンジニアリング思考", note: "構造、動き、バランス、因果を学ぶ。" },
      { title: "責任ある AI リテラシー", note: "AI を思考の代わりではなく道具として使う。" },
      { title: "つくる自信", note: "自分の力でアイデアが現実になる体験。" },
    ],
    screenTitle: "手を動かす時間を増やし、受け身の画面時間を減らす。",
    screenBody: "必要なヒントを伝えたら、視線をパーツや仕組み、目の前の問題へ戻します。",
  },
  ai: {
    eyebrow: "AI の健やかな役割",
    title: "AI はガイド。発明家は子どもです。",
    body: "CHIMII は設計、問いかけ、励ましを提供します。選び、つくり、試し、直し、個性を与えるのは子どもです。",
    childRole: "選ぶ · つくる · 試す · 改造する",
    aiRole: "理解する · 計画する · 導く · 励ます",
  },
  cta: {
    eyebrow: "次のアイデアが待っている",
    title: "最初に何へ命を吹き込みますか？",
    body: "想像力を画面の外へ、現実の世界へ。",
    button: "アイデアから始める",
  },
  footer: {
    tagline: "子どもの想像力を、現実の発明へ。",
    copyright: "© {year} CHIMII. All rights reserved.",
  },
};

const ko: InventionCopy = {
  navigation: {
    how: "만드는 방법",
    make: "만들 수 있는 것",
    parents: "부모님을 위해",
    aria: "기본 탐색",
  },
  header: {
    cta: "발명 시작하기",
    dashboard: "CHIMII 열기",
    backHome: "홈으로 돌아가기",
    backHomeShort: "홈",
    language: "언어 선택",
  },
  hero: {
    eyebrow: "CHIMII · AI 발명 키트",
    title: "상상하고. 만들고. 생명을 불어넣어요.",
    body: "아이들의 아이디어를 실제로 움직이는 창작물로 바꿔 주는 AI 발명 키트.",
    cta: "발명 시작하기",
    secondaryCta: "어떻게 작동하나요?",
    note: "어떤 아이디어든 · 이미 가진 부품으로 · 내 손으로 만들기",
    studioLabel: "아이디어에서 현실로",
    ideaLabel: "아이가 상상해요",
    ideaPrompt: "내가 집에 오면 꼬리를 흔드는 작은 경비견을 만들고 싶어!",
    planLabel: "CHIMII가 만들 수 있는 설계로",
    planNote: "보유 부품 확인 · 안정적인 단계 설계",
    creationLabel: "아이가 생명을 불어넣어요",
    creationNote: "만들고, 움직이고, 다시 바꾸기",
  },
  flow: {
    eyebrow: "하나의 아이디어 · 세 단계",
    title: "AI는 번역하고, 아이는 발명합니다.",
    body: "CHIMII는 아이디어와 손에 있는 부품을 이해하고, 첫 블록부터 처음 움직이는 순간까지 함께합니다.",
    steps: [
      { kicker: "01 · IMAGINE IT", title: "말하거나 그려요.", body: "로봇 반려동물, 비밀 기지, 엉뚱한 레이서—모든 발명은 아이의 말에서 시작됩니다." },
      { kicker: "02 · BUILD IT", title: "실제로 만들 수 있는 설계를 받아요.", body: "이미 가진 부품을 이해하고 명확하고 튼튼한 단계별 안내를 만듭니다." },
      { kicker: "03 · BRING IT TO LIFE", title: "움직이고, 반응하고, 달라지게 해요.", body: "움직임과 표정을 더하고, 시험하고 고치며 나만의 발명으로 완성합니다." },
    ],
  },
  possibilities: {
    eyebrow: "무엇을 만들까요?",
    title: "또 하나의 볼거리가 아니라, 직접 만들 무언가를.",
    body: "이야기는 아이가 정하고, CHIMII는 구현할 구조를 찾습니다.",
    items: [
      { title: "로봇 반려동물", note: "움직이고 반응해요" },
      { title: "비밀 기지", note: "문과 숨은 공간이 있어요" },
      { title: "영리한 함정", note: "기어와 중력으로 작동해요" },
      { title: "자동 레이서", note: "달리며 시험하고 조정해요" },
      { title: "표정 인형", note: "얼굴로 감정을 보여줘요" },
    ],
  },
  parents: {
    eyebrow: "생각하는 힘을 키우는 설계",
    title: "엉뚱한 발명 속에서 오래가는 능력이 자랍니다.",
    body: "완성품의 즐거움만큼 그 과정에서 일어나는 생각을 중요하게 봅니다.",
    values: [
      { title: "창의력", note: "열린 아이디어를 나만의 설계로 바꿔요." },
      { title: "문제 해결력", note: "시도하고, 실패를 발견하고, 다른 길을 찾아요." },
      { title: "공학적 사고", note: "구조, 움직임, 균형, 원인을 탐구해요." },
      { title: "책임 있는 AI 리터러시", note: "AI를 생각의 대체물이 아닌 도구로 사용해요." },
      { title: "창작 자신감", note: "내 노력으로 아이디어가 현실이 되는 경험을 해요." },
    ],
    screenTitle: "직접 만드는 시간은 늘리고, 수동적인 화면 시간은 줄여요.",
    screenBody: "CHIMII는 꼭 필요한 다음 힌트만 주고, 아이의 시선을 부품과 장치, 눈앞의 실제 문제로 돌려보냅니다.",
  },
  ai: {
    eyebrow: "더 건강한 AI의 역할",
    title: "AI는 안내자. 발명가는 아이입니다.",
    body: "CHIMII는 설계와 질문, 격려를 제공합니다. 선택하고, 만들고, 시험하고, 고치며 개성을 더하는 건 아이입니다.",
    childRole: "선택 · 만들기 · 시험 · 다시 발명",
    aiRole: "이해 · 계획 · 안내 · 격려",
  },
  cta: {
    eyebrow: "다음 아이디어가 기다리고 있어요",
    title: "가장 먼저 무엇에 생명을 불어넣을까요?",
    body: "상상력이 화면을 나와 현실 세계로 향하게 해 주세요.",
    button: "아이디어로 시작하기",
  },
  footer: {
    tagline: "아이들의 상상력을 현실의 발명으로.",
    copyright: "© {year} CHIMII. All rights reserved.",
  },
};

export const inventionCopy: Record<Locale, InventionCopy> = {
  en,
  "zh-Hans": zh,
  ja,
  ko,
};
