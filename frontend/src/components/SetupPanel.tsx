import React from "react";
import { Film, X, Play, ChevronDown } from "lucide-react";
import { EXAMPLES, Example, NICHES } from "../constants";
import { card, fieldHint, fieldLabel, input, sectionTitle } from "./ui";

interface SetupPanelProps {
  applyExample: (example: Example) => void;
  videoPreview: string | null;
  fileName: string | null;
  fileInputRef: React.RefObject<HTMLInputElement | null>;
  handleFile: (e: React.ChangeEvent<HTMLInputElement>) => void;
  removeVideo: () => void;
  niche: string;
  setNiche: (value: string) => void;
  demographic: string;
  setDemographic: (value: string) => void;
  contentDescription: string;
  setContentDescription: (value: string) => void;
  population: number;
  setPopulation: (value: number) => void;
  boost: number;
  setBoost: (value: number) => void;
  running: boolean;
  runSim: () => void;
}

// What's still needed before a run would mean anything ("" when ready).
export function missingInput(niche: string, demographic: string, hook: string, hasVideo: boolean) {
  if (!niche.trim() && !demographic.trim()) return "Pick a niche or describe your audience.";
  if (!hook.trim() && !hasVideo) return "Describe your first 3 seconds or upload the video.";
  return "";
}

export default function SetupPanel({
  applyExample,
  videoPreview,
  fileName,
  fileInputRef,
  handleFile,
  removeVideo,
  niche,
  setNiche,
  demographic,
  setDemographic,
  contentDescription,
  setContentDescription,
  population,
  setPopulation,
  boost,
  setBoost,
  running,
  runSim,
}: SetupPanelProps) {
  const missing = missingInput(niche, demographic, contentDescription, !!videoPreview);
  const nicheKey = niche.trim().toLowerCase();

  return (
    <section className={`${card} lg:sticky lg:top-20`} aria-label="Simulation setup">
      <form
        className="space-y-6 p-5"
        onSubmit={(e) => {
          e.preventDefault();
          if (!missing) runSim();
        }}
      >
        {/* Niche */}
        <div className="space-y-2">
          <label htmlFor="niche" className={fieldLabel}>
            Your niche
          </label>
          <input
            id="niche"
            value={niche}
            onChange={(e) => setNiche(e.target.value)}
            placeholder="Pick one below or type your own — e.g. chess, knitting"
            maxLength={80}
            className={input}
          />
          <div className="flex flex-wrap gap-1.5">
            {NICHES.map((n) => {
              const active = nicheKey === n.value.toLowerCase();
              return (
                <button
                  type="button"
                  key={n.value}
                  onClick={() => setNiche(active ? "" : n.value)}
                  aria-pressed={active}
                  className={`rounded-full border px-2.5 py-1 text-xs font-medium transition ${
                    active
                      ? "border-indigo-600 bg-indigo-600 text-white dark:border-indigo-500 dark:bg-indigo-500"
                      : "border-zinc-200 text-zinc-600 hover:border-zinc-300 hover:bg-zinc-50 dark:border-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-800"
                  }`}
                >
                  {n.label}
                </button>
              );
            })}
          </div>
        </div>

        {/* Audience */}
        <div className="space-y-2">
          <label htmlFor="demographic" className={fieldLabel}>
            Who is it for? <span className="font-normal text-zinc-400">(optional)</span>
          </label>
          <textarea
            id="demographic"
            value={demographic}
            onChange={(e) => setDemographic(e.target.value)}
            placeholder="Age, location, what they care about — e.g. busy parents in the US, 28–45"
            rows={2}
            maxLength={500}
            className={`${input} resize-none`}
          />
        </div>

        {/* Hook */}
        <div className="space-y-2">
          <label htmlFor="hook" className={fieldLabel}>
            Your hook
          </label>
          <p className={fieldHint}>What viewers see, hear and read in the first 3 seconds.</p>
          <textarea
            id="hook"
            value={contentDescription}
            onChange={(e) => setContentDescription(e.target.value)}
            placeholder="e.g. This 10-minute pasta uses 4 ingredients and beats every restaurant version."
            rows={4}
            maxLength={2000}
            className={`${input} resize-none`}
          />

          {videoPreview ? (
            <div className="flex items-center gap-3 rounded-lg border border-zinc-200 p-2 dark:border-zinc-700">
              <video key={videoPreview} src={videoPreview} muted className="h-14 w-10 shrink-0 rounded bg-black object-cover" />
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-medium">{fileName}</div>
                <div className={fieldHint}>The opening is transcribed and described for the personas.</div>
              </div>
              <button
                type="button"
                onClick={removeVideo}
                className="rounded-md p-1.5 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
                aria-label="Remove video"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              className="flex w-full items-center justify-center gap-2 rounded-lg border border-dashed border-zinc-300 px-3 py-2.5 text-sm text-zinc-600 transition hover:border-indigo-400 hover:bg-indigo-50/50 hover:text-indigo-700 dark:border-zinc-700 dark:text-zinc-400 dark:hover:border-indigo-500 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-300"
            >
              <Film className="h-4 w-4" />
              Upload the video <span className="whitespace-nowrap text-zinc-400">· up to 50 MB</span>
            </button>
          )}
          <input ref={fileInputRef} type="file" accept="video/*" className="hidden" onChange={handleFile} data-testid="video-input" />
        </div>

        {/* Advanced */}
        <details className="group rounded-lg border border-zinc-200 dark:border-zinc-800">
          <summary className="flex cursor-pointer list-none items-center justify-between px-3 py-2 text-sm font-medium text-zinc-700 dark:text-zinc-300">
            Simulation settings
            <span className="flex items-center gap-2 text-xs font-normal text-zinc-500">
              {population} viewers · {boost * 4} seed
              <ChevronDown className="h-4 w-4 transition group-open:rotate-180" />
            </span>
          </summary>
          <div className="space-y-4 border-t border-zinc-200 px-3 py-3 dark:border-zinc-800">
            <div className="space-y-1.5">
              <label htmlFor="population" className={fieldLabel}>
                Audience size
              </label>
              <select id="population" value={population} onChange={(e) => setPopulation(+e.target.value)} className={input}>
                {[50, 100, 150, 200, 250].map((n) => (
                  <option key={n} value={n}>
                    {n} simulated viewers
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <label htmlFor="boost" className={fieldLabel}>
                  First-round reach
                </label>
                <span className="text-sm tabular-nums text-zinc-500">{boost * 4} viewers</span>
              </div>
              <input
                id="boost"
                type="range"
                min={1}
                max={10}
                value={boost}
                onChange={(e) => setBoost(+e.target.value)}
                className="w-full accent-indigo-600"
              />
              <p className={fieldHint}>How many people the platform shows it to first, before shares and recommendations.</p>
            </div>
          </div>
        </details>

        <div className="space-y-2">
          <button
            type="submit"
            disabled={running || !!missing}
            className="flex w-full items-center justify-center gap-2 rounded-lg bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500 disabled:cursor-not-allowed disabled:bg-zinc-300 disabled:text-zinc-500 dark:disabled:bg-zinc-800 dark:disabled:text-zinc-500"
          >
            {running ? (
              <>
                <span className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                Simulating…
              </>
            ) : (
              <>
                <Play className="h-4 w-4" />
                Run simulation
              </>
            )}
          </button>
          {missing && !running && <p className={`${fieldHint} text-center`}>{missing}</p>}
        </div>

        <div className="space-y-2 border-t border-zinc-100 pt-4 dark:border-zinc-800">
          <div className={sectionTitle}>Try an example</div>
          <div className="flex flex-wrap gap-1.5">
            {EXAMPLES.map((ex) => (
              <button
                type="button"
                key={ex.label}
                onClick={() => applyExample(ex)}
                className="rounded-md bg-zinc-100 px-2 py-1 text-xs font-medium text-zinc-700 transition hover:bg-zinc-200 dark:bg-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-700"
              >
                {ex.label}
              </button>
            ))}
          </div>
        </div>
      </form>
    </section>
  );
}
