import { Action, Edge, Persona, Verdict } from "./types";

// Unset in local dev (talk to the API directly); "" in the Docker build (same origin via nginx).
export const API_BASE = process.env.REACT_APP_API_URL ?? "http://localhost:8000";
export const POLL_INTERVAL_MS = 500;
export const POLL_TIMEOUT_MS = 15 * 60 * 1000; // LLM runs on a rate-limited Groq tier can take minutes

export const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

export async function fetchJSON(url: string, init?: RequestInit) {
  const res = await fetch(url, init);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    const detail = Array.isArray(body.detail)
      ? body.detail.map((d: { msg: string }) => d.msg).join("; ")
      : body.detail;
    throw new Error(detail || `Request failed (${res.status})`);
  }
  return body;
}

// Map the API's raw /api/results payload onto the UI's Persona model.
export function parsePersonas(resultData: any): Persona[] {
  return resultData.personas.map((p: any) => ({
    id: p.persona_id,
    name: p.name,
    age: p.age,
    gender: p.gender || "unknown",
    location: p.location,
    followers: p.followers || 0,
    interests: p.interests || [],
    behavior: p.behavior || "",
    inTarget: p.in_target,
    shown: p.shown !== false,
    action: p.action as Action,
    watch: p.watch,
    round: p.round || (p.shown ? 1 : 0),
    why: p.why,
    x: 0,
    y: 0,
  }));
}

// API edges reference 1-based persona ids; the canvas indexes personas from 0.
export function parseEdges(resultData: any): Edge[] {
  return (resultData.edges || []).map((e: any) => ({
    from: e.from - 1,
    to: e.to - 1,
    type: e.type,
    round: e.round,
  }));
}

export function parseVerdict(resultData: any): Verdict {
  return {
    head: resultData.verdict.head,
    sub: resultData.verdict.sub,
    verdict: resultData.verdict.verdict,
    reached: resultData.personas.filter((x: any) => x.shown).length,
    inPct: resultData.verdict.in_pct,
    shareRate: resultData.verdict.share_rate,
    shares: resultData.breakdown.shares,
    shownCount: resultData.personas.length,
    virality: resultData.virality,
    algoScore: resultData.algo_score,
    engagement: resultData.engagement,
  };
}
