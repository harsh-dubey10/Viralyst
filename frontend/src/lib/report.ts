import { Analysis, Verdict } from "../types";

// Plain-text run summary for the "Copy summary" button.
export function buildSummaryText(verdict: Verdict, analysis: Analysis | null, runId: string, population: number) {
  const lines = [
    `Viralyst simulation ${runId}`.trim(),
    `Verdict: ${verdict.head} ${verdict.sub}`,
    `Virality score: ${verdict.virality}/100`,
    `Reached ${verdict.reached}/${population} · ${verdict.engagement}% engaged · ${verdict.shareRate}% shared (${verdict.shares})`,
  ];
  if (analysis?.recommendations.length) {
    lines.push("", "How to improve:", ...analysis.recommendations.map((r) => `- ${r}`));
  }
  return lines.join("\n");
}

// Trigger a browser download of `data` as pretty-printed JSON.
export function downloadJSON(data: unknown, filename: string) {
  const blob = new Blob([JSON.stringify(data, null, 2)], {
    type: "application/json",
  });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
