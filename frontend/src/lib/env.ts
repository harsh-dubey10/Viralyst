// Browser capability checks. Kept free of three.js imports so the main bundle can use
// them without pulling three.js out of its lazy chunk.

// jsdom (tests) and some locked-down browsers have no WebGL; callers skip 3D entirely then.
export function webglAvailable(): boolean {
  try {
    if (typeof window === "undefined" || !window.WebGLRenderingContext) return false;
    const c = document.createElement("canvas");
    return !!(c.getContext("webgl2") || c.getContext("webgl"));
  } catch {
    return false;
  }
}

export function prefersReducedMotion(): boolean {
  return typeof window.matchMedia === "function" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}
