import { Verdict } from "../types";

// Plain-text run summary for the "Copy Summary" button.
export function buildSummaryText(verdict: Verdict, runId: string, population: number) {
  return `Viralyst Simulation Report [${runId}]
Headline: ${verdict.head} (${verdict.verdict})
Summary: ${verdict.sub}
Reach: ${verdict.reached}/${population} (${verdict.inPct}% In-Target)
Share Rate: ${verdict.shareRate}% (${verdict.shares} organic shares)
Virality Index: ${verdict.virality}/100`;
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
