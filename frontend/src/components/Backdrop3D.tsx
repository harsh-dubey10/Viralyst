import React, { useEffect, useRef } from "react";
import { BackdropMood, BackdropScene, ReactionMix } from "../three/BackdropScene";
import { webglAvailable } from "../lib/env";
import { ACTION_STYLES } from "../constants";

interface Backdrop3DProps {
  dark: boolean;
  mood: BackdropMood;
  mix: ReactionMix | null;
}

const themeFor = (dark: boolean) => ({
  dark,
  colors: {
    share: ACTION_STYLES.share.fill[dark ? "dark" : "light"],
    comment: ACTION_STYLES.comment.fill[dark ? "dark" : "light"],
    like: ACTION_STYLES.like.fill[dark ? "dark" : "light"],
  },
});

// Full-viewport three.js layer behind the page. Lazy-loaded so three.js never blocks
// first paint; renders nothing where WebGL is unavailable.
export default function Backdrop3D({ dark, mood, mix }: Backdrop3DProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const sceneRef = useRef<BackdropScene | null>(null);

  useEffect(() => {
    if (!canvasRef.current || !webglAvailable()) return;
    try {
      sceneRef.current = new BackdropScene(canvasRef.current, themeFor(dark));
    } catch {
      return; // context creation can still fail (GPU blocklist); the page works without it
    }
    return () => {
      sceneRef.current?.dispose();
      sceneRef.current = null;
    };
    // The scene is created once; theme and mood updates go through the effects below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => sceneRef.current?.setTheme(themeFor(dark)), [dark]);
  useEffect(() => sceneRef.current?.setMood(mood, mix), [mood, mix]);

  return <canvas ref={canvasRef} aria-hidden="true" className="pointer-events-none fixed inset-0 -z-0 h-full w-full opacity-90" />;
}
