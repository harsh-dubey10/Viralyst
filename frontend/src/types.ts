export type Action = "like" | "share" | "comment" | "skip" | "never";

export interface Persona {
  id: number;
  name: string;
  age: number;
  gender: string;
  location: string;
  followers: number;
  interests: string[];
  behavior: string;
  inTarget: boolean;
  shown: boolean;
  action: Action;
  watch: number;
  round: number;
  x: number;
  y: number;
  why?: string;
}

export interface Edge {
  from: number;
  to: number;
  type: "direct" | "algo";
  round: number;
}

export interface Breakdown {
  likes: number;
  shares: number;
  comments: number;
  scrolls: number;
}

export interface Analysis {
  audience_topics: string[];
  audience_age: string;
  audience_regions: string[];
  content_topics: string[];
  audience_fit: number; // -1 when the audience has no detectable interests
  hook_score: number;
  hook_signals: string[];
  recommendations: string[];
  hook_skip_rate: number;
  relevant_skip_rate: number;
  in_target_engagement: number;
  out_target_engagement: number;
  amplification: number;
  engine: "llm" | "heuristic" | "mixed";
  verdict_distribution: Record<string, number>;
  confidence: number; // -1 for LLM runs
  replicates: number;
  video_hook?: string; // what the worker extracted from the uploaded video's opening
  video_note?: string; // why the video couldn't be (fully) analysed
  topic_source?: "llm" | "keywords"; // how audience and hook topics were read
  hook_source?: "llm" | "keywords"; // how the hook was scored
  niche?: string; // the niche as given
}

export interface Verdict {
  head: string;
  sub: string;
  verdict: string;
  reached: number;
  inPct: number;
  shareRate: number;
  shares: number;
  shownCount: number;
  virality: number;
  algoScore: number;
  engagement: number;
}

export interface Region {
  x0: number;
  x1: number;
  y0: number;
  y1: number;
}

export interface ActionStyle {
  label: string;
  badge: string; // Tailwind classes (with dark: variants) for a ring badge
  fill: { light: string; dark: string }; // canvas node colour per theme
}
