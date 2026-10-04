import React from "react";
import { Analysis } from "../types";

function Chips({ items, empty, tone }: { items: string[]; empty: string; tone: string }) {
  if (!items.length) {
    return <span className="text-[11px] italic text-slate-500">{empty}</span>;
  }
  return (
    <div className="flex flex-wrap gap-1">
      {items.map((t) => (
        <span key={t} className={`text-[10px] border px-2 py-0.5 rounded-md font-semibold ${tone}`}>
          {t}
        </span>
      ))}
    </div>
  );
}

function ScoreBar({ label, value, hint }: { label: string; value: number; hint?: string }) {
  const color = value >= 65 ? "bg-emerald-500" : value >= 35 ? "bg-amber-500" : "bg-rose-500";
  return (
    <div>
      <div className="flex justify-between text-[11px] font-semibold mb-1">
        <span className="text-slate-500">{label}</span>
        <span className="font-mono font-bold">{value < 0 ? "n/a" : `${value}/100`}</span>
      </div>
      <div className="h-1.5 rounded-full overflow-hidden bg-slate-200 dark:bg-slate-800">
        {value >= 0 && <div className={`h-full rounded-full ${color}`} style={{ width: `${value}%` }} />}
      </div>
      {hint && <div className="text-[10px] text-slate-500 mt-1">{hint}</div>}
    </div>
  );
}

// Explains which signals from the inputs drove the verdict.
export default function AnalysisPanel({ analysis: a, darkMode }: { analysis: Analysis; darkMode: boolean }) {
  const card = `p-4 rounded-xl border shadow-xs space-y-3 ${
    darkMode ? "bg-slate-950/60 border-slate-800" : "bg-[#FAF8F5] border-[#E5DEC9]"
  }`;
  const chipTone = darkMode
    ? "bg-slate-900 border-slate-700 text-indigo-300"
    : "bg-white border-[#E8DFC9] text-[#5C4328]";
  const label = "text-[10px] font-bold uppercase tracking-wider text-slate-500 mb-1";
  const verdicts = Object.entries(a.verdict_distribution || {}).sort((x, y) => y[1] - x[1]);

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <div className={card}>
        <div className="flex items-center justify-between">
          <div className="text-xs font-bold">WHAT THE MODEL READ FROM YOUR INPUTS</div>
          {a.topic_source && (
            <span className="text-[10px] font-mono text-slate-500">
              {a.topic_source === "llm" ? "Topics read by LLM" : "Topics by keyword match"}
            </span>
          )}
        </div>
        <div>
          <div className={label}>Audience interests</div>
          <Chips items={a.audience_topics} empty="None detected — describe what they care about" tone={chipTone} />
          {(a.audience_age || a.audience_regions.length > 0) && (
            <div className="text-[10px] text-slate-500 mt-1">
              {[a.audience_age, a.audience_regions.join(", ")].filter(Boolean).join(" · ")}
            </div>
          )}
        </div>
        {a.video_hook && (
          <div>
            <div className={label}>Read from your video (first 3s)</div>
            <p className="text-[11px] whitespace-pre-line">{a.video_hook}</p>
          </div>
        )}
        <div>
          <div className={label}>Hook topics</div>
          <Chips items={a.content_topics} empty="None detected" tone={chipTone} />
        </div>
        <ScoreBar
          label="Audience fit"
          value={a.audience_fit}
          hint={a.audience_fit < 0 ? "Audience too vague to measure fit" : "How much the hook speaks to the audience's interests"}
        />
        <ScoreBar label="Hook strength" value={a.hook_score} hint={a.hook_signals.join(" · ") || "No strong hook signals found"} />
      </div>

      <div className={card}>
        <div className="flex items-center justify-between">
          <div className="text-xs font-bold">WHY THIS VERDICT</div>
          <span className="text-[10px] font-mono text-slate-500">
            {a.engine === "llm" ? "LLM personas" : a.engine === "mixed" ? "LLM + offline model" : "Offline model"}
          </span>
        </div>
        <div className="grid grid-cols-3 gap-2 text-center">
          {[
            ["Relevant viewers who skipped", a.relevant_skip_rate >= 0 ? a.relevant_skip_rate : a.hook_skip_rate],
            ["In-target engaged", a.in_target_engagement],
            ["Out-of-target engaged", a.out_target_engagement],
          ].map(([k, v]) => (
            <div key={k as string} className={`rounded-lg border p-2 ${darkMode ? "border-slate-800" : "border-[#E8DFC9] bg-white"}`}>
              <div className="text-base font-black font-mono">{v}%</div>
              <div className="text-[10px] text-slate-500 leading-tight">{k}</div>
            </div>
          ))}
        </div>
        {a.confidence >= 0 && verdicts.length > 0 && (
          <div className="text-[11px]">
            <span className={`font-bold ${a.confidence < 50 ? "text-amber-600 dark:text-amber-400" : ""}`}>
              Confidence {a.confidence}%{a.confidence < 50 ? " · borderline result" : ""}
            </span>
            <span className="text-slate-500">
              {" "}— {verdicts.map(([v, n]) => `${v.replace(/\.$/, "")} ${n}/${a.replicates}`).join(", ")} simulated runs
            </span>
          </div>
        )}
        <div>
          <div className={label}>How to improve</div>
          {a.recommendations.length ? (
            <ul className="space-y-1 text-[11px] list-disc pl-4">
              {a.recommendations.map((r) => (
                <li key={r}>{r}</li>
              ))}
            </ul>
          ) : (
            <span className="text-[11px] text-slate-500">No major issues found in the hook.</span>
          )}
        </div>
      </div>
    </div>
  );
}
