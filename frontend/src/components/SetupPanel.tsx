import React from "react";
import { Film, X, Play, ChevronDown } from "lucide-react";
import { EXAMPLES, Example, NICHES, seedSize } from "../constants";
import { card, fieldHint, fieldLabel, input, primaryButton, sectionTitle } from "./ui";

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
  const seed = seedSize(population, boost);

  return (
    <section className={`${card} lg:sticky lg:top-[76px]`} aria-label="Simulation setup">
      <form
        className="space-y-6 p-5 sm:p-6"
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
                  className={`rounded-full border px-3 py-1 text-xs font-medium transition duration-300 ease-spring ${
                    active ? "scale-105 border-accent bg-accent text-white shadow-glow" : "border-ink/15 text-ink/70 hover:border-ink/40 hover:text-ink"
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
            Who is it for? <span className="font-normal text-ink/40">(optional)</span>
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
            <div className="flex items-center gap-3 rounded-lg border border-ink/15 p-2">
              <video key={videoPreview} src={videoPreview} muted className="h-14 w-10 shrink-0 rounded bg-black object-cover" />
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-medium">{fileName}</div>
                <div className={fieldHint}>The opening is transcribed and described for the personas.</div>
              </div>
              <button
                type="button"
                onClick={removeVideo}
                className="rounded-md p-1.5 text-ink/45 hover:bg-ink/10 hover:text-ink"
                aria-label="Remove video"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              className="flex w-full items-center justify-center gap-2 rounded-lg border border-dashed border-ink/25 px-3 py-2.5 text-sm text-ink/65 transition hover:border-accent hover:bg-accent/10 hover:text-accent"
            >
              <Film className="h-4 w-4" />
              Upload the video <span className="whitespace-nowrap text-ink/40">· up to 50 MB</span>
            </button>
          )}
          <input ref={fileInputRef} type="file" accept="video/*" className="hidden" onChange={handleFile} data-testid="video-input" />
        </div>

        {/* Advanced */}
        <details className="group rounded-lg border border-ink/10">
          <summary className="flex cursor-pointer list-none items-center justify-between px-3 py-2 text-sm font-medium text-ink/80">
            Simulation settings
            <span className="flex items-center gap-2 text-xs font-normal text-ink/50">
              {population} viewers · {seed} seed
              <ChevronDown className="h-4 w-4 transition group-open:rotate-180" />
            </span>
          </summary>
          <div className="space-y-4 border-t border-ink/10 px-3 py-3">
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
                <span className="text-sm tabular-nums text-ink/50">{seed} viewers</span>
              </div>
              <input
                id="boost"
                type="range"
                min={1}
                max={10}
                value={boost}
                onChange={(e) => setBoost(+e.target.value)}
                className="w-full accent-[rgb(var(--accent))]"
              />
              <p className={fieldHint}>How many people the platform shows it to first, before shares and recommendations.</p>
            </div>
          </div>
        </details>

        <div className="space-y-2">
          <button
            type="submit"
            disabled={running || !!missing}
            className={`${primaryButton} h-11 w-full justify-center text-[15px] font-semibold disabled:cursor-not-allowed disabled:bg-ink/15 disabled:text-ink/40 disabled:shadow-none`}
          >
            {running ? (
              <>
                <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
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

        <div className="space-y-2 border-t border-ink/10 pt-4">
          <div className={sectionTitle}>Try an example</div>
          <div className="flex flex-wrap gap-1.5">
            {EXAMPLES.map((ex) => (
              <button
                type="button"
                key={ex.label}
                onClick={() => applyExample(ex)}
                className="rounded-full bg-ink/[.06] px-3 py-1 text-xs font-medium text-ink/75 transition duration-300 ease-spring hover:-translate-y-0.5 hover:bg-ink hover:text-page"
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
