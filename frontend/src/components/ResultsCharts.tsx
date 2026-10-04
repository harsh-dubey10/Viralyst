import React from "react";
import {
  ResponsiveContainer,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip as RechartsTooltip,
  Cell,
  PieChart,
  Pie,
} from "recharts";
import { ChartDatum, Verdict } from "../types";

interface ResultsChartsProps {
  darkMode: boolean;
  verdict: Verdict;
  funnelData: ChartDatum[];
  demographicPieData: ChartDatum[];
}

// Recharts engagement funnel and in/out-of-target cohort split.
export default function ResultsCharts({
  darkMode,
  verdict,
  funnelData,
  demographicPieData,
}: ResultsChartsProps) {
  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 pt-2">
      <div
        className={`p-4 rounded-xl border shadow-xs ${
          darkMode
            ? "bg-slate-950/60 border-slate-800"
            : "bg-[#FAF8F5] border-[#E5DEC9]"
        }`}
      >
        <div className="text-xs font-bold mb-3 flex items-center justify-between">
          <span>AUDIENCE ENGAGEMENT FUNNEL</span>
          <span className="text-[10px] text-slate-500 font-mono font-medium">
            Action Breakdown
          </span>
        </div>
        <div className="h-44">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={funnelData} layout="vertical">
              <XAxis
                type="number"
                stroke={darkMode ? "#64748B" : "#94A3B8"}
                fontSize={10}
              />
              <YAxis
                type="category"
                dataKey="name"
                stroke={darkMode ? "#94A3B8" : "#64748B"}
                fontSize={11}
                width={70}
              />
              <RechartsTooltip
                contentStyle={{
                  backgroundColor: darkMode ? "#0F172A" : "#FFFFFF",
                  borderColor: darkMode ? "#334155" : "#E2E8F0",
                  color: darkMode ? "#F8FAFC" : "#0F172A",
                  borderRadius: "10px",
                  fontSize: "11px",
                  boxShadow: "0 10px 15px -3px rgb(0 0 0 / 0.1)",
                }}
              />
              <Bar dataKey="value" radius={[0, 6, 6, 0]}>
                {funnelData.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={entry.fill} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div
        className={`p-4 rounded-xl border shadow-xs ${
          darkMode
            ? "bg-slate-950/60 border-slate-800"
            : "bg-[#FAF8F5] border-[#E5DEC9]"
        }`}
      >
        <div className="text-xs font-bold mb-3 flex items-center justify-between">
          <span>COHORT SPLIT (IN-TARGET VS OUT-OF-TARGET)</span>
          <span className="text-[10px] text-slate-500 font-mono font-medium">
            Spread Distribution
          </span>
        </div>
        <div className="h-44 flex items-center justify-center">
          <div className="h-full flex-1 min-w-0">
          <ResponsiveContainer width="100%" height="100%">
            <PieChart>
              <Pie
                data={demographicPieData}
                innerRadius={45}
                outerRadius={65}
                paddingAngle={5}
                dataKey="value"
              >
                {demographicPieData.map((entry, index) => (
                  <Cell key={`pie-cell-${index}`} fill={entry.fill} />
                ))}
              </Pie>
              <RechartsTooltip
                contentStyle={{
                  backgroundColor: darkMode ? "#0F172A" : "#FFFFFF",
                  borderColor: darkMode ? "#334155" : "#E2E8F0",
                  color: darkMode ? "#F8FAFC" : "#0F172A",
                  borderRadius: "10px",
                  fontSize: "11px",
                  boxShadow: "0 10px 15px -3px rgb(0 0 0 / 0.1)",
                }}
              />
            </PieChart>
          </ResponsiveContainer>
          </div>
          <div className="space-y-2 pr-2 sm:pr-6 shrink-0">
            <div className="flex items-center gap-2 text-xs">
              <span className="w-2.5 h-2.5 rounded-full bg-[#4F46E5]" />
              <span className="font-bold">
                In-Target: {verdict.inPct}%
              </span>
            </div>
            <div className="flex items-center gap-2 text-xs">
              <span className="w-2.5 h-2.5 rounded-full bg-[#F43F5E]" />
              <span className="font-bold">
                Out-of-Target: {100 - verdict.inPct}%
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
