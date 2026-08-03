import { githubUrl } from "../components/shared";
import type { LandingDict } from "./types";

export function createEnDict(allowSignup: boolean): LandingDict {
  return {
  header: {
    cta: "Get started",
    dashboard: "Open workbench",
    navigation: "Primary navigation",
    openMenu: "Open navigation menu",
    closeMenu: "Close navigation menu",
  },

  hero: {
    eyebrow: "B2B OUTBOUND · China → Global",
    headlineLine1: "Stop waiting for inbound leads.",
    headlineLine2: "Actively walk overseas buyers in.",
    subheading:
      "No ads, no waiting for inquiries — direct your Agent in plain language to actively lock onto overseas B2B buyers and walk each one in. New teams use it to bootstrap; proven teams use it to replicate their playbook.",
    cta: "ChuHaiCha AI · Live",
    ctaSecondary: "See the full outbound flow",
    downloadDesktop: "Download desktop",
    talkToSales: "Talk to sales",
    worksWith: "Trusted by",
  },

  stats: {
    items: [
      { value: "10+", label: "Leading enterprises served" },
      { value: "200+", label: "Deep research reports delivered" },
      { value: "6 steps", label: "Full outbound loop, automated" },
    ],
    trustedBy:
      "Trusted by leading teams in smart hardware · software services · AI going global",
  },

  liveDemo: {
    badge: "LIVE DEMO",
    title: "Live demo",
    subtitle: "Watch the Agent walk an overseas buyer in, step by step",
    browserBar: "meridian.app / Outreach Workbench LIVE",
    steps: [
      { id: "research", title: "German channel drilldown · research", detail: "Locked onto offline appliance chains · lead with NRW / Bavaria" },
      { id: "targets", title: "Lock chain buyers · targets", detail: "MediaHaus and two other chains, currently sourcing suppliers" },
      { id: "outreach", title: "Draft outreach emails · outreach", detail: "Customized per buyer, citing each one's buying signals" },
      { id: "reply", title: "High-intent inquiry received · reply", detail: "MediaHaus buyer replied, filed to inbox" },
    ],
    standby: "Full flow complete · awaiting your next instruction",
    analyzing: "German retail channel · drilldown analysis in progress…",
    analysisItems: [
      { label: "Offline appliance chains", percent: 52 },
      { label: "E-commerce retailers", percent: 33 },
      { label: "Regional distributors", percent: 15 },
    ],
    analysisNote:
      "◆ Lead regions **NRW / Bavaria** · highest store density, concentrated bulk buying · mid-tier gap after CleanMax price hike",
    locked: "Chain buyers locked, verified",
    lockedTargets: [
      { initials: "MH", name: "MediaHaus", meta: "Appliance chain · NRW · currently sourcing suppliers", tag: "✓ High match" },
      { initials: "EX", name: "Expert SE", meta: "Appliance chain · nationwide · centralized buying", tag: "✓ High match" },
      { initials: "EP", name: "EP:Group", meta: "Appliance chain · Bavaria · dense stores", tag: "✓ High match" },
    ],
    drafting: "EN · generated",
    draftHeader: "TO: S. BRANDT · Buyer · MEDIAHAUS",
    draftBody:
      "Dear Ms. Brandt,\n\nSaw you're sourcing robot vacuums for the fall lineup — with CleanMax's price hike leaving the mid-tier open,\n\nwe ship from our DE warehouse in 12 days…",
    replyTitle: "High-intent inquiry · inbox reply handling",
    replyMeta: "S. Brandt · MediaHaus replied",
    replyBody:
      "\"Interesting. Can you send your catalog and MOQ for the S9?\"",
    replyTag: "Intent level: high · NRW appliance chain buyer",
    replyFooter: "✓ Filed to inbox · high intent pinned",
  },

  comparison: {
    label: "Same job · two ways to live",
    headlineLine1: "Selling into overseas channels —",
    headlineLine2: "used to be a bet. Now it's a sentence.",
    pastLabel: "Past · expensive, slow, risky",
    presentLabel: "Present · fast, cheap, reversible",
    past: [
      { question: "How to start", answer: "Fly there, hire a local team" },
      { question: "Time to results", answer: "Half a year just to learn the ropes" },
      { question: "Markets you can test", answer: "Budget for only one bet" },
      { question: "If you bet wrong", answer: "Millions down the drain" },
    ],
    present: [
      { question: "How to start", answer: "One sentence to the Agent" },
      { question: "Time to results", answer: "First batch of buyers same day" },
      { question: "Markets you can test", answer: "As many as you want" },
      { question: "If it doesn't fit", answer: "Switch markets and rerun" },
    ],
  },

  features: {
    teammates: {
      label: "PRODUCT 01 · Live",
      title: "ChuHaiCha AI · WeChat mini-program",
      description:
        "Selling domestically, you check QiChaCha first. Going global, you check ChuHaiCha first. Enter an overseas company to get registration info, business signals, and an AI deep report — including where your product should break in.",
      cards: [
        { title: "Registration verification", description: "Registration, directors, status — pulled from official registries." },
        { title: "AI deep research", description: "Buying signals, channel structure, decision-makers — every claim sourced with confidence." },
        { title: "Entry-path recommendations", description: "Upload your product materials; the report tells you exactly how to pitch this buyer." },
      ],
    },
    autonomous: {
      label: "Global companies",
      title: "Look up anyone, instantly.",
      description:
        "Coverage across major overseas company registries. From Tesco in the UK to MediaHaus in Germany — type a name, get registration, buying scale, and feasibility in seconds. AI simultaneously factors in your product to tell you the entry path.",
      cards: [
        { title: "Official registry integration", description: "Companies House, Kantar, GLEIF and other authoritative sources, fully traceable." },
        { title: "Tailored to your business", description: "Upload product materials and the report auto-factors your category into entry recommendations." },
        { title: "Deep report · one click", description: "Feasibility / buying intel / entry path / risk — four dimensions in one shot." },
      ],
    },
    skills: {
      label: "WeChat entry",
      title: "Scan with WeChat, start your first lookup free",
      description:
        "No app to download, no account to register. Search 「出海查AI」 on WeChat, enter an overseas company, and get your first AI deep report in seconds. First lookup is free.",
      cards: [
        { title: "Search「出海查AI」on WeChat", description: "Open WeChat, search the mini-program name, and start your first company lookup." },
        { title: "First lookup free", description: "No limits on company size or region. From UK retail giants to German chain buyers." },
        { title: "One-click from report to outbound", description: "The button at the end of the report opens the full six-step outbound flow — from understanding one company to taking down a whole market." },
      ],
    },
    runtimes: {
      label: "Demo use case",
      title: "TESCO PLC · public-info demo",
      description:
        "Demo data is based on public information about TESCO PLC. It shows what a complete ChuHaiCha deep report looks like — from registration and buying scale, to entry recommendations tailored to your category.",
      cards: [
        { title: "Registration verification", description: "Company number 00445790 · Active · Companies House · 27.4% market share." },
        { title: "Buying intel", description: "Baby category annual buying £3–5M · supplier requirements BSCI / UKCA." },
        { title: "AI entry-path recommendation", description: "Enter via Own Label contract manufacturing first, then negotiate brand listing — requires BSCI/SEDEX + UKCA." },
      ],
    },
  },

  transition: {
    headlineLine1: "Understanding one company is just the start.",
    headlineLine2: "Next, *take down* the whole market.",
    body: "The button at the end of a ChuHaiCha report triggers AI outreach automatically — opening the full six-step outbound flow ↓",
  },

  workflow: {
    label: "PRODUCT 02 · Internal beta, waitlist",
    headline: "Outbound engine: from market research, to one step before close (beta)",
    subheading:
      "Not another email-blast tool. A workbench covering the full outbound journey — every step has AI running legs and making calls, with you signing off.",
    steps: [
      {
        id: "research_market",
        code: "STEP 01 / RESEARCH_MARKET",
        title: "01 Market research",
        detail: "After locking a market, auto-drill into the regions and channels worth hitting",
        panel: {
          title: "Three candidate markets, ranked",
          subtitle: "Awaiting your lead-market confirmation",
          body: "Lumo S9 robot vacuum · distribution",
          bullets: [
            "DE Germany · 87 · mature retail · CR5 62% concentrated · mid-tier gap",
            "FR France · 73 · fragmented channels · distribution-led",
            "ND Nordics · 69 · high ASP · long decision cycle",
          ],
        },
      },
      {
        id: "build_icp",
        code: "STEP 02 / BUILD_ICP",
        title: "02 Build ICP",
        detail: "Split the market into concrete buyer groups; size them and map decision chains",
        panel: {
          title: "German market, split into two buyer groups",
          subtitle: "Awaiting your lead-group pick",
          body: "Basis: product positioning + public buying signals",
          bullets: [
            "Group ① · main: retail chains · category buying directors · ~1,800 size · chain: director → category VP · reach: medium",
            "Group ② · supplement: independent appliance dealers · owners · ~3,200 size · chain: single-point decision · reach: low",
          ],
        },
      },
      {
        id: "find_contacts",
        code: "STEP 03 / FIND_CONTACTS",
        title: "03 Find contacts",
        detail: "Surface decision-makers, verify emails, hand you a list you can actually send to",
        panel: {
          title: "Decision-makers + deliverable emails, verified",
          subtitle: "Pulled from public sources · deduped one by one",
          body: "Email SMTP verification passed",
          bullets: [
            "Decision-maker name + title + department",
            "Emails SMTP-verified, invalid addresses filtered",
            "Source and update time annotated",
          ],
        },
      },
      {
        id: "draft_sequence",
        code: "STEP 04 / DRAFT_SEQUENCE",
        title: "04 Draft sequence",
        detail: "A bespoke English outreach email per buyer, citing their actual buying signals",
        panel: {
          title: "Every buyer, their own letter",
          subtitle: "First email + 3 follow-ups",
          body: "Generated from each buyer's buying signals · not a template blasted to thousands",
          bullets: [
            "Per-buyer custom, not variable substitution",
            "First email cites the buyer's real buying signals",
            "3 follow-ups paced automatically by decision cycle",
          ],
        },
      },
      {
        id: "schedule_send",
        code: "STEP 05 / SCHEDULE_SEND",
        title: "05 Schedule & send",
        detail: "Deliver in the buyer's working hours by timezone; domain warm-up and rate-limiting",
        panel: {
          title: "At their 9 AM, on the dot",
          subtitle: "Timezone-scheduled · rate-limited delivery",
          body: "Sender reputation managed · no midnight disturbances",
          bullets: [
            "Delivered in recipient's local working hours",
            "Domain warm-up curve, blacklist risk avoided",
            "First email and follow-ups at optimal intervals",
          ],
        },
      },
      {
        id: "handle_reply",
        code: "STEP 06 / HANDLE_REPLY",
        title: "06 Handle replies",
        detail: "Replies auto-graded and filed to inbox; high-intent pinned and surfaced",
        panel: {
          title: "Inbox auto-graded, high intent pinned",
          subtitle: "Replies auto-graded by intent",
          body: "You're only notified when it's your move · handle only the few that really matter",
          bullets: [
            "Intent grading: high / follow-up / invalid",
            "High intent pinned, with a nudge to take over",
            "Follow-ups auto-queued for the next round",
          ],
        },
      },
    ],
    noteLabel: "Basis",
    note: "Product positioning + public buying signals · confidence: medium · groups can be fine-tuned in the workbench",
  },

  founder: {
    label: "Why us",
    headlineLine1: "Let every Chinese team",
    headlineLine2: "fight into global markets.",
    paragraphs: [
      "I was a top B2B seller at several Silicon Valley startups, led customer growth at a Fortune 500 like Uber, and ran a 20-person outbound team in China — personally closing overseas enterprise deals for Chinese companies. I've seen the best playbooks and stepped in every pit.",
      "But I'm clear on one thing: overseas markets should never belong only to companies that can afford local sales veterans. When AI takes over the heaviest 'research + judgment + outreach', the barrier to going global gets leveled completely — export founders, first-time overseas entrepreneurs, and small teams can all walk buyers in like local veterans.",
    ],
    punchline: "That's what MeridianOS does: turn going global into something anyone can pull off.",
    name: "Zhou Yulin",
    role: "Founder & CEO",
  },

  waitlist: {
    label: "Internal beta · limited monthly slots",
    headline: "Join the waitlist for the outbound engine beta",
    body: "Leave your contact and we'll notify you first at launch. Existing ChuHaiCha users get priority.",
    cta: "Get started",
    note: "Existing ChuHaiCha users get priority.",
  },

  howItWorks: {
    label: "Get started",
    headlineMain: "ChuHaiCha AI · search WeChat,",
    headlineFaded: "get your first research report.",
    steps: [
      {
        title: allowSignup ? "Search「出海查AI」on WeChat" : "Sign in to the workbench",
        description: allowSignup
          ? "Search the 「出海查AI」 mini-program on WeChat — no app download, no account, opens in seconds."
          : "Sign in with email + verification code to enter the workbench and start your first overseas-customer lookup.",
      },
      {
        title: "Enter an overseas company name",
        description:
          "From Tesco in the UK to MediaHaus in Germany — type the overseas company you want to look up. Get registration, business signals, and an AI deep report in seconds.",
      },
      {
        title: "Upload your product materials",
        description:
          "After uploading, the report auto-factors your category into entry-path recommendations — how to pitch this buyer, which certifications you need, which line to lead with.",
      },
      {
        title: "One-click into the full six-step outbound flow",
        description:
          "The button at the end of the report opens the full six-step outbound engine — from understanding one company to taking down an entire overseas market.",
      },
    ],
    cta: "Get started",
    ctaGithub: "View on GitHub",
  },

  openSource: {
    label: "ChuHaiCha AI",
    headlineLine1: "Scan with WeChat,",
    headlineLine2: "start your first lookup free.",
    description:
      "No download, no registration. Search「出海查AI」on WeChat, enter an overseas company, and get your first AI deep report in seconds — including where your product should break in.",
    cta: "Search「出海查AI」on WeChat",
    highlights: [
      { title: "No app to download", description: "WeChat mini-program, opens instantly. Works on the domestic network — no VPN, no setup." },
      { title: "First lookup free", description: "No limits on company size or region. From UK retail giants to German chain buyers." },
      { title: "Authoritative sources", description: "Companies House, Kantar, GLEIF and other authoritative databases. Every claim sourced with confidence." },
      { title: "Tailored to your business", description: "Upload product materials and the report auto-factors your category into entry-path recommendations — how to pitch this buyer." },
    ],
  },

  faq: {
    label: "FAQ",
    headline: "Questions & answers.",
    items: [
      {
        question: "What is ChuHaiCha AI? How does it relate to the outbound engine?",
        answer:
          "ChuHaiCha AI is MeridianOS's WeChat mini-program, providing free overseas company lookup: registration, business signals, and AI deep reports. The outbound engine is the paid product — it runs the full six-step outbound flow on top of ChuHaiCha reports: market research, ICP building, contact finding, sequence drafting, scheduled sending, and reply handling.",
      },
      {
        question: "Do I need a VPN? Do I need to download an app?",
        answer:
          "No. ChuHaiCha AI is a WeChat mini-program — opens directly on the domestic network, no VPN needed. The outbound engine beta runs in a web workbench, with a desktop client coming later.",
      },
      {
        question: "How is this different from LinkedIn or overseas yellow pages?",
        answer:
          "ChuHaiCha AI isn't just a company名片. It combines authoritative databases with AI deep research to tell you the company's buying scale, decision chain, and feasibility — and factors in your product category to recommend an entry path: how to pitch, which certs you need, which line to lead with.",
      },
      {
        question: "Is the outbound engine available now? How do I get in?",
        answer:
          "The outbound engine is in internal beta, with limited monthly slots. Leave your contact and we'll notify you first at launch. Existing ChuHaiCha users get priority.",
      },
      {
        question: "Are my product materials safe? Will they be used to train models?",
        answer:
          "Your product materials are used only to generate research reports and entry-path recommendations for your business. They are never used for model training or shared with third parties.",
      },
      {
        question: "Which overseas markets and industries do you support?",
        answer:
          "We've served leading enterprises in smart hardware, software services, and AI going global. Coverage spans North America, Europe, Southeast Asia, and the Middle East. If your category or target market isn't in the demo, reach out to sales for a custom plan.",
      },
    ],
  },

  footer: {
    tagline:
      "Let every Chinese team fight like a local overseas veteran. Make overseas business doable for everyone.",
    cta: "Get started",
    groups: {
      product: {
        label: "Product",
        links: [
          { label: "ChuHaiCha AI", href: "#product" },
          { label: "Outbound flow", href: "#workflow" },
          { label: "Why us", href: "#founder" },
          { label: "Download", href: "/download" },
        ],
      },
      resources: {
        label: "Resources",
        links: [
          { label: "API", href: githubUrl },
          { label: "X (Twitter)", href: "https://x.com/MeridianOSAI" },
        ],
      },
      company: {
        label: "Company",
        links: [
          { label: "About", href: "/about" },
          { label: "Open source", href: "#open-source" },
          { label: "Contact sales", href: "/contact-sales" },
          { label: "GitHub", href: githubUrl },
        ],
      },
    },
    copyright: "© {year} MeridianOS, Inc. All rights reserved.",
  },

  about: {
    title: "About MeridianOS",
    nameLine: {
      prefix: "MeridianOS — ",
      mul: "Meridian",
      tiplexed: "OS · ",
      i: "the ",
      nformationAnd: "meridian ",
      c: "line",
      omputing: " that ",
      a: "connects ",
      gent: "east to west.",
    },
    paragraphs: [
      "MeridianOS takes its name from the meridian line — the longitude that connects the world east to west. What Chinese teams going global lack is never product power — it's a network that can find overseas buyers one by one and walk them in.",
      "In the past, only companies that could afford local sales veterans could lay this network. Fly there, hire a local team, bet on a single market — get it wrong and millions go down the drain. What MeridianOS does is use AI to take over the heaviest 'research + judgment + outreach' and level the barrier to going global completely.",
      "We believe overseas markets should never belong only to big companies. Export founders, first-time overseas entrepreneurs, and small teams should all be able to walk overseas buyers in like local veterans.",
      "That's MeridianOS — a workbench covering the full outbound journey. From market research, ICP building, and contact finding, to sequence drafting, scheduled sending, and reply handling — every step has AI running legs and making calls, with you signing off.",
      "MeridianOS is operated by MeridianOS, Inc. (Delaware C-Corp). Our 20-person team in China is led by a founder who was a top B2B seller at several Silicon Valley startups and led customer growth at a Fortune 500 like Uber.",
    ],
    cta: "View on GitHub",
  },

  download: {
    hero: {
      macArm64: {
        title: "MeridianOS for macOS",
        sub: "Apple Silicon · daemon bundled, no setup",
        primary: "Download (.dmg)",
        altZip: "or download .zip",
      },
      macIntel: {
        title: "MeridianOS for macOS",
        sub: "Intel · daemon bundled, no setup",
        primary: "Download (.dmg)",
        altZip: "or download .zip",
      },
      winX64: {
        title: "MeridianOS for Windows",
        sub: "daemon bundled, no setup",
        primary: "Download (.exe)",
      },
      winArm64: {
        title: "MeridianOS for Windows",
        sub: "ARM · daemon bundled, no setup",
        primary: "Download (.exe)",
      },
      linux: {
        title: "MeridianOS for Linux",
        sub: "daemon bundled, no setup",
        primary: "Download AppImage",
        altFormats: "or .deb / .rpm",
      },
      unknown: {
        title: "Pick your platform",
        sub: "All supported installers are listed below.",
      },
      safariMacHint: "On an Intel Mac? Pick the Intel build below.",
      archFallbackHint: "Wrong arch? All formats are listed below.",
    },
    allPlatforms: {
      title: "All platforms",
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
      unavailable: "Not available",
    },
    cli: {
      title: "Prefer the CLI?",
      sub: "For servers, remote dev boxes, headless environments. Same daemon as Desktop, installed via terminal.",
      installLabel: "Install",
      startLabel: "Start daemon",
      sshNote: "Already on a server? Run the same command over SSH.",
      copyLabel: "Copy",
      copiedLabel: "Copied",
    },
    cloud: {
      title: "Cloud runtime (waitlist)",
      sub: "We'll host the runtime for you — not live yet. Leave your email and we'll notify you at launch.",
    },
    footer: {
      releaseNotes: "v{version} release notes",
      allReleases: "View all releases",
      currentVersion: "Current version: {version}",
      versionUnavailable: "Version fetch failed — check GitHub instead",
    },
  },
  contactSales: {
    pageTitle: "Contact sales",
    pageDescription:
      "Learn how to roll out MeridianOS's AI outbound workflow in your team.",
    eyebrow: "Contact sales",
    title: "Tell us what you need first",
    subtitle: "Before a formal conversation, let us tailor the right plan for you.",
    notice: {
      badge: "System accepts enterprise email domains only.",
      body: "Requests from personal email (e.g. @gmail.com, @outlook.com) are not processed.",
    },
    fields: {
      firstName: "First name",
      lastName: "Last name",
      businessEmail: "Business email",
      businessEmailHint: "Use a real enterprise email domain so we can follow up.",
      companyName: "Company name",
      companySize: "Company size",
      countryRegion: "Country / region",
      useCase: "How do you plan to use MeridianOS or work with us?",
      goals: "Your goals or challenges",
      goalsHint:
        "Tell us what you want to achieve with MeridianOS, or the challenges you're facing. The more detail, the better we can help.",
      selectPlaceholder: "Please select",
      submit: "Submit",
      submitting: "Submitting…",
    },
    companySizes: [
      { value: "1-10", label: "1 – 10" },
      { value: "11-50", label: "11 – 50" },
      { value: "51-200", label: "51 – 200" },
      { value: "201-500", label: "201 – 500" },
      { value: "501-1000", label: "501 – 1,000" },
      { value: "1000+", label: "1,000+" },
    ],
    useCases: [
      { value: "evaluate", label: "Evaluating MeridianOS for my team" },
      { value: "adopt_team", label: "Want to roll out across my team / company" },
      { value: "self_host", label: "Need to self-host on my own infra" },
      { value: "integrate", label: "Want to integrate with existing tools" },
      { value: "partner", label: "Partnership / channel inquiry" },
      { value: "other", label: "Other" },
    ],
    countries: [
      "Mainland China", "Hong Kong SAR", "Macau SAR", "Taiwan", "Singapore",
      "Malaysia", "Indonesia", "Thailand", "Vietnam", "Philippines", "Japan",
      "South Korea", "India", "UAE", "Saudi Arabia", "Israel", "Turkey",
      "United States", "Canada", "United Kingdom", "Germany", "France",
      "Netherlands", "Sweden", "Switzerland", "Spain", "Italy", "Ireland",
      "Norway", "Denmark", "Finland", "Belgium", "Portugal", "Australia",
      "New Zealand", "South Africa", "Brazil", "Mexico", "Argentina", "Chile",
      "Other",
    ],
    consent: {
      intro:
        "MeridianOS, Inc. respects your privacy. We use your personal information only to manage your account and provide the products or services you've requested. We occasionally want to share product updates, best practices, or industry insights — check below if you'd like to receive them.",
      outreach:
        "I'd like to receive one-to-one communications from MeridianOS, Inc., including service updates, support inquiries, and business follow-ups.",
      updates:
        "I'd like to receive MeridianOS product updates, insights, and event invitations.",
      unsubscribe:
        "You can unsubscribe at any time. For how we handle your data and your privacy rights, see",
      submitConsent:
        "By clicking Submit, you agree that MeridianOS, Inc. may store and process the information you submit to deliver the content you've requested.",
      privacyLinkLabel: "privacy policy.",
      privacyLinkHref: "/about",
    },
    success: {
      title: "Got it — thank you!",
      message:
        "The MeridianOS team will reply within three business days. In the meantime, feel free to check our docs or star us on GitHub.",
      cta: "Back to home",
    },
    errors: {
      generic: "Submission failed — please try again later.",
      rateLimit: "This email has been submitted recently — please try later.",
      freeEmail:
        "Please use an enterprise email — free email (gmail, outlook, etc.) is not accepted.",
      invalidEmail: "Email address is malformed.",
    },
  },
  };
}
