import React, { useEffect, useState } from "react";
import { AlertCircle, Check, Copy, Download, Lightbulb } from "lucide-react";
import { Action, Analysis, Breakdown, Verdict } from "../types";
import type { RunState } from "../App";
import { ACTION_STYLES } from "../constants";
import AnalysisPanel from "./AnalysisPanel";
import { card, secondaryButton, sectionTitle } from "./ui";
import { prefersReducedMotion } from "../lib/env";

interface ResultsPanelProps {
  runState: RunState;
  statusText: string;
  verdict: Verdict | null;
  breakdown: Breakdown | null;
  analysis: Analysis | null;
  population: number;
  copied: boolean;
  copySummary: () => void;
  exportJSON: () => void;
}

// Colour per verdict headline, so the outcome reads at a glance.
const VERDICT_TONE: Record<string, string> = {
  "Viral hit.": "text-good",
  "Niche hit.": "text-accent",
  "Echo chamber.": "text-warn",
  "Audience mismatch.": "text-warn",
  "Weak hook.": "text-bad",
  "Weak signal.": "text-bad",
};

// Counts up to `to` over ~1s (ease-out), so a new score lands with some weight.
function useCountUp(to: number, ms = 1100) {
  const [n, setN] = useState(prefersReducedMotion() ? to : 0);
  useEffect(() => {
    if (prefersReducedMotion()) {
      setN(to);
      return;
    }
    let raf = 0;
    const t0 = performance.now();
    const tick = (t: number) => {
      const p = Math.min(1, (t - t0) / ms);
      setN(Math.round(to * (1 - Math.pow(1 - p, 3))));
      if (p < 1) raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [to, ms]);
  return n;
}

function ScoreRing({ value }: { value: number }) {
  const shown = useCountUp(value);
  const C = 2 * Math.PI * 52;
  return (
    <div className="relative h-36 w-36 shrink-0" role="img" aria-label={`Virality score ${value} out of 100`}>
      <svg viewBox="0 0 120 120" className="h-full w-full -rotate-90">
        <circle cx="60" cy="60" r="52" fill="none" strokeWidth="6" className="stroke-ink/10" />
        <circle
          cx="60"
          cy="60"
          r="52"
          fill="none"
          strokeWidth="6"
          strokeLinecap="round"
          className="stroke-accent transition-[stroke-dashoffset] duration-100"
          strokeDasharray={C}
          strokeDashoffset={C * (1 - shown / 100)}
        />
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="font-display text-5xl font-bold tabular-nums tracking-tight">{shown}</span>
        <span className="text-[11px] uppercase tracking-[0.14em] text-ink/50">virality</span>
      </div>
    </div>
  );
}

function Metric({ label, value, hint }: { label: string; value: string; hint: string }) {
  return (
    <div className="p-4 sm:p-5">
      <div className={sectionTitle}>{label}</div>
      <div className="mt-2 font-display text-3xl font-bold tabular-nums tracking-tight">{value}</div>
      <div className="mt-0.5 text-xs text-ink/55">{hint}</div>
    </div>
  );
}

// CSS variables carrying an action's light and dark fill, for the dark: variant to pick.
const fillVars = (key: Action) =>
  ({ "--fill": ACTION_STYLES[key].fill.light, "--fill-dark": ACTION_STYLES[key].fill.dark } as React.CSSProperties);
const fillClass = "[background:var(--fill)] dark:[background:var(--fill-dark)]";

// One stacked bar of what everyone reached did.
function ReactionBar({ breakdown }: { breakdown: Breakdown }) {
  const parts = [
    { key: "share" as const, n: breakdown.shares },
    { key: "comment" as const, n: breakdown.comments },
    { key: "like" as const, n: breakdown.likes },
    { key: "skip" as const, n: breakdown.scrolls },
  ];
  const total = parts.reduce((s, p) => s + p.n, 0) || 1;
  return (
    <div className="space-y-2 p-4">
      <div className="flex h-2.5 overflow-hidden rounded-full bg-ink/10">
        {parts.map((p) =>
          p.n ? (
            <div
              key={p.key}
              style={{ ...fillVars(p.key), width: `${(p.n / total) * 100}%` }}
              className={`h-full border-r-2 border-surface last:border-r-0 ${fillClass}`}
            />
          ) : null
        )}
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-ink/65">
        {parts.map((p) => (
          <span key={p.key} className="inline-flex items-center gap-1.5">
            <span className={`h-2 w-2 rounded-full ${fillClass}`} style={fillVars(p.key)} />
            {ACTION_STYLES[p.key].label} <span className="tabular-nums font-medium text-ink">{p.n}</span>
          </span>
        ))}
      </div>
    </div>
  );
}

export default function ResultsPanel({
  runState,
  statusText,
  verdict,
  breakdown,
  analysis,
  population,
  copied,
  copySummary,
  exportJSON,
}: ResultsPanelProps) {
  if (runState === "idle") {
    return (
      <section className={`${card} p-6 sm:p-8`}>
        <div className={sectionTitle}>Ready when you are</div>
        <h2 className="headline mt-3 text-[clamp(2rem,1.4rem+2.4vw,3.25rem)]">
          See how it lands <span className="font-serif font-normal italic text-accent">before</span> you post.
        </h2>
        <p className="mt-4 max-w-xl text-sm text-ink/65">
          Pick your niche, describe your hook (or upload the video), and run it. A synthetic audience reacts, shares it with each other, and the
          platform decides whether to push it further. You'll get a verdict and specific fixes.
        </p>
      </section>
    );
  }

  if (runState === "running") {
    return (
      <section className={`${card} overflow-hidden p-6 sm:p-8`} aria-live="polite">
        <div className={sectionTitle}>Simulating</div>
        <div className="headline mt-3 text-[clamp(1.75rem,1.2rem+2vw,2.75rem)]">{statusText || "Running…"}</div>
        <div className="mt-6 h-1 overflow-hidden rounded-full bg-ink/10">
          <div className="h-full w-full animate-shimmer bg-[linear-gradient(90deg,transparent,rgb(var(--accent)),transparent)] bg-[length:50%_100%] bg-no-repeat" />
        </div>
        <div className="mt-3 text-xs text-ink/55">With LLM personas this can take a minute or two.</div>
      </section>
    );
  }

  if (runState === "error") {
    return (
      <section className="rounded-xl border border-bad/30 bg-bad/10 p-5 text-bad" role="alert">
        <div className="flex items-start gap-3">
          <AlertCircle className="mt-0.5 h-5 w-5 shrink-0" />
          <div className="text-sm">{statusText}</div>
        </div>
      </section>
    );
  }

  if (!verdict || !breakdown) return null;
  const engine = analysis?.engine === "llm" ? "LLM personas" : analysis?.engine === "mixed" ? "LLM + offline model" : "Offline model";

  return (
    <div className="space-y-6">
      <section className={`${card} animate-fade-up overflow-hidden`}>
        <div className="flex flex-col gap-6 p-6 sm:flex-row sm:items-center sm:justify-between sm:p-8">
          <div className="min-w-0 space-y-3">
            <div className={sectionTitle}>Verdict · {verdict.verdict}</div>
            <h2 className={`headline text-[clamp(2.75rem,1.5rem+4.5vw,5.5rem)] ${VERDICT_TONE[verdict.head] ?? ""}`}>{verdict.head.replace(/\.$/, "")}</h2>
            <p className="max-w-2xl font-serif text-xl leading-snug tracking-[-0.01em] text-ink/80">{verdict.sub}</p>
          </div>
          <ScoreRing value={verdict.virality} />
        </div>

        <div className="grid grid-cols-2 divide-ink/10 border-t border-ink/10 sm:grid-cols-4 sm:divide-x">
          <Metric
            label="Reached"
            value={`${verdict.reached}`}
            hint={analysis ? `of ${population} · ${analysis.amplification}× the first round` : `of ${population}`}
          />
          <Metric label="Engaged" value={`${verdict.engagement}%`} hint="liked, commented or shared" />
          <Metric label="Shared" value={`${verdict.shareRate}%`} hint={`${verdict.shares} ${verdict.shares === 1 ? "share" : "shares"}`} />
          <Metric label="Algorithm push" value={`${verdict.algoScore}`} hint="out of 100" />
        </div>

        <div className="border-t border-ink/10">
          <ReactionBar breakdown={breakdown} />
        </div>

        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-ink/10 px-5 py-3 text-xs text-ink/55">
          <span>
            {engine}
            {analysis && analysis.confidence >= 0 && (
              <>
                {" · "}
                <span className={analysis.confidence < 50 ? "font-medium text-warn" : ""}>
                  Confidence {analysis.confidence}%{analysis.confidence < 50 ? " · borderline" : ""}
                </span>
              </>
            )}
          </span>
          <div className="flex gap-2">
            <button onClick={copySummary} className={secondaryButton}>
              {copied ? <Check className="h-4 w-4 text-good" /> : <Copy className="h-4 w-4" />}
              {copied ? "Copied" : "Copy summary"}
            </button>
            <button onClick={exportJSON} className={secondaryButton}>
              <Download className="h-4 w-4" />
              Export
            </button>
          </div>
        </div>
      </section>

      {analysis?.engine === "mixed" && (
        <div className="flex items-start gap-3 rounded-xl border border-warn/30 bg-warn/10 p-4 text-sm text-warn" role="status">
          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
          <span>
            Only {analysis.llm_share ?? "some"}% of viewers were simulated by the LLM — the rest fell back to the offline model, usually because of
            Groq rate limits. Treat this result with caution, or wait a minute and run it again.
          </span>
        </div>
      )}

      {analysis && analysis.recommendations.length > 0 && (
        <section className={`${card} animate-fade-up p-6`} style={{ animationDelay: "120ms" }}>
          <h3 className="mb-4 flex items-center gap-2 font-display text-xl font-bold tracking-tight">
            <Lightbulb className="h-5 w-5 text-accent" /> How to improve
          </h3>
          <ol className="divide-y divide-ink/10">
            {analysis.recommendations.map((r, i) => (
              <li key={r} className="flex gap-4 py-3 text-sm text-ink/80 first:pt-0 last:pb-0">
                <span className="font-display text-sm font-bold tabular-nums text-accent">{String(i + 1).padStart(2, "0")}</span>
                <span>{r}</span>
              </li>
            ))}
          </ol>
        </section>
      )}

      {analysis && <AnalysisPanel analysis={analysis} />}
    </div>
  );
}
