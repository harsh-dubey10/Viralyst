import * as THREE from "three";
export { prefersReducedMotion } from "../lib/env";

// Retina screens get at most 1.5× — the scenes are soft and full-DPR fill costs a lot.
export const pixelRatio = () => Math.min(window.devicePixelRatio || 1, 1.5);

// Round, soft-edged points with per-vertex colour, size and alpha.
export function softPointsMaterial() {
  return new THREE.ShaderMaterial({
    transparent: true,
    depthWrite: false,
    uniforms: { uPixelRatio: { value: pixelRatio() }, uPulse: { value: 0 } },
    vertexShader: /* glsl */ `
      attribute vec3 color;
      attribute float size;
      attribute float alpha;
      uniform float uPixelRatio;
      uniform float uPulse;
      varying vec3 vColor;
      varying float vAlpha;
      void main() {
        vColor = color;
        vAlpha = alpha;
        vec4 mv = modelViewMatrix * vec4(position, 1.0);
        float pulse = 1.0 + uPulse * 0.35 * sin(position.x * 7.0 + position.y * 5.0 + uPulse * 20.0);
        gl_PointSize = size * pulse * uPixelRatio * (6.0 / -mv.z);
        gl_Position = projectionMatrix * mv;
      }`,
    fragmentShader: /* glsl */ `
      varying vec3 vColor;
      varying float vAlpha;
      void main() {
        float d = length(gl_PointCoord - 0.5);
        if (d > 0.5) discard;
        gl_FragColor = vec4(vColor, vAlpha * smoothstep(0.5, 0.15, d));
      }`,
  });
}

// Evenly spread n points over a sphere (Fibonacci lattice).
export function fibonacciSphere(n: number, radius: number): THREE.Vector3[] {
  const pts: THREE.Vector3[] = [];
  const golden = Math.PI * (3 - Math.sqrt(5));
  for (let i = 0; i < n; i++) {
    const y = 1 - (i / Math.max(n - 1, 1)) * 2;
    const r = Math.sqrt(1 - y * y);
    const t = golden * i;
    pts.push(new THREE.Vector3(Math.cos(t) * r * radius, y * radius, Math.sin(t) * r * radius));
  }
  return pts;
}
