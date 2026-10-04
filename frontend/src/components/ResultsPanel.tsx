import React, { useMemo } from "react";
import { TrendingUp } from "lucide-react";
import { Analysis, Breakdown, Verdict } from "../types";
import AnalysisPanel from "./AnalysisPanel";
import ResultsCharts from "./ResultsCharts";

interface ResultsPanelProps {
  darkMode: boolean;
  verdict: Verdict | null;
  breakdown: Breakdown | null;
  analysis: Analysis | null;
  population: number;
}

// Section 03: verdict banner, KPI cards, analysis explanation and charts.
export default function ResultsPanel({
  darkMode,
  verdict,
  breakdown,
  analysis,
  population,
}: ResultsPanelProps) {
  const funnelData = useMemo(() => {
    if (!breakdown || !verdict) return [];
    return [
      { name: "Reached", value: verdict.reached, fill: "#4F46E5" },
      { name: "Likes", value: breakdown.likes, fill: "#0284C7" },
      { name: "Comments", value: breakdown.comments, fill: "#7C3AED" },
      { name: "Shares", value: breakdown.shares, fill: "#059669" },
    ];
  }, [breakdown, verdict]);

  const demographicPieData = useMemo(() => {
    if (!verdict) return [];
    return [
      { name: "In-Target", value: verdict.inPct, fill: "#4F46E5" },
      { name: "Out-of-Target", value: 100 - verdict.inPct, fill: "#F43F5E" },
    ];
  }, [verdict]);

  return (
    <section
      className={`border rounded-2xl p-4 sm:p-6 shadow-xs space-y-6 ${
        darkMode
          ? "border-slate-800/90 bg-slate-900/60 backdrop-blur-xl"
          : "border-[#E5DEC9] bg-white/95 backdrop-blur-md"
      }`}
    >
      <div
        className={`flex items-center justify-between border-b pb-3 ${
          darkMode ? "border-slate-800" : "border-[#EAE3D2]"
        }`}
      >
        <div className="flex items-center gap-2 text-xs font-bold tracking-wider">
          <span className="w-5 h-5 rounded-lg bg-purple-600 text-white flex items-center justify-center text-[10px] font-black shadow-xs">
            03
          </span>
          <span className={darkMode ? "text-slate-200" : "text-[#3D3021]"}>
            EXECUTIVE VERDICT & PREDICTIVE KPIS
          </span>
        </div>
        <span className="text-[11px] text-slate-500 font-semibold">
          Algorithm Score & Conversion Funnel
        </span>
      </div>

      {!verdict ? (
        <div className="py-8 text-center">
          <TrendingUp className="w-8 h-8 text-slate-400 mx-auto mb-2" />
          <div
            className={`text-sm font-bold mb-1 ${
              darkMode ? "text-slate-300" : "text-[#4A3C2A]"
            }`}
          >
            Verdict & Analytics Output Appears Here
          </div>
          <p className="text-xs text-slate-500 font-medium">
            Run a simulation to generate virality index, reach breakdown, and
            actionable hook recommendations.
          </p>
        </div>
      ) : (
        <div className="space-y-6">
          {/* Verdict Header Banner with Rich Gradient */}
          <div
            className={`flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-5 rounded-2xl border ${
              darkMode
                ? "bg-gradient-to-r from-slate-950 via-indigo-950/30 to-slate-950 border-slate-800"
                : "bg-gradient-to-r from-[#FAF7F2] via-[#F4EFE6]/80 to-[#FAF7F2] border-[#DFD6C4]"
            }`}
          >
            <div>
              <div className="flex items-center gap-3">
                <h3 className="text-2xl font-black tracking-tight">
                  {verdict.head}
                </h3>
                <span
                  className={`text-xs font-extrabold px-3 py-1 rounded-full border shadow-xs ${
                    darkMode
                      ? "bg-indigo-500/20 text-indigo-300 border-indigo-500/40"
                      : "bg-[#EFE9DC] text-[#4A3924] border-[#DFD6C4]"
                  }`}
                >
                  {verdict.verdict}
                </span>
              </div>
              <p
                className={`text-xs mt-1.5 leading-relaxed max-w-3xl font-medium ${
                  darkMode ? "text-slate-400" : "text-[#5C4D3B]"
                }`}
              >
                {verdict.sub}
              </p>
            </div>

            <div className="sm:text-right sm:pl-4">
              <div className="text-[10px] uppercase font-black text-slate-400 mb-0.5 tracking-wider">
                Virality Index
              </div>
              <div className="text-3xl font-black text-indigo-600 dark:text-indigo-400 font-mono">
                {verdict.virality}
                <span className="text-sm text-slate-400 font-normal">/100</span>
              </div>
            </div>
          </div>

          {/* KPI Cards Grid */}
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
            <div
              className={`p-3.5 rounded-xl border shadow-xs ${
                darkMode
                  ? "bg-slate-950/60 border-slate-800"
                  : "bg-[#FAF8F5] border-[#E5DEC9]"
              }`}
            >
              <div className="text-[10px] font-bold text-slate-500 mb-1">
                TOTAL REACH
              </div>
              <div className="text-xl font-black text-indigo-600 dark:text-indigo-400 font-mono">
                {verdict.reached}
              </div>
              <div className="text-[10px] text-slate-500 mt-1 font-medium">
                {verdict.reached} of {population} agents
              </div>
            </div>

            <div
              className={`p-3.5 rounded-xl border shadow-xs ${
                darkMode
                  ? "bg-slate-950/60 border-slate-800"
                  : "bg-[#FAF8F5] border-[#E5DEC9]"
              }`}
            >
              <div className="text-[10px] font-bold text-slate-500 mb-1">
                IN / OUT DEMO RATIO
              </div>
              <div className="text-xl font-black font-mono">
                {verdict.inPct}% / {100 - verdict.inPct}%
              </div>
              <div
                className={`h-1.5 rounded-full mt-2 overflow-hidden ${
                  darkMode ? "bg-slate-800" : "bg-slate-200"
                }`}
              >
                <div
                  className="h-full bg-indigo-600 rounded-full"
                  style={{ width: `${verdict.inPct}%` }}
                />
              </div>
            </div>

            <div
              className={`p-3.5 rounded-xl border shadow-xs ${
                darkMode
                  ? "bg-slate-950/60 border-slate-800"
                  : "bg-[#FAF8F5] border-[#E5DEC9]"
              }`}
            >
              <div className="text-[10px] font-bold text-slate-500 mb-1">
                ORGANIC SHARE RATE
              </div>
              <div className="text-xl font-black text-emerald-600 dark:text-emerald-400 font-mono">
                {verdict.shareRate}%
              </div>
              <div className="text-[10px] text-slate-500 mt-1 font-medium">
                {verdict.shares} total shares
              </div>
            </div>

            <div
              className={`p-3.5 rounded-xl border shadow-xs ${
                darkMode
                  ? "bg-slate-950/60 border-slate-800"
                  : "bg-[#FAF8F5] border-[#E5DEC9]"
              }`}
            >
              <div className="text-[10px] font-bold text-slate-500 mb-1">
                ALGORITHM SCORE
              </div>
              <div className="text-xl font-black text-purple-600 dark:text-purple-400 font-mono">
                {verdict.algoScore}
              </div>
              <div className="text-[10px] text-slate-500 mt-1 font-medium">
                Push Probability
              </div>
            </div>

            <div
              className={`p-3.5 rounded-xl border shadow-xs ${
                darkMode
                  ? "bg-slate-950/60 border-slate-800"
                  : "bg-[#FAF8F5] border-[#E5DEC9]"
              }`}
            >
              <div className="text-[10px] font-bold text-slate-500 mb-1">
                ENGAGEMENT %
              </div>
              <div className="text-xl font-black text-amber-600 dark:text-amber-400 font-mono">
                {verdict.engagement}%
              </div>
              <div className="text-[10px] text-slate-500 mt-1 font-medium">
                Non-skip rate
              </div>
            </div>
          </div>

          {analysis && <AnalysisPanel analysis={analysis} darkMode={darkMode} />}

          {/* Recharts Analytics Charts */}
          <ResultsCharts
            darkMode={darkMode}
            verdict={verdict}
            funnelData={funnelData}
            demographicPieData={demographicPieData}
          />
        </div>
      )}
    </section>
  );
}
