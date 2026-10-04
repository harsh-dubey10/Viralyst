import { Action, ActionStyle } from "./types";

export const ACTION_CONFIG_LIGHT: Record<Action, ActionStyle> = {
  share: {
    label: "SHARED",
    badge: "bg-emerald-50 text-emerald-800 border-emerald-200 font-bold",
    color: "text-emerald-600",
    fill: "#059669",
    icon: "↗",
  },
  like: {
    label: "LIKED",
    badge: "bg-sky-50 text-sky-800 border-sky-200 font-bold",
    color: "text-sky-600",
    fill: "#0284C7",
    icon: "♥",
  },
  comment: {
    label: "COMMENTED",
    badge: "bg-purple-50 text-purple-800 border-purple-200 font-bold",
    color: "text-purple-600",
    fill: "#7C3AED",
    icon: "◉",
  },
  skip: {
    label: "SKIPPED",
    badge: "bg-slate-100 text-slate-600 border-slate-200 font-medium",
    color: "text-slate-500",
    fill: "#94A3B8",
    icon: "✕",
  },
  never: {
    label: "UNREACHED",
    badge: "bg-slate-50 text-slate-400 border-slate-200 font-normal",
    color: "text-slate-400",
    fill: "#E2E8F0",
    icon: "·",
  },
};

export const ACTION_CONFIG_DARK: Record<Action, ActionStyle> = {
  share: {
    label: "SHARED",
    badge: "bg-emerald-500/20 text-emerald-300 border-emerald-500/40 font-bold",
    color: "text-emerald-400",
    fill: "#10B981",
    icon: "↗",
  },
  like: {
    label: "LIKED",
    badge: "bg-sky-500/20 text-sky-300 border-sky-500/40 font-bold",
    color: "text-sky-400",
    fill: "#38BDF8",
    icon: "♥",
  },
  comment: {
    label: "COMMENTED",
    badge: "bg-purple-500/20 text-purple-300 border-purple-500/40 font-bold",
    color: "text-purple-400",
    fill: "#A855F7",
    icon: "◉",
  },
  skip: {
    label: "SKIPPED",
    badge: "bg-slate-800 text-slate-300 border-slate-700 font-medium",
    color: "text-slate-400",
    fill: "#64748B",
    icon: "✕",
  },
  never: {
    label: "UNREACHED",
    badge: "bg-slate-900 text-slate-500 border-slate-800 font-normal",
    color: "text-slate-500",
    fill: "#334155",
    icon: "·",
  },
};

export const CAMPAIGN_PRESETS = [
  {
    tag: "🔥 B2B SaaS",
    title: "AI Growth Playbook",
    tabStyle: "bg-[#F7F3EB] hover:bg-[#EFE7D8] text-[#5C4328] border-[#E5DAC4] dark:bg-amber-950/30 dark:hover:bg-amber-950/50 dark:text-amber-300 dark:border-amber-800/60",
    badgeStyle: "bg-[#EFE7D8] text-[#4A351E] dark:bg-amber-900/50 dark:text-amber-200",
    demographic: "Gen Z tech founders & SaaS product managers looking for viral growth strategies",
    content: "Revealing the exact 3-step distribution playbook we used to scale an AI tool from 0 to 50k MRR without paid ads. Hook: Stop building features nobody asked for.",
  },
  {
    tag: "🤖 Dev Tools",
    title: "Trace Visualizer",
    tabStyle: "bg-[#EEF7F2] hover:bg-[#E2F0E8] text-[#1E4A35] border-[#CFE5D8] dark:bg-emerald-950/30 dark:hover:bg-emerald-950/50 dark:text-emerald-300 dark:border-emerald-800/60",
    badgeStyle: "bg-[#D8EFE2] text-[#163A29] dark:bg-emerald-900/50 dark:text-emerald-200",
    demographic: "Full-stack software engineers, DevOps leads, and Indie Hackers building in public",
    content: "Why 90% of developers waste 4 hours every week debugging microservices. Introducing an automated trace visualizer that spots bottlenecks in 1 click.",
  },
  {
    tag: "📱 DTC Product",
    title: "Eco Coffee Drop",
    tabStyle: "bg-[#FDF2ED] hover:bg-[#FAE4D9] text-[#7A361D] border-[#F5D5C4] dark:bg-orange-950/30 dark:hover:bg-orange-950/50 dark:text-orange-300 dark:border-orange-800/60",
    badgeStyle: "bg-[#F7DACB] text-[#612A15] dark:bg-orange-900/50 dark:text-orange-200",
    demographic: "Eco-conscious millennial urban professionals & sustainable lifestyle creators",
    content: "We tested 100 plastic-free coffee pods so you don't have to. Here is the only one that actually brews espresso crema without degrading the planet.",
  },
  {
    tag: "💡 Creator Hook",
    title: "Short-Form Formula",
    tabStyle: "bg-[#F5F2FC] hover:bg-[#ECE5FA] text-[#4A2F8A] border-[#DFD6F5] dark:bg-purple-950/30 dark:hover:bg-purple-950/50 dark:text-purple-300 dark:border-purple-800/60",
    badgeStyle: "bg-[#E6DBFA] text-[#392170] dark:bg-purple-900/50 dark:text-purple-200",
    demographic: "Solo content creators, YouTube shorts editors, and digital agency operators",
    content: "The psychology behind 3-second visual hooks that stop doomscrolling on TikTok & Instagram Reels. Master the pattern interrupt framework.",
  },
];

export type CampaignPreset = typeof CAMPAIGN_PRESETS[0];
