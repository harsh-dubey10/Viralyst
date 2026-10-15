import React from "react";
import { ArrowDown } from "lucide-react";
import { primaryButton } from "./ui";

// Words rise one after another out of clipped line boxes (Editions .title__word).
function Reveal({ text, delay = 0, step = 70, className = "" }: { text: string; delay?: number; step?: number; className?: string }) {
  return (
    <span className={className}>
      {text.split(" ").map((w, i) => (
        <React.Fragment key={i}>
          <span className="reveal-word">
            <span style={{ animationDelay: `${delay + i * step}ms` }}>{w}</span>
          </span>{" "}
        </React.Fragment>
      ))}
    </span>
  );
}

const STEPS = [
  { n: "01", title: "Describe the hook", body: "What viewers see, hear and read in the first three seconds — or upload the video." },
  { n: "02", title: "A synthetic crowd reacts", body: "Up to 250 personas watch, skip, like, comment and share, each for their own reasons." },
  { n: "03", title: "Watch it spread", body: "Shares travel through the network, the algorithm decides whether to push it, you get the verdict." },
];

export default function Hero() {
  return (
    <section id="top" className="relative flex min-h-[calc(100svh-56px)] flex-col justify-between px-4 pb-10 pt-14 sm:px-8 sm:pt-20">
      <div className="mx-auto w-full max-w-[1680px]">
        <p className="mb-6 animate-fade-up text-[11px] font-medium uppercase tracking-[0.18em] text-ink/55">Synthetic audience · pre-publish testing</p>
        <h1 className="headline max-w-[14ch] text-[clamp(3.25rem,1.5rem+8.5vw,11rem)]">
          <Reveal text="Test the hook" />
          <br />
          <Reveal text="before the" delay={260} />
          <Reveal text="feed does." delay={400} className="font-serif font-normal italic tracking-[-0.02em] text-accent" />
        </h1>
        <div className="mt-10 flex max-w-2xl animate-fade-up flex-col gap-6 sm:flex-row sm:items-center" style={{ animationDelay: "700ms" }}>
          <p className="font-serif text-[clamp(1.35rem,1rem+1vw,1.9rem)] leading-[1.1] tracking-[-0.02em] text-ink/80">
            Hundreds of AI viewers react to your post, share it with each other, and show you whether it breaks out — before you spend a single impression.
          </p>
        </div>
        <div className="mt-8 flex animate-fade-up flex-wrap gap-3" style={{ animationDelay: "850ms" }}>
          <a href="#simulate" className={`${primaryButton} h-11 px-5 text-[15px]`}>
            Run a simulation <ArrowDown className="h-4 w-4" />
          </a>
        </div>
      </div>

      <ol className="mx-auto mt-16 grid w-full max-w-[1680px] gap-px overflow-hidden rounded-xl border border-ink/10 bg-ink/10 sm:grid-cols-3">
        {STEPS.map((s, i) => (
          <li key={s.n} className="animate-fade-up bg-surface/80 p-5 backdrop-blur-md" style={{ animationDelay: `${1000 + i * 120}ms` }}>
            <div className="font-display text-sm font-bold text-accent">{s.n}</div>
            <div className="mt-6 font-display text-xl font-bold tracking-tight">{s.title}</div>
            <p className="mt-1.5 text-sm text-ink/60">{s.body}</p>
          </li>
        ))}
      </ol>
    </section>
  );
}
