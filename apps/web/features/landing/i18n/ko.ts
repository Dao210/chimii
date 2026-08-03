import { githubUrl } from "../components/shared";
import { createEnDict } from "./en";
import type { LandingDict } from "./types";

export function createKoDict(allowSignup: boolean): LandingDict {
  const base = createEnDict(allowSignup);

  return {
    ...base,
    header: {
      cta: "시작하기",
      dashboard: "워크벤치 열기",
      navigation: "주요 메뉴",
      openMenu: "메뉴 열기",
      closeMenu: "메뉴 닫기",
    },
    hero: {
      eyebrow: "B2B 아웃바운드 · 중국 → 글로벌",
      headlineLine1: "인바운드를 기다리지 마세요.",
      headlineLine2: "해외 바이어를 직접 끌어들이세요.",
      subheading:
        "광고 없이, 문의 대기 없이 — 자연어로 에이전트를 지휘해 해외 B2B 바이어를 직접 특정하고 한 명씩 끌어들이세요. 새 팀은 발판을 다지는 데, 숙련 팀은 플레이북을 복제하는 데 사용합니다.",
      cta: "出海查 AI · 라이브",
      ctaSecondary: "전체拓客 플로우 보기",
      downloadDesktop: "데스크톱 다운로드",
      talkToSales: "영업팀에 문의",
      worksWith: "도입처",
  },
    stats: {
      items: [
        { value: "10+", label: "리딩 기업 도입" },
        { value: "200+", label: "심층 조사 리포트 납품" },
        { value: "6단계", label: "拓客 풀자동 루프" },
      ],
      trustedBy:
        "스마트 하드웨어 · 소프트웨어 서비스 · AI 글로벌 진출 의 리딩 팀에서 도입",
    },
    liveDemo: {
      badge: "LIVE DEMO",
      title: "데모 중",
      subtitle: "에이전트가 해외 바이어를 어떻게 단계별로 끌어들이는지 보세요",
      browserBar: "meridian.app / 아웃리치 워크벤치 LIVE",
      steps: [
        { id: "research", title: "독일 채널 드릴다운 research", detail: "오프라인 가전 체인 특정 · 선행은 NRW / 바이에른" },
        { id: "targets", title: "체인 바이어 특정 targets", detail: "MediaHaus 외 3개사, 현재 공급업체 모집 중" },
        { id: "outreach", title: "개척 이메일 작성 outreach", detail: "각사 맞춤, 각자의 구매 신호 인용" },
        { id: "reply", title: "고의향 인바운드 수신 reply", detail: "MediaHaus 구매 담당자 답장, 수신함으로 정리" },
      ],
      standby: "전체 플로우 완료 · 다음 지시 대기 중",
      analyzing: "독일 소매 채널 · 드릴다운 분석 중…",
      analysisItems: [
        { label: "오프라인 가전 체인", percent: 52 },
        { label: "이커머스 플랫폼 소매", percent: 33 },
        { label: "지역 디스트리뷰터", percent: 15 },
      ],
      analysisNote:
        "◆ 선행 지역 **NRW / 바이에른** · 매장 밀도 최고, 집단 구매 집중 · CleanMax 인상 후 반값대 공석",
      locked: "체인 바이어 특정 · 검증 완료",
      lockedTargets: [
        { initials: "MH", name: "MediaHaus", meta: "가전 체인 · NRW · 공급업체 모집 중", tag: "✓ 높은 매치" },
        { initials: "EX", name: "Expert SE", meta: "가전 체인 · 전국 · 집구매 결정", tag: "✓ 높은 매치" },
        { initials: "EP", name: "EP:Group", meta: "가전 체인 · 바이에른 · 매장 밀집", tag: "✓ 높은 매치" },
      ],
      drafting: "EN · 생성됨",
      draftHeader: "TO: S. BRANDT · Buyer · MEDIAHAUS",
      draftBody:
        "Dear Ms. Brandt,\n\nSaw you're sourcing robot vacuums for the fall lineup — with CleanMax's price hike leaving the mid-tier open,\n\nwe ship from our DE warehouse in 12 days…",
      replyTitle: "고의향 인바운드 · 수신함 답장 처리",
      replyMeta: "S. Brandt · MediaHaus 가 답장했습니다",
      replyBody:
        "\"Interesting. Can you send your catalog and MOQ for the S9?\"",
      replyTag: "의향 등급: 높음 · NRW 가전 체인 구매",
      replyFooter: "✓ 수신함에 정리 · 고의향 고정",
    },
    comparison: {
      label: "같은 일 · 두 가지 삶의 방식",
      headlineLine1: "해외 채널에 판매 —",
      headlineLine2: "과거엔 도박이었다면, 지금은 한마디로.",
      pastLabel: "과거 · 비싸고, 느리고, 위험",
      presentLabel: "현재 · 빠르고, 저렴하고, 가역적",
      past: [
        { question: "시작 방법", answer: "현지로 날아가 로컬 팀을 고용" },
        { question: "성과까지의 시간", answer: "반 년 걸려 겨우 감 잡기" },
        { question: "시험 가능한 시장", answer: "예산은 하나에만 베팅" },
        { question: "베팅이 틀리면", answer: "수백만이 물거품" },
      ],
      present: [
        { question: "시작 방법", answer: "에이전트에게 한마디" },
        { question: "성과까지의 시간", answer: "당일 첫 바이어" },
        { question: "시험 가능한 시장", answer: "원하는 만큼" },
        { question: "맞지 않으면", answer: "다른 시장으로 전환해 재실행" },
      ],
    },
    features: {
      teammates: {
        label: "PRODUCT 01 · 라이브",
        title: "出海查 AI · 위챗 미니 프로그램",
        description:
          "국내 비즈니스에서는 기업정보를 먼저查합니다. 글로벌 진출에서는出海查를 먼저 쓰세요. 해외 기업명을 입력하면 등기 정보, 경영 신호, AI 심층 리포트 — 제품을 어디서切入해야 하는지까지 — 를 받습니다.",
        cards: [
          { title: "등기 정보 검증", description: "등기 정보, 이사 구성, 존속 상태 — 공식 데이터베이스에서 추출." },
          { title: "AI 심층 조사", description: "구매 동향, 채널 구조, 의사결정자 — 모든 결론에 출처와 신뢰도 명시." },
          { title: "切入 경로 제안", description: "제품 자료를 업로드하면, 리포트가 직접 알려줍니다: 이 고객과 어떻상해야 하는지." },
        ],
      },
      autonomous: {
        label: "글로벌 기업",
        title: "한 번이면 압니다.",
        description:
          "주요 해외 시장의 기업 데이터베이스를 커버. 영국 Tesco부터 독일 MediaHaus까지, 사명을 입력하면 수 초 내에 등기 정보, 구매 규모, 협업 가능성을 받습니다 — AI가 동시에 제품을 고려해切入 경로를 제시.",
        cards: [
          { title: "공식 데이터베이스 연동", description: "Companies House, Kantar, GLEIF 등 권위 있는 출처, 추적 가능." },
          { title: "비즈니스에 최적화", description: "제품 자료를 업로드하면 리포트가 자동으로 카테고리를 반영한切入 제안 생성." },
          { title: "심층 리포트 · 원클릭 생성", description: "협업 가능성 / 구매 정보 /切入 경로 / 리스크 — 4가지 차원을 한 번에." },
        ],
      },
      skills: {
        label: "위챗 진입",
        title: "위챗으로 스캔, 첫 조사 무료로",
        description:
          "앱 다운로드 불필요, 계정 가입 불필요. 위챗에서「출해조AI」를 검색하고 해외 기업명을 입력하면 수 초 내에 첫 AI 심층 리포트를 받습니다. 첫 조사는 무료.",
        cards: [
          { title: "위챗에서「출해조AI」검색", description: "위챗을 열고 미니 프로그램 이름을 검색하면 첫 기업 조사 시작." },
          { title: "첫 조사 무료", description: "기업 규모·지역 제한 없음. 영국 소매 대형부터 독일 체인 바이어까지 조사 가능." },
          { title: "리포트 끝에서 원클릭", description: "리포트 끝의 버튼이 완전한 6단계拓客 플로우로 직통 — 한 회사 이해에서 시장 전체 제압으로." },
        ],
      },
      runtimes: {
        label: "데모 유스케이스",
        title: "TESCO PLC · 공개 정보 데모",
        description:
          "데모 데이터는 TESCO PLC의 공개 정보에 기반. 완전한出海查 심층 리포트가 어떤 모습인지를 보여줍니다 — 등기 정보, 구매 규모, 카테고리를 반영한切入 제안까지.",
        cards: [
          { title: "등기 정보 검증", description: "등록 번호 00445790 · Active 존속 · Companies House · 시장 점유율 27.4%." },
          { title: "구매 정보", description: "베이비 카테고리 연간 구매 £3–5억 · 공급업체 요건 BSCI / UKCA." },
          { title: "AI切入 경로 제안", description: "먼저 Own Label OEM으로切入, 이후 브랜드 입점 협상 — BSCI/SEDEX + UKCA 필요." },
        ],
      },
    },
    transition: {
      headlineLine1: "한 회사를 이해하는 건 시작점에 불과합니다.",
      headlineLine2: "다음은, 시장 전체를*제압하는* 것입니다.",
      body: "出海查 리포트 끝의 버튼이 AI 자동 아웃리치를 실행 — 완전한 6단계拓客 플로우로 ↓",
    },
    workflow: {
      label: "PRODUCT 02 · 내부 베타, 대기자 명단",
      headline: "拓客 엔진:시장 조사부터, 성약 한 발 전까지(내부 베타)",
      subheading:
        "또 하나의 대량 메일 도구가 아닙니다.出海拓客 전 과정을 커버하는 워크벤치 — 각 단계마다 AI가 뛰고 판단하며, 당신이 승인하에 진행합니다.",
      steps: [
        {
          id: "research_market",
          code: "STEP 01 / RESEARCH_MARKET",
          title: "01 시장 조사",
          detail: "시장 특정 후, 공격해야 할 지역과 채널로 자동 드릴다운",
          panel: {
            title: "3개 후보 시장, 순위 정렬됨",
            subtitle: "선행 시장 확인 대기 중",
            body: "Lumo S9 로봇 청소기 · 유통",
            bullets: [
              "DE 독일 · 87 · 성숙 소매 · CR5 62% 집중 · 반값대 공석",
              "FR 프랑스 · 73 · 채널 분산 · 유통 중심",
              "ND 북유럽 · 69 · 고객단가 · 의사결정 주기 김",
            ],
          },
        },
        {
          id: "build_icp",
          code: "STEP 02 / BUILD_ICP",
          title: "02 ICP 구축",
          detail: "시장을 구체적인 바이어 그룹으로 분할, 규모와 의사결정 체인 산출",
          panel: {
            title: "독일 시장, 2개 바이어 그룹으로 분할",
            subtitle: "주력 그룹 선택 대기 중",
            body: "근거: 제품 포지셔닝 + 공개 구매 신호",
            bullets: [
              "그룹 ① · 주력:소매 체인 · 카테고리 구매 디렉터 · 규모 ~1,800 · 체인: 디렉터 → 카테고리 VP · 난이도 중",
              "그룹 ② · 보충:독립 가전 딜러 · 책임자 · 규모 ~3,200 · 체인: 단일 결정 · 난이도 하",
            ],
          },
        },
        {
          id: "find_contacts",
          code: "STEP 03 / FIND_CONTACTS",
          title: "03 연락처 검색",
          detail: "의사결정자 추출, 이메일 검증, 발송 가능한 리스트 제공",
          panel: {
            title: "의사결정자 + 전달 가능 이메일, 검증 완료",
            subtitle: "공개 소스에서 추출 · 중복 제거",
            body: "이메일 SMTP 검증 통과",
            bullets: [
              "의사결정자 이름 + 직책 + 부서",
              "이메일 SMTP 검증, 무효 주소 필터",
              "정보 출처와 갱신 시간 명시",
            ],
          },
        },
        {
          id: "draft_sequence",
          code: "STEP 04 / DRAFT_SEQUENCE",
          title: "04 시퀀스 작성",
          detail: "바이어마다 전용 영어 개척 이메일 작성, 구매 신호 인용",
          panel: {
            title: "각 바이어, 전용 이메일",
            subtitle: "첫 메일 + 3라운드 팔로업",
            body: "각 바이어의 구매 신호에서 생성 · 템플릿 대량 발송 아님",
            bullets: [
              "바이어별 맞춤, 변수 치환 아님",
              "첫 메일은 해당 바이어의 실제 구매 신호 인용",
              "3라운드 팔로업은 의사결정 주기에 따라 자동 편성",
            ],
          },
        },
        {
          id: "schedule_send",
          code: "STEP 05 / SCHEDULE_SEND",
          title: "05 예약 발송",
          detail: "상대 시간대의 업무 시간에 배달, 도메인 웜업과 속도 제한",
          panel: {
            title: "상대의 아침 9시에, 정시 도착",
            subtitle: "시간대 예약 · 속도 제한 발송",
            body: "발신 평판 관리 · 심야 방해 없음",
            bullets: [
              "상대 로컬 업무 시간에 배달",
              "도메인 웜업 곡선, 블랙리스트 회피",
              "첫 메일과 팔로업을 최적 간격으로 편성",
            ],
          },
        },
        {
          id: "handle_reply",
          code: "STEP 06 / HANDLE_REPLY",
          title: "06 답장 처리",
          detail: "답장을 자동 분급해 수신함으로, 고의향은 고정 알림",
          panel: {
            title: "수신함 자동 분급, 고의향 고정",
            subtitle: "답장 자동 의향 분급",
            body: "당신 차례일 때만 알림 · 정말 중요한 몇 건만 처리",
            bullets: [
              "의향 분급: 높음 / 팔로업 / 무효",
              "고의향 고정, 인계 촉구",
              "팔로업형은 다음 라운드에 자동 큐잉",
            ],
          },
        },
      ],
      noteLabel: "근거",
      note: "제품 포지셔닝 + 공개 구매 신호 · 신뢰도: 중 · 그룹은 워크벤치 내에서 정밀 조정 가능",
    },
    founder: {
      label: "왜 우리인가",
      headlineLine1: "모든 중국 팀이,",
      headlineLine2: "글로벌 시장에 뛰어들 수 있도록.",
      paragraphs: [
        "저는 여러 실리콘밸리 주목 스타트업에서 B2B 세일즈 챔피언을 지냈고, Uber 같은 Fortune 500에서 고객 성장을 이끌었고, 중국에서 20명 규모의 출해 팀을 이끌며 중국 기업의 제품을 해외 대형 고객에게 직접 판매했습니다. 최고의 플레이북과 모든 함정을 알고 있습니다.",
        "하지만 한 가지는 분명합니다 — 해외 시장은 로컬 영업 베테랑을 고용할 수 있는 대기업만의 것이어서는 안 됩니다. AI가 가장 무거운 「조사 + 판단 + 아웃리치」를 맡을 때, 출해의 문턱은 완전히 평준화됩니다 — 외무 오너, 처음 해외에 도전하는 창업가, 소규모 팀 누구나 로컬 베테랑처럼 바이어를 한 명씩 끌어들일 수 있게 됩니다.",
      ],
      punchline: "그것이 MeridianOS가 하는 일 — 출해를, 누구나 할 수 있는 것으로.",
      name: "Zhou Yulin",
      role: "창업자 & CEO",
    },
    waitlist: {
      label: "내부 베타 · 매월 소수",
      headline: "拓客 엔진 내부 베타 대기자 명단에 참여",
      body: "연락처를 남겨주시면 런칭 시 첫 번째로 알려드립니다. 기존出海查 사용자가 우선입니다.",
      cta: "시작하기",
      note: "기존出海查 사용자가 우선입니다.",
    },
    howItWorks: {
      label: "시작하기",
      headlineMain: "出海查 AI · 위챗에서 검색,",
      headlineFaded: "첫 조사 리포트를 받으세요.",
      steps: [
        {
          title: allowSignup ? "위챗에서「출해조AI」검색" : "워크벤치 로그인",
          description: allowSignup
            ? "위챗에서 미니 프로그램「출해조AI」를 검색 — 앱 불필요, 계정 불필요, 수 초 내 시작."
            : "이메일과 인증 코드로 로그인하여 워크벤치에 진입, 첫 해외 고객 조사를 시작.",
        },
        {
          title: "해외 기업명 입력",
          description:
            "영국 Tesco부터 독일 MediaHaus까지 — 조사할 해외 기업을 입력. 수 초 내에 등기 정보, 경영 신호, AI 심층 리포트를 받습니다.",
        },
        {
          title: "제품 자료 업로드",
          description:
            "업로드 후, 리포트는 자동으로 카테고리를 반영한切入 경로 제안을 생성 — 이 고객과 어떻상해야 하는지, 어떤 인증이 필요한지, 어떤 라인을 선행할지.",
        },
        {
          title: "원클릭으로 완전한 6단계拓客 플로우로",
          description:
            "리포트 끝의 버튼이 완전한 6단계拓客 엔진으로 직통 — 한 회사 이해에서 해외 시장 전체 제압으로.",
        },
      ],
      cta: "시작하기",
      ctaGithub: "GitHub에서 보기",
    },
    openSource: {
      label: "出海查 AI",
      headlineLine1: "위챗으로 스캔,",
      headlineLine2: "첫 조사를 무료로.",
      description:
        "다운로드 불필요, 가입 불필요. 위챗에서「출해조AI」를 검색하고 해외 기업명을 입력하면 수 초 내에 첫 AI 심층 리포트를 받습니다 — 제품을 어디서切入해야 하는지까지.",
      cta: "위챗에서「출해조AI」검색",
      highlights: [
        { title: "앱 다운로드 불필요", description: "위챗 미니 프로그램으로 즉시 실행. 국내 네트워크에서 직접 접속, VPN 불필요, 설정 불필요." },
        { title: "첫 조사 무료", description: "기업 규모·지역 제한 없음. 영국 소매 대형부터 독일 체인 바이어까지 조사 가능." },
        { title: "권위 있는 데이터 소스", description: "Companies House, Kantar, GLEIF 등 권위 있는 데이터베이스. 모든 결론에 출처와 신뢰도 명시." },
        { title: "비즈니스에 최적화", description: "제품 자료를 업로드하면 리포트가 자동으로 카테고리를 반영한切入 경로 제안을 생성 — 이 고객과 어떻상해야 하는지." },
      ],
    },
    faq: {
      label: "FAQ",
      headline: "질문과 답변.",
      items: [
        {
          question: "出海查 AI란 무엇인가요?拓客 엔진과는 어떤 관계인가요?",
          answer:
            "出海查 AI는 MeridianOS의 위챗 미니 프로그램으로, 해외 기업 조사를 무료로 제공합니다: 등기 정보, 경영 신호, AI 심층 리포트. 拓客 엔진은 유료 제품으로, 出海查 리포트 위에 완전한 6단계拓客 플로우를 구축합니다 — 시장 조사, ICP 구축, 연락처 검색부터, 개척 이메일 작성, 예약 발송, 답장 처리까지.",
        },
        {
          question: "VPN이 필요한가요? 앱 다운로드는?",
          answer:
            "불필요합니다. 出海查 AI는 위챗 미니 프로그램으로, 국내 네트워크에서 직접 열립니다. 拓客 엔진 내부 베타석은 웹 워크벤치에서 사용, 데스크톱 버전은 추후 제공 예정.",
        },
        {
          question: "LinkedIn이나 해외 기업 명부와 무엇이 다른가요?",
          answer:
            "出海查 AI는 단순한 기업 명함이 아닙니다. 권위 있는 데이터베이스와 AI 심층 조사를 결합해, 그 기업의 구매 규모, 의사결정 체인, 협업 가능성을 알려줍니다. 또한 제품 카테고리를 반영한切入 경로 제안을 생성 — 이 고객과 어떻상해야 하는지, 어떤 인증이 필요한지, 어떤 라인을 선행할지.",
        },
        {
          question: "拓客 엔진은 지금 바로 사용할 수 있나요? 어떻게 신청하나요?",
          answer:
            "拓客 엔진은 현재 내부 테스트 중이며, 매월 소수의석입니다. 연락처를 남겨주시면 런칭 시 첫 번째로 알려드립니다. 기존出海查 사용자가 우선입니다.",
        },
        {
          question: "제품 자료는 안전한가요? 모델 학습에 사용되나요?",
          answer:
            "제품 자료는 비즈니스 맞춤형 조사 리포트와切入 경로 제안을 생성하는 데에만 사용됩니다. 모델 학습에 사용되지 않으며, 제3자와 공유되지 않습니다.",
        },
        {
          question: "어떤 해외 시장·업종을 지원하나요?",
          answer:
            "스마트 하드웨어, 소프트웨어 서비스, AI 글로벌 진출 등 리딩 기업에 도입되었습니다. 북미, 유럽, 동남아, 중동 등 주요 해외 시장을 커버. 데모에 없는 카테고리나 타겟 시장이 있다면 영업팀에 문의해 맞춤方案을 논의하세요.",
        },
      ],
    },
    footer: {
      tagline:
        "모든 중국 팀이, 해외 로컬 베테랑처럼 싸울 수 있도록. 해외 비즈니스를, 누구나 할 수 있는 것으로.",
      cta: "시작하기",
      groups: {
        product: {
          label: "제품",
          links: [
            { label: "出海查 AI", href: "#product" },
            { label: "拓客 플로우", href: "#workflow" },
            { label: "왜 우리인가", href: "#founder" },
            { label: "다운로드", href: "/download" },
          ],
        },
        resources: {
          label: "리소스",
          links: [
            { label: "API", href: githubUrl },
            { label: "X (Twitter)", href: "https://x.com/MeridianOSAI" },
          ],
        },
        company: {
          label: "회사",
          links: [
            { label: "소개", href: "/about" },
            { label: "오픈소스", href: "#open-source" },
            { label: "영업 문의", href: "/contact-sales" },
            { label: "GitHub", href: githubUrl },
          ],
        },
      },
      copyright: "© {year} MeridianOS, Inc. All rights reserved.",
    },
    about: {
      title: "MeridianOS 소개",
      nameLine: {
        prefix: "MeridianOS — ",
        mul: "Meridian",
        tiplexed: "OS · ",
        i: "경",
        nformationAnd: "선",
        c: "이",
        omputing: "동서를",
        a: "하나의",
        gent: "네트워크로.",
      },
      paragraphs: [
        "MeridianOS(자오기)의 이름은 「자오선」— 지구상의 경선으로, 세계를 동에서 서로 하나의 네트워크로 연결하는 선 — 에서 따왔습니다. 중국 팀의 출해에 부족한 것은 결코 제품력이 아니라, 해외 바이어를 한 명씩 찾아내고 끌어들이는 네트워크입니다.",
        "과거, 이 네트워크를 구축할 수 있었던 것은 로컬 영업 베테랑을 고용할 수 있는 대기업뿐이었습니다. 현지로 날아가, 로컬 팀을 고용하고, 단일 시장에 베팅하는 — 베팅이 틀리면 수백만이 물거품. MeridianOS가 하는 일은 AI에게 가장 무거운 「조사 + 판단 + 아웃리치」를 맡기고, 출해의 문턱을 완전히 평준화하는 것입니다.",
        "우리는 믿습니다 — 해외 시장은 결코 대기업만의 것이어서는 안 됩니다. 외무 오너, 처음 해외에 도전하는 창업가, 소규모 팀 누구나 로컬 베테랑처럼 바이어를 한 명씩 끌어들일 수 있어야 합니다.",
        "그것이 MeridianOS — 출해拓客 전 과정을 커버하는 워크벤치입니다. 시장 조사, ICP 구축, 연락처 검색부터, 개척 이메일 작성, 예약 발송, 답장 처리까지, 각 단계마다 AI가 뛰고 판단하며, 당신이 승인하에 진행합니다.",
        "MeridianOS는 MeridianOS, Inc.(Delaware C-Corp)가 운영합니다. 중국에는 20명 규모의 팀이 있으며, 창업자는 여러 실리콘밸리 주목 스타트업에서 B2B 세일즈 챔피언을 지냈고, Uber 같은 Fortune 500에서 고객 성장을 이끈 경험을 갖고 있습니다.",
      ],
      cta: "GitHub에서 보기",
    },
    download: {
      hero: {
        macArm64: {
          title: "MeridianOS for macOS",
          sub: "Apple Silicon · 데몬 포함, 설정 불필요",
          primary: "다운로드(.dmg)",
          altZip: "또는 .zip",
        },
        macIntel: {
          title: "MeridianOS for macOS",
          sub: "Intel · 데몬 포함, 설정 불필요",
          primary: "다운로드(.dmg)",
          altZip: "또는 .zip",
        },
        winX64: {
          title: "MeridianOS for Windows",
          sub: "데몬 포함, 설정 불필요",
          primary: "다운로드(.exe)",
        },
        winArm64: {
          title: "MeridianOS for Windows",
          sub: "ARM · 데몬 포함, 설정 불필요",
          primary: "다운로드(.exe)",
        },
        linux: {
          title: "MeridianOS for Linux",
          sub: "데몬 포함, 설정 불필요",
          primary: "AppImage 다운로드",
          altFormats: "또는 .deb / .rpm",
        },
        unknown: {
          title: "플랫폼 선택",
          sub: "지원되는 모든 설치 파일은 아래에 있습니다.",
        },
        safariMacHint: "Intel Mac인가요? 아래에서 Intel 버전을 선택하세요.",
        archFallbackHint: "아키텍처가 다른가요? 모든 포맷은 아래에 있습니다.",
      },
      allPlatforms: {
        title: "모든 플랫폼",
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
        unavailable: "사용 불가",
      },
      cli: {
        title: "CLI를 원하시나요?",
        sub: "서버, 원격 개발기, 헤드리스 환경용. Desktop과 동일한 데몬을 터미널에서 설치.",
        installLabel: "설치",
        startLabel: "데몬 시작",
        sshNote: "서버에 있나요? SSH로 동일한 명령을 실행할 수 있습니다.",
        copyLabel: "복사",
        copiedLabel: "복사됨",
      },
      cloud: {
        title: "Cloud runtime(대기자 명단)",
        sub: "runtime을 호스팅해 드립니다 — 현재 미출시. 이메일을 남겨주시면 출시 시 알려드립니다.",
      },
      footer: {
        releaseNotes: "v{version} 릴리스 노트",
        allReleases: "모든 릴리스 보기",
        currentVersion: "현재 버전:{version}",
        versionUnavailable: "버전 가져오기 실패 — GitHub에서 확인하세요",
      },
    },
    contactSales: {
      pageTitle: "영업 문의",
      pageDescription:
        "MeridianOS의 AI 출해拓客 워크플로를 팀에 어떻게 도입할지 안내해 드립니다.",
      eyebrow: "영업 문의",
      title: "먼저 요구사항을 들려주세요",
      subtitle: "공식 상담 전에, 가장 적합한 방안을 맞춤 제안해 드립니다.",
      notice: {
        badge: "시스템은 기업 이메일 도메인만 인식합니다.",
        body: "개인 이메일(@gmail.com, @outlook.com 등)의 요청은 처리되지 않습니다.",
      },
      fields: {
        firstName: "이름",
        lastName: "성",
        businessEmail: "기업 이메일",
        businessEmailHint: "후속 조치를 위해 실제 기업 이메일 도메인을 사용하세요.",
        companyName: "회사명",
        companySize: "회사 규모",
        countryRegion: "국가 / 지역",
        useCase: "MeridianOS를 어떻게 사용하시거나 협력하실 예정인가요?",
        goals: "목표나 과제",
        goalsHint:
          "MeridianOS로 달성하고자 하는 목표나 직면한 과제를 알려주세요. 자세할수록 더 적절한 지원을 제공할 수 있습니다.",
        selectPlaceholder: "선택하세요",
        submit: "제출",
        submitting: "제출 중…",
      },
      companySizes: [
        { value: "1-10", label: "1 – 10명" },
        { value: "11-50", label: "11 – 50명" },
        { value: "51-200", label: "51 – 200명" },
        { value: "201-500", label: "201 – 500명" },
        { value: "501-1000", label: "501 – 1,000명" },
        { value: "1000+", label: "1,000명 이상" },
      ],
      useCases: [
        { value: "evaluate", label: "팀을 위해 MeridianOS 평가 중" },
        { value: "adopt_team", label: "팀 / 회사 내에 도입 희망" },
        { value: "self_host", label: "자체 인프라에서 자체 호스팅 필요" },
        { value: "integrate", label: "기존 도구와 연동 희망" },
        { value: "partner", label: "파트너십 / 채널 문의" },
        { value: "other", label: "기타" },
      ],
      countries: [
        "중국 본토", "홍콩", "마카오", "대만", "싱가포르", "말레이시아",
        "인도네시아", "태국", "베트남", "필리핀", "일본", "한국", "인도",
        "UAE", "사우디아라비아", "이스라엘", "튀르키예", "미국", "캐나다", "영국",
        "독일", "프랑스", "네덜란드", "스웨덴", "스위스", "스페인", "이탈리아", "아일랜드",
        "노르웨이", "덴마크", "핀란드", "벨기에", "포르투갈", "호주",
        "뉴질랜드", "남아프리카", "브라질", "멕시코", "아르헨티나", "칠레", "기타",
      ],
      consent: {
        intro:
          "MeridianOS, Inc.는 고객의 프라이버시를 존중합니다. 개인정보는 계정 관리 및 요청하신 제품/서비스 제공에만 사용합니다. 제품 업데이트, 모범 사례, 업계 인사이트를 가끔 공유합니다 — 수신을 원하시면 아래에 체크하세요.",
        outreach:
          "MeridianOS, Inc.의 일대일 커뮤니케이션(서비스 업데이트, 지원 문의, 비즈니스 후속)을 수신하고 싶습니다.",
        updates: "MeridianOS의 제품 업데이트, 인사이트, 이벤트 초대를 수신하고 싶습니다.",
        unsubscribe:
          "언제든지 수신을 취소할 수 있습니다. 데이터 처리 방식과 프라이버시 권리에 대해서는",
        submitConsent:
          "「제출」을 클릭하면, MeridianOS, Inc.가 요청하신 콘텐츠를 제공하기 위해 제출된 정보를 저장·처리하는 것에 동의한 것으로 간주됩니다.",
        privacyLinkLabel: "프라이버시 정책.",
        privacyLinkHref: "/about",
      },
      success: {
        title: "접수되었습니다 — 감사합니다!",
        message:
          "MeridianOS 팀이 3영업일 이내에 답변드립니다. 그 동안 문서를 확인하시거나 GitHub에서 Star를 눌러주세요.",
        cta: "홈으로 돌아가기",
      },
      errors: {
        generic: "제출 실패 — 나중에 다시 시도해 주세요.",
        rateLimit: "이 이메일은 최근 여러 번 제출되었습니다 — 나중에 다시 시도해 주세요.",
        freeEmail: "기업 이메일을 사용하세요 — 무료 이메일(gmail, outlook 등)은 받지 않습니다.",
        invalidEmail: "이메일 주소 형식이 올바르지 않습니다.",
      },
    },
  };
}
