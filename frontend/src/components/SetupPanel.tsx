import React from "react";
import { Sparkles, Film, Users, Zap, HelpCircle } from "lucide-react";
import { CAMPAIGN_PRESETS, CampaignPreset } from "../constants";

interface SetupPanelProps {
  darkMode: boolean;
  applyPreset: (preset: CampaignPreset) => void;
  videoPreview: string | null;
  fileName: string | null;
  fileInputRef: React.RefObject<HTMLInputElement | null>;
  handleFile: (e: React.ChangeEvent<HTMLInputElement>) => void;
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

// Quick preset tabs plus the campaign config form (section 01).
export default function SetupPanel({
  darkMode,
  applyPreset,
  videoPreview,
  fileName,
  fileInputRef,
  handleFile,
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
  return (
    <>
      {/* Preset Quick Fill Tabs (Warm Beige, Sage, Terracotta, Lavender) */}
      <div className="flex items-center gap-2 overflow-x-auto pb-1 text-xs">
        <span
          className={`font-bold uppercase tracking-wider text-[10px] flex items-center gap-1 shrink-0 ${
            darkMode ? "text-slate-400" : "text-[#756550]"
          }`}
        >
          <Sparkles className="w-3.5 h-3.5 text-indigo-500" /> Quick Presets:
        </span>
        {CAMPAIGN_PRESETS.map((preset) => (
          <button
            key={preset.title}
            onClick={() => applyPreset(preset)}
            className={`shrink-0 border px-3 py-1.5 rounded-xl text-xs shadow-xs hover:shadow-sm transition flex items-center gap-1.5 font-medium ${preset.tabStyle}`}
          >
            <span className={`px-1.5 py-0.5 rounded-md font-bold text-[10px] ${preset.badgeStyle}`}>
              {preset.tag}
            </span>
            <span>{preset.title}</span>
          </button>
        ))}
      </div>

      {/* SECTION 01: CAMPAIGN CONFIG */}
      <section
        className={`border rounded-2xl shadow-xs overflow-hidden ${
          darkMode
            ? "border-slate-800/90 bg-slate-900/60 backdrop-blur-xl"
            : "border-[#E5DEC9] bg-white/95 backdrop-blur-md"
        }`}
      >
        <div
          className={`px-5 py-3 border-b flex items-center justify-between ${
            darkMode
              ? "border-slate-800 bg-slate-950/60"
              : "border-[#EAE3D2] bg-gradient-to-r from-[#FBF9F4] via-[#F4EFE6]/60 to-white"
          }`}
        >
          <div className="flex items-center gap-2 text-xs font-bold tracking-wider">
            <span className="w-5 h-5 rounded-lg bg-indigo-600 text-white flex items-center justify-center text-[10px] font-black shadow-xs">
              01
            </span>
            <span className={darkMode ? "text-slate-200" : "text-[#3D3021]"}>
              CAMPAIGN HOOK & AUDIENCE SETUP
            </span>
          </div>
          <span className="hidden sm:inline text-[11px] text-slate-500 font-semibold">
            Describe the audience and the first 3 seconds
          </span>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 xl:[grid-template-columns:1.1fr_1.2fr_1.4fr_auto_auto]">
          {/* Video / Creative Uploader */}
          <div
            className={`p-4 border-b xl:border-b-0 md:border-r flex flex-col justify-between ${
              darkMode ? "border-slate-800" : "border-[#EAE3D2]"
            }`}
          >
            <div>
              <div
                className={`text-[11px] font-bold tracking-wider mb-2 flex items-center gap-1.5 ${
                  darkMode ? "text-slate-400" : "text-[#695B48]"
                }`}
              >
                <Film className="w-3.5 h-3.5 text-indigo-500" /> CONTENT VIDEO
              </div>
              <div
                onClick={() => !videoPreview && fileInputRef.current?.click()}
                className={`border-2 border-dashed rounded-xl p-3 text-center cursor-pointer transition ${
                  darkMode
                    ? "border-slate-700 bg-slate-950/50 hover:bg-slate-950/80 hover:border-indigo-400"
                    : "border-[#DCD3BE] bg-[#FAF8F5]/80 hover:bg-[#F3EFE6] hover:border-indigo-400"
                }`}
              >
                {videoPreview ? (
                  <div className="space-y-2">
                    <div className="overflow-hidden rounded-lg border border-slate-700 h-24 bg-black">
                      <video
                        key={videoPreview}
                        src={videoPreview}
                        controls
                        className="w-full h-full object-cover"
                      />
                    </div>
                    <div className="flex items-center justify-between text-left">
                      <span className="text-xs font-bold truncate max-w-[120px]">
                        {fileName}
                      </span>
                      <span className="text-[10px] text-emerald-800 bg-emerald-100 border border-emerald-300 px-1.5 py-0.5 rounded font-mono font-bold">
                        Ready
                      </span>
                    </div>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        if (fileInputRef.current) fileInputRef.current.value = "";
                        fileInputRef.current?.click();
                      }}
                      className={`text-[11px] w-full border rounded-lg py-1 transition font-semibold ${
                        darkMode
                          ? "border-slate-700 hover:bg-slate-800 text-slate-200"
                          : "border-[#D6CCA8] hover:bg-[#EBE3CF] text-[#4A3924]"
                      }`}
                    >
                      Replace Video
                    </button>
                  </div>
                ) : (
                  <div className="flex flex-col items-center justify-center py-5">
                    <Film className="w-6 h-6 text-slate-400 mb-1.5" />
                    <div
                      className={`text-xs font-bold ${
                        darkMode ? "text-slate-200" : "text-[#4A3C2A]"
                      }`}
                    >
                      Upload Video Hook
                    </div>
                    <div className="text-[10px] text-slate-500 mt-0.5 font-medium">
                      MP4, MOV, WEBM (Max 50MB)
                    </div>
                  </div>
                )}
              </div>
              <input
                ref={fileInputRef}
                type="file"
                accept="video/*"
                className="hidden"
                onChange={handleFile}
              />
            </div>
          </div>

          {/* Target Demographic */}
          <div
            className={`p-4 border-b xl:border-b-0 md:border-r flex flex-col justify-between ${
              darkMode ? "border-slate-800" : "border-[#EAE3D2]"
            }`}
          >
            <div>
              <div
                className={`text-[11px] font-bold tracking-wider mb-2 flex items-center gap-1.5 ${
                  darkMode ? "text-slate-400" : "text-[#695B48]"
                }`}
              >
                <Users className="w-3.5 h-3.5 text-indigo-500" /> TARGET
                DEMOGRAPHIC
              </div>
              <textarea
                value={demographic}
                onChange={(e) => setDemographic(e.target.value)}
                placeholder="e.g. Gen Z tech founders, AI developers, and product managers in North America & India"
                className={`w-full border rounded-xl p-2.5 outline-none resize-none text-xs min-h-[105px] transition ${
                  darkMode
                    ? "border-slate-800 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-900/30 bg-slate-950/70 text-slate-100 placeholder-slate-500"
                    : "border-[#DFD6C4] focus:border-indigo-500 focus:ring-2 focus:ring-indigo-100 bg-[#FCFBF8] text-slate-800 placeholder-slate-400"
                }`}
                rows={4}
              />
            </div>
          </div>

          {/* Content Description */}
          <div
            className={`p-4 border-b xl:border-b-0 md:border-r flex flex-col justify-between ${
              darkMode ? "border-slate-800" : "border-[#EAE3D2]"
            }`}
          >
            <div>
              <div
                className={`text-[11px] font-bold tracking-wider mb-2 flex items-center gap-1.5 ${
                  darkMode ? "text-slate-400" : "text-[#695B48]"
                }`}
              >
                <Sparkles className="w-3.5 h-3.5 text-purple-500" /> HOOK &
                CORE MESSAGE
              </div>
              <textarea
                value={contentDescription}
                onChange={(e) => setContentDescription(e.target.value)}
                placeholder="Describe your first 3 seconds, key insight, and CTA (evaluated by each persona)"
                className={`w-full border rounded-xl p-2.5 outline-none resize-none text-xs min-h-[105px] transition ${
                  darkMode
                    ? "border-slate-800 focus:border-purple-500 focus:ring-2 focus:ring-purple-900/30 bg-slate-950/70 text-slate-100 placeholder-slate-500"
                    : "border-[#DFD6C4] focus:border-purple-500 focus:ring-2 focus:ring-purple-100 bg-[#FCFBF8] text-slate-800 placeholder-slate-400"
                }`}
                rows={4}
              />
            </div>
          </div>

          {/* Population & Boost */}
          <div
            className={`p-4 border-b md:border-b-0 md:border-r xl:min-w-48 flex flex-col justify-between gap-3 ${
              darkMode ? "border-slate-800" : "border-[#EAE3D2]"
            }`}
          >
            <div>
              <div
                className={`text-[11px] font-bold tracking-wider mb-1.5 flex items-center justify-between ${
                  darkMode ? "text-slate-400" : "text-[#695B48]"
                }`}
              >
                <span>AGENT POPULATION</span>

              </div>
              <select
                value={population}
                onChange={(e) => setPopulation(+e.target.value)}
                className={`w-full border rounded-xl px-2.5 py-1.5 text-xs outline-none font-bold focus:border-indigo-500 shadow-xs ${
                  darkMode
                    ? "border-slate-700 bg-slate-950 text-slate-100"
                    : "border-[#DFD6C4] bg-[#FCFBF8] text-slate-800"
                }`}
              >
                <option value={50}>50 AI Personas</option>
                <option value={100}>100 AI Personas</option>
                <option value={150}>150 AI Personas</option>
                <option value={200}>200 AI Personas</option>
                <option value={250}>250 AI Personas</option>
              </select>
            </div>

            <div>
              <div
                className={`flex items-center justify-between text-[11px] mb-1 font-semibold ${
                  darkMode ? "text-slate-400" : "text-[#695B48]"
                }`}
              >
                <span className="flex items-center gap-1">
                  SEED BOOST
                  <span
                    title="Number of initial seed personas tested in Round 1 (Boost × 4)"
                    className="cursor-help text-slate-400"
                  >
                    <HelpCircle className="w-3 h-3" />
                  </span>
                </span>
                <span className="font-bold text-indigo-600 dark:text-indigo-400 font-mono">
                  {boost * 4} seeds
                </span>
              </div>
              <input
                type="range"
                min={1}
                max={10}
                step={1}
                value={boost}
                onChange={(e) => setBoost(+e.target.value)}
                className="w-full h-1.5 accent-indigo-600 bg-slate-200 dark:bg-slate-700 rounded-lg cursor-pointer"
              />
            </div>
          </div>

          {/* Run Button with Dynamic Gradient */}
          <button
            onClick={runSim}
            disabled={running}
            className="px-8 py-5 xl:py-0 text-xs font-black tracking-wider transition min-w-36 flex flex-col items-center justify-center gap-1.5 text-white relative overflow-hidden group shadow-md shadow-indigo-500/20"
            style={{
              background: running
                ? "#64748B"
                : "linear-gradient(135deg, #2563EB 0%, #4F46E5 50%, #7C3AED 100%)",
            }}
          >
            {running ? (
              <>
                <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin" />
                <span className="text-[10px] font-mono text-indigo-100">
                  SIMULATING...
                </span>
              </>
            ) : (
              <>
                <Zap className="w-5 h-5 text-white group-hover:scale-110 transition-transform" />
                <span className="text-xs">RUN SIMULATION</span>
              </>
            )}
          </button>
        </div>
      </section>
    </>
  );
}
