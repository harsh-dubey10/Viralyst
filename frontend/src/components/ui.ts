// Shared Tailwind class strings so every card, label and input looks the same.
// Shopify Editions style: hairline borders, small radii, soft offset shadow, pill buttons.
export const card = "rounded-xl border border-ink/10 bg-surface/85 shadow-card backdrop-blur-md";
export const fieldLabel = "block text-sm font-medium text-ink";
export const fieldHint = "text-xs text-ink/55";
export const input =
  "w-full rounded-lg border border-ink/15 bg-page/40 px-3 py-2 text-sm text-ink placeholder-ink/35 outline-none transition focus:border-accent focus:ring-4 focus:ring-accent/15";
export const sectionTitle = "text-[11px] font-medium uppercase tracking-[0.14em] text-ink/50";
export const pill =
  "inline-flex h-[34px] items-center gap-1.5 rounded-full px-3.5 text-sm font-medium transition duration-300 ease-spring focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent";
export const secondaryButton = `${pill} border border-ink/15 text-ink/80 hover:bg-ink hover:text-page`;
export const primaryButton = `${pill} bg-ink text-page hover:bg-accent hover:text-white hover:shadow-glow`;
