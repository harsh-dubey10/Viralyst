import { Action, ActionStyle } from "./types";

// One style per action; badge classes carry their own dark: variants, and the canvas
// picks the fill for the current theme.
export const ACTION_STYLES: Record<Action, ActionStyle> = {
  share: {
    label: "Shared",
    badge: "bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-500/15 dark:text-emerald-300 dark:ring-emerald-500/30",
    fill: { light: "#059669", dark: "#34D399" },
  },
  comment: {
    label: "Commented",
    badge: "bg-violet-50 text-violet-700 ring-violet-200 dark:bg-violet-500/15 dark:text-violet-300 dark:ring-violet-500/30",
    fill: { light: "#7C3AED", dark: "#A78BFA" },
  },
  like: {
    label: "Liked",
    badge: "bg-sky-50 text-sky-700 ring-sky-200 dark:bg-sky-500/15 dark:text-sky-300 dark:ring-sky-500/30",
    fill: { light: "#0284C7", dark: "#38BDF8" },
  },
  skip: {
    label: "Scrolled past",
    badge: "bg-zinc-100 text-zinc-600 ring-zinc-200 dark:bg-zinc-800 dark:text-zinc-300 dark:ring-zinc-700",
    fill: { light: "#A1A1AA", dark: "#71717A" },
  },
  never: {
    label: "Not reached",
    badge: "bg-zinc-50 text-zinc-400 ring-zinc-200 dark:bg-zinc-900 dark:text-zinc-500 dark:ring-zinc-800",
    fill: { light: "#E4E4E7", dark: "#27272A" },
  },
};

// Suggested niches. `value` is what the worker receives; anything typed works too —
// a niche the worker doesn't know becomes a custom interest of its own.
export const NICHES = [
  { label: "Cooking", value: "cooking" },
  { label: "Travel", value: "travel" },
  { label: "Fitness", value: "fitness" },
  { label: "Beauty", value: "beauty" },
  { label: "Fashion", value: "fashion" },
  { label: "Gaming", value: "gaming" },
  { label: "Personal finance", value: "personal finance" },
  { label: "Comedy", value: "comedy" },
  { label: "Parenting", value: "parenting" },
  { label: "Education", value: "education" },
  { label: "Music", value: "music" },
  { label: "Pets", value: "pets" },
  { label: "Tech & AI", value: "AI tools" },
  { label: "Business", value: "startup ops" },
];

export interface Example {
  label: string;
  niche: string;
  demographic: string;
  content: string;
}

export const EXAMPLES: Example[] = [
  {
    label: "Weeknight pasta",
    niche: "cooking",
    demographic: "Busy parents in the US, 28–45, who cook dinner most nights",
    content: "This 10-minute garlic butter pasta uses only 4 ingredients — and it beats every restaurant version I've had.",
  },
  {
    label: "Vietnam on a budget",
    niche: "travel",
    demographic: "Budget backpackers in their 20s planning trips to Southeast Asia",
    content: "I spent 30 days in Vietnam on $600. Here's exactly where every dollar went.",
  },
  {
    label: "Chess trap",
    niche: "chess",
    demographic: "Club players rated 1200–1800",
    content: "This 5-move chess trap wins the queen against the Sicilian — and 80% of players fall for it.",
  },
  {
    label: "AI growth playbook",
    niche: "AI tools",
    demographic: "Startup founders and SaaS product managers",
    content: "The exact 3-step distribution playbook we used to take an AI tool from 0 to 50k MRR without paid ads. Stop building features nobody asked for.",
  },
];
