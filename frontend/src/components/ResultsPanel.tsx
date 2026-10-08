import React from "react";
import { AlertCircle, Check, Copy, Download, Lightbulb, Sparkles } from "lucide-react";
import { Action, Analysis, Breakdown, Verdict } from "../types";
import type { RunState } from "../App";
import { ACTION_STYLES } from "../constants";
import AnalysisPanel from "./AnalysisPanel";
import { card, secondaryButton, sectionTitle } from "./ui";

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
  "Viral hit.": "text-emerald-600 dark:text-emerald-400",
  "Niche hit.": "text-indigo-600 dark:text-indigo-400",
  "Echo chamber.": "text-amber-600 dark:text-amber-400",
  "Audience mismatch.": "text-orange-600 dark:text-orange-400",
  "Weak hook.": "text-rose-600 dark:text-rose-400",
  "Weak signal.": "text-rose-600 dark:text-rose-400",
};

function Metric({ label, value, hint }: { label: string; value: string; hint: string }) {
  return (
    <div className="p-4">
      <div className="text-xs font-medium text-zinc-500 dark:text-zinc-400">{label}</div>
      <div className="mt-1 text-2xl font-semibold tabular-nums tracking-tight">{value}</div>
      <div className="mt-0.5 text-xs text-zinc-500 dark:text-zinc-400">{hint}</div>
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
      <div className="flex h-2.5 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
        {parts.map((p) =>
          p.n ? (
            <div
              key={p.key}
              style={{ ...fillVars(p.key), width: `${(p.n / total) * 100}%` }}
              className={`h-full border-r-2 border-white last:border-r-0 dark:border-zinc-900 ${fillClass}`}
            />
          ) : null
        )}
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-zinc-600 dark:text-zinc-400">
        {parts.map((p) => (
          <span key={p.key} className="inline-flex items-center gap-1.5">
            <span className={`h-2 w-2 rounded-full ${fillClass}`} style={fillVars(p.key)} />
            {ACTION_STYLES[p.key].label} <span className="tabular-nums font-medium text-zinc-900 dark:text-zinc-100">{p.n}</span>
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
      <section className={`${card} p-6`}>
        <div className="flex items-start gap-4">
          <div className="rounded-xl bg-indigo-50 p-2.5 text-indigo-600 dark:bg-indigo-500/10 dark:text-indigo-400">
            <Sparkles className="h-5 w-5" />
          </div>
          <div className="space-y-1">
            <h2 className="text-base font-semibold">See how your post lands before you publish it</h2>
            <p className="max-w-xl text-sm text-zinc-600 dark:text-zinc-400">
              Pick your niche, describe your hook (or upload the video), and run the simulation. A synthetic audience reacts to it, shares it with
              each other, and the platform decides whether to push it further. You'll get a verdict and specific fixes.
            </p>
          </div>
        </div>
      </section>
    );
  }

  if (runState === "running") {
    return (
      <section className={`${card} p-6`} aria-live="polite">
        <div className="flex items-center gap-3">
          <span className="h-5 w-5 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent dark:border-indigo-400 dark:border-t-transparent" />
          <div>
            <div className="text-sm font-medium">{statusText || "Running…"}</div>
            <div className="text-xs text-zinc-500 dark:text-zinc-400">With LLM personas this can take a minute or two.</div>
          </div>
        </div>
      </section>
    );
  }

  if (runState === "error") {
    return (
      <section className="rounded-2xl border border-rose-200 bg-rose-50 p-5 text-rose-800 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-300" role="alert">
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
      <section className={card}>
        <div className="flex flex-col gap-4 p-5 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0 space-y-1.5">
            <div className={sectionTitle}>Verdict</div>
            <h2 className={`text-2xl font-semibold tracking-tight ${VERDICT_TONE[verdict.head] ?? ""}`}>{verdict.head.replace(/\.$/, "")}</h2>
            <p className="max-w-2xl text-sm text-zinc-600 dark:text-zinc-400">{verdict.sub}</p>
          </div>
          <div className="shrink-0 sm:text-right">
            <div className={sectionTitle}>Virality score</div>
            <div className="text-4xl font-semibold tabular-nums tracking-tight">
              {verdict.virality}
              <span className="text-base font-normal text-zinc-400">/100</span>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-2 divide-zinc-100 border-t border-zinc-100 dark:divide-zinc-800 dark:border-zinc-800 sm:grid-cols-4 sm:divide-x">
          <Metric
            label="Reached"
            value={`${verdict.reached}`}
            hint={analysis ? `of ${population} · ${analysis.amplification}× the first round` : `of ${population}`}
          />
          <Metric label="Engaged" value={`${verdict.engagement}%`} hint="liked, commented or shared" />
          <Metric label="Shared" value={`${verdict.shareRate}%`} hint={`${verdict.shares} ${verdict.shares === 1 ? "share" : "shares"}`} />
          <Metric label="Algorithm push" value={`${verdict.algoScore}`} hint="out of 100" />
        </div>

        <div className="border-t border-zinc-100 dark:border-zinc-800">
          <ReactionBar breakdown={breakdown} />
        </div>

        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-zinc-100 px-5 py-3 text-xs text-zinc-500 dark:border-zinc-800 dark:text-zinc-400">
          <span>
            {engine}
            {analysis && analysis.confidence >= 0 && (
              <>
                {" · "}
                <span className={analysis.confidence < 50 ? "font-medium text-amber-600 dark:text-amber-400" : ""}>
                  Confidence {analysis.confidence}%{analysis.confidence < 50 ? " · borderline" : ""}
                </span>
              </>
            )}
          </span>
          <div className="flex gap-2">
            <button onClick={copySummary} className={secondaryButton}>
              {copied ? <Check className="h-4 w-4 text-emerald-500" /> : <Copy className="h-4 w-4" />}
              {copied ? "Copied" : "Copy summary"}
            </button>
            <button onClick={exportJSON} className={secondaryButton}>
              <Download className="h-4 w-4" />
              Export
            </button>
          </div>
        </div>
      </section>

      {analysis && analysis.recommendations.length > 0 && (
        <section className={`${card} p-5`}>
          <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold">
            <Lightbulb className="h-4 w-4 text-amber-500" /> How to improve
          </h3>
          <ol className="space-y-2">
            {analysis.recommendations.map((r, i) => (
              <li key={r} className="flex gap-3 text-sm text-zinc-700 dark:text-zinc-300">
                <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-zinc-100 text-xs font-medium tabular-nums text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400">
                  {i + 1}
                </span>
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
