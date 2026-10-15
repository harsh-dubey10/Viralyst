import * as THREE from "three";
import { fibonacciSphere, pixelRatio, prefersReducedMotion, softPointsMaterial } from "./util";

export type BackdropMood = "idle" | "running" | "done";

// Share of the crowd to colour by each reaction once a result is in.
export interface ReactionMix {
  share: number;
  comment: number;
  like: number;
}

export interface BackdropTheme {
  dark: boolean;
  colors: { share: string; comment: string; like: string };
}

const CROWD = 1400;
const LINKS = 700;

// The page backdrop: a slowly turning sphere of points, one per "viewer" in an imagined
// feed, with faint links between neighbours. It speeds up and pulses while a simulation
// runs, then recolours to the result's reaction mix.
export class BackdropScene {
  private renderer: THREE.WebGLRenderer;
  private scene = new THREE.Scene();
  private camera = new THREE.PerspectiveCamera(42, 1, 0.1, 50);
  private group = new THREE.Group();
  private points: THREE.Points;
  private links: THREE.LineSegments;
  private material = softPointsMaterial();
  private colors: Float32Array;
  private target: Float32Array;
  private base = new THREE.Color();
  private mood: BackdropMood = "idle";
  private pointer = new THREE.Vector2();
  private frame = 0;
  private clock = new THREE.Clock();
  private spin = 0.04;
  private still = prefersReducedMotion();
  private order: number[];
  private theme!: BackdropTheme;
  private mix: ReactionMix | null = null;

  constructor(private canvas: HTMLCanvasElement, theme: BackdropTheme) {
    this.renderer = new THREE.WebGLRenderer({ canvas, alpha: true, antialias: false, powerPreference: "low-power" });
    this.renderer.setPixelRatio(pixelRatio());
    this.camera.position.set(0, 0, 7.2);

    const pts = fibonacciSphere(CROWD, 2.4);
    const pos = new Float32Array(CROWD * 3);
    const size = new Float32Array(CROWD);
    const alpha = new Float32Array(CROWD);
    pts.forEach((p, i) => {
      // A little radial jitter so it reads as a crowd, not a lattice.
      p.multiplyScalar(1 + (Math.random() - 0.5) * 0.08);
      p.toArray(pos, i * 3);
      size[i] = 3 + Math.random() * 4.5;
      alpha[i] = 0.35 + Math.random() * 0.5;
    });
    this.colors = new Float32Array(CROWD * 3);
    this.target = new Float32Array(CROWD * 3);
    // Random but fixed order in which points take on reaction colours.
    this.order = Array.from({ length: CROWD }, (_, i) => i).sort(() => Math.random() - 0.5);

    const geo = new THREE.BufferGeometry();
    geo.setAttribute("position", new THREE.BufferAttribute(pos, 3));
    geo.setAttribute("color", new THREE.BufferAttribute(this.colors, 3));
    geo.setAttribute("size", new THREE.BufferAttribute(size, 1));
    geo.setAttribute("alpha", new THREE.BufferAttribute(alpha, 1));
    this.points = new THREE.Points(geo, this.material);

    // Short links from a random sample of points to their nearest neighbour (~2ms once).
    const linkPos = new Float32Array(LINKS * 6);
    for (let k = 0; k < LINKS; k++) {
      const i = Math.floor(Math.random() * CROWD);
      let best = -1;
      let bestD = Infinity;
      for (let j = 0; j < CROWD; j++) {
        if (j === i) continue;
        const dx = pos[i * 3] - pos[j * 3];
        const dy = pos[i * 3 + 1] - pos[j * 3 + 1];
        const dz = pos[i * 3 + 2] - pos[j * 3 + 2];
        const d = dx * dx + dy * dy + dz * dz;
        if (d < bestD) {
          bestD = d;
          best = j;
        }
      }
      linkPos.set(pos.subarray(i * 3, i * 3 + 3), k * 6);
      linkPos.set(pos.subarray(best * 3, best * 3 + 3), k * 6 + 3);
    }
    const linkGeo = new THREE.BufferGeometry();
    linkGeo.setAttribute("position", new THREE.BufferAttribute(linkPos, 3));
    this.links = new THREE.LineSegments(linkGeo, new THREE.LineBasicMaterial({ transparent: true, opacity: 0.12 }));

    this.group.add(this.points, this.links);
    this.group.rotation.set(0.35, 0, 0.15);
    this.scene.add(this.group);

    this.setTheme(theme);
    this.colors.set(this.target);
    this.resize();
    window.addEventListener("resize", this.resize);
    window.addEventListener("pointermove", this.onPointer, { passive: true });
    document.addEventListener("visibilitychange", this.onVisibility);
    this.start();
  }

  setTheme(theme: BackdropTheme) {
    this.theme = theme;
    this.base.set(theme.dark ? "#f7f7ee" : "#292919");
    (this.links.material as THREE.LineBasicMaterial).color.set(theme.dark ? "#f7f7ee" : "#292919");
    (this.links.material as THREE.LineBasicMaterial).opacity = theme.dark ? 0.1 : 0.14;
    this.paint();
    if (this.still) this.render();
  }

  setMood(mood: BackdropMood, mix: ReactionMix | null) {
    this.mood = mood;
    this.mix = mood === "done" ? mix : null;
    this.paint();
    if (this.still) {
      this.colors.set(this.target);
      (this.points.geometry.attributes.color as THREE.BufferAttribute).needsUpdate = true;
      this.render();
    }
  }

  // Fill the target colours: neutral crowd, or the reaction mix laid over it.
  private paint() {
    const c = new THREE.Color();
    const { share = 0, comment = 0, like = 0 } = this.mix ?? {};
    const cut = [share, share + comment, share + comment + like].map((f) => Math.round(f * CROWD));
    this.order.forEach((idx, rank) => {
      if (rank < cut[0]) c.set(this.theme.colors.share);
      else if (rank < cut[1]) c.set(this.theme.colors.comment);
      else if (rank < cut[2]) c.set(this.theme.colors.like);
      else c.copy(this.base);
      c.toArray(this.target, idx * 3);
    });
  }

  private onPointer = (e: PointerEvent) => {
    this.pointer.set((e.clientX / window.innerWidth) * 2 - 1, (e.clientY / window.innerHeight) * 2 - 1);
  };

  private onVisibility = () => {
    if (document.hidden) cancelAnimationFrame(this.frame);
    else this.start();
  };

  private resize = () => {
    const w = window.innerWidth;
    const h = window.innerHeight;
    this.renderer.setSize(w, h, false);
    this.camera.aspect = w / h;
    // Narrow screens: pull back so the sphere still fits, and sit it lower.
    this.camera.position.z = w < 640 ? 9.5 : 7.2;
    this.camera.updateProjectionMatrix();
    if (this.still) this.render();
  };

  private start() {
    cancelAnimationFrame(this.frame);
    this.clock.getDelta();
    if (this.still) {
      this.render();
      return;
    }
    const tick = () => {
      this.frame = requestAnimationFrame(tick);
      this.step(Math.min(this.clock.getDelta(), 0.05));
      this.render();
    };
    tick();
  }

  private step(dt: number) {
    const running = this.mood === "running";
    this.spin += ((running ? 0.55 : 0.05) - this.spin) * Math.min(1, dt * 2.5);
    this.group.rotation.y += this.spin * dt;
    this.material.uniforms.uPulse.value += ((running ? 1 : 0) - this.material.uniforms.uPulse.value) * dt * 2;

    // Ease toward the pointer for a gentle parallax.
    this.group.rotation.x += (0.35 + this.pointer.y * 0.12 - this.group.rotation.x) * dt * 1.5;
    this.camera.position.x += (this.pointer.x * 0.35 - this.camera.position.x) * dt * 1.5;
    this.camera.lookAt(0, 0, 0);

    // Scroll lifts the sphere away so it frames the hero, then recedes behind the app.
    const scroll = Math.min(window.scrollY / window.innerHeight, 1.5);
    this.group.position.y += (scroll * 1.2 - this.group.position.y) * dt * 4;

    const col = this.points.geometry.attributes.color as THREE.BufferAttribute;
    let moving = false;
    const k = Math.min(1, dt * 3);
    for (let i = 0; i < this.colors.length; i++) {
      const d = this.target[i] - this.colors[i];
      if (Math.abs(d) > 0.002) {
        this.colors[i] += d * k;
        moving = true;
      }
    }
    if (moving) col.needsUpdate = true;
  }

  private render() {
    this.renderer.render(this.scene, this.camera);
  }

  dispose() {
    cancelAnimationFrame(this.frame);
    window.removeEventListener("resize", this.resize);
    window.removeEventListener("pointermove", this.onPointer);
    document.removeEventListener("visibilitychange", this.onVisibility);
    this.points.geometry.dispose();
    this.links.geometry.dispose();
    this.material.dispose();
    (this.links.material as THREE.Material).dispose();
    this.renderer.dispose();
  }
}
