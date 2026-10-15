import { Action, ActionStyle } from "./types";

// One style per action; badge classes carry their own dark: variants, and the canvas
// picks the fill for the current theme.
export const ACTION_STYLES: Record<Action, ActionStyle> = {
  share: {
    label: "Shared",
    badge: "bg-violet-500/15 text-violet-700 ring-violet-500/30 dark:text-violet-300",
    fill: { light: "#6233d7", dark: "#9a75ff" },
  },
  comment: {
    label: "Commented",
    badge: "bg-amber-500/15 text-amber-800 ring-amber-500/30 dark:text-amber-300",
    fill: { light: "#c2780f", dark: "#f0b65a" },
  },
  like: {
    label: "Liked",
    badge: "bg-sky-500/15 text-sky-800 ring-sky-500/30 dark:text-sky-300",
    fill: { light: "#1f7fbf", dark: "#57afdf" },
  },
  skip: {
    label: "Scrolled past",
    badge: "bg-ink/5 text-ink/70 ring-ink/15",
    fill: { light: "#909083", dark: "#5c5c4e" },
  },
  never: {
    label: "Not reached",
    badge: "bg-ink/[.03] text-ink/45 ring-ink/10",
    fill: { light: "#cfcfc2", dark: "#2a2a27" },
  },
};

export const actionColors = (dark: boolean) =>
  Object.fromEntries(Object.entries(ACTION_STYLES).map(([k, v]) => [k, v.fill[dark ? "dark" : "light"]])) as Record<Action, string>;

// How many people see the post first. Mirrors the worker: 12.5–35% of the population
// depending on boost, at least 6 and at most half.
export function seedSize(population: number, boost: number) {
  return Math.min(Math.max(Math.round(population * (0.1 + 0.025 * boost)), 6), Math.floor(population / 2));
}

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
