import React from "react";
import { ChevronDown } from "lucide-react";
import { Analysis } from "../types";
import { card, sectionTitle } from "./ui";

function Chips({ items, empty }: { items: string[]; empty: string }) {
  if (!items.length) {
    return <span className="text-sm italic text-ink/50">{empty}</span>;
  }
  return (
    <div className="flex flex-wrap gap-1.5">
      {items.map((t) => (
        <span key={t} className="rounded-full border border-ink/15 px-2.5 py-0.5 text-xs font-medium text-ink/80">
          {t}
        </span>
      ))}
    </div>
  );
}

function ScoreBar({ label, value, hint }: { label: string; value: number; hint?: string }) {
  const color = value >= 65 ? "bg-good" : value >= 35 ? "bg-warn" : "bg-bad";
  return (
    <div>
      <div className="mb-1.5 flex justify-between text-sm">
        <span className="font-medium">{label}</span>
        <span className="tabular-nums text-ink/55">{value < 0 ? "n/a" : `${value}/100`}</span>
      </div>
      <div className="h-1.5 overflow-hidden rounded-full bg-ink/10">
        {value >= 0 && <div className={`h-full rounded-full ${color} origin-left animate-[grow_1s_cubic-bezier(.22,1,.36,1)_both]`} style={{ width: `${value}%` }} />}
      </div>
      {hint && <div className="mt-1.5 text-xs text-ink/55">{hint}</div>}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-ink/10 p-3">
      <div className="font-display text-2xl font-bold tabular-nums tracking-tight">{value}</div>
      <div className="text-xs leading-snug text-ink/55">{label}</div>
    </div>
  );
}

// Collapsible detail: which signals the model read from the inputs and what drove the verdict.
export default function AnalysisPanel({ analysis: a }: { analysis: Analysis }) {
  const verdicts = Object.entries(a.verdict_distribution || {}).sort((x, y) => y[1] - x[1]);
  const skip = a.relevant_skip_rate >= 0 ? a.relevant_skip_rate : a.hook_skip_rate;

  return (
    <details className={`${card} group`}>
      <summary className="flex cursor-pointer list-none items-center justify-between p-6 font-display text-xl font-bold tracking-tight">
        How the model read your post
        <ChevronDown className="h-5 w-5 text-ink/45 transition duration-300 ease-spring group-open:rotate-180" />
      </summary>

      <div className="grid gap-6 border-t border-ink/10 p-6 md:grid-cols-2">
        <div className="space-y-4">
          <div className="space-y-1.5">
            <div className={sectionTitle}>Audience interests</div>
            <Chips items={a.audience_topics} empty="None detected — pick a niche" />
            {(a.audience_age || a.audience_regions.length > 0) && (
              <div className="text-xs text-ink/55">{[a.audience_age, a.audience_regions.join(", ")].filter(Boolean).join(" · ")}</div>
            )}
          </div>
          <div className="space-y-1.5">
            <div className={sectionTitle}>Your post is about</div>
            <Chips items={a.content_topics} empty="Nothing specific detected" />
            {a.topic_source && (
              <div className="text-xs text-ink/55">{a.topic_source === "llm" ? "Topics read by LLM" : "Topics by keyword match"}</div>
            )}
          </div>
          {a.video_hook && (
            <div className="space-y-1.5">
              <div className={sectionTitle}>Read from your video (first 3s)</div>
              <p className="whitespace-pre-line rounded-lg bg-ink/5 p-3 text-sm text-ink/80">{a.video_hook}</p>
            </div>
          )}
          {a.video_rest && (
            <div className="space-y-1.5">
              <div className={sectionTitle}>The rest of your video</div>
              <p className="rounded-lg bg-ink/5 p-3 text-sm text-ink/80">{a.video_rest}</p>
            </div>
          )}
        </div>

        <div className="space-y-5">
          <ScoreBar
            label="Audience fit"
            value={a.audience_fit}
            hint={a.audience_fit < 0 ? "Audience too vague to measure fit" : "How directly the post speaks to your niche"}
          />
          <ScoreBar
            label="Hook strength"
            value={a.hook_score}
            hint={
              (a.hook_source === "llm" ? "Rated by LLM for this audience" : "Scored by keyword rules") +
              (a.hook_signals.length ? ` · ${a.hook_signals.join(" · ")}` : "")
            }
          />
          <div className="grid grid-cols-3 gap-2">
            <Stat label="interested viewers scrolled past" value={`${skip}%`} />
            <Stat label="of your audience engaged" value={`${a.in_target_engagement}%`} />
            <Stat label="of everyone else engaged" value={`${a.out_target_engagement}%`} />
          </div>
          {a.confidence >= 0 && verdicts.length > 0 && (
            <p className="text-xs text-ink/55">
              Across {a.replicates} simulated runs: {verdicts.map(([v, n]) => `${v.replace(/\.$/, "")} ${n}`).join(", ")}
            </p>
          )}
        </div>
      </div>
    </details>
  );
}
