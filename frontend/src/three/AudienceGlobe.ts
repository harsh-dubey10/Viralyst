import * as THREE from "three";
import { Action, Edge, Persona } from "../types";
import { fibonacciSphere, pixelRatio, prefersReducedMotion } from "./util";

export type ActionColors = Record<Action, string>;

export interface GlobeTheme {
  dark: boolean;
  colors: ActionColors;
}

export interface GlobeCallbacks {
  onSelect: (index: number | null) => void;
  onHover: (index: number | null, x: number, y: number) => void;
}

const R = 2;
const ARC_SEGMENTS = 28;
const GROW_SECONDS = 0.8;

interface Arc {
  line: THREE.Line;
  round: number;
  shownAt: number; // clock time the arc started growing, or -1 while hidden
}

// The simulated audience as a draggable globe: one node per viewer (your audience on
// one hemisphere, everyone else on the other), coloured by what they did, with arcs
// for shares and algorithmic recommendations that grow in round by round.
export class AudienceGlobe {
  private renderer: THREE.WebGLRenderer;
  private scene = new THREE.Scene();
  private camera = new THREE.PerspectiveCamera(38, 1, 0.1, 50);
  private group = new THREE.Group();
  private nodes: THREE.InstancedMesh | null = null;
  private nodeGeo = new THREE.SphereGeometry(0.085, 16, 12);
  private nodeMat = new THREE.MeshLambertMaterial();
  private arcs: Arc[] = [];
  private shell: THREE.LineSegments;
  private divider: THREE.Line;
  private personas: Persona[] = [];
  private positions: THREE.Vector3[] = [];
  private round = 0;
  private selected: number | null = null;
  private hovered: number | null = null;
  private theme: GlobeTheme;
  private clock = new THREE.Clock();
  private frame = 0;
  private visible = true;
  private still = prefersReducedMotion();
  private raycaster = new THREE.Raycaster();
  private ndc = new THREE.Vector2();
  private drag = { active: false, moved: false, x: 0, y: 0, vx: 0, vy: 0, lastInput: -10 };
  private io: IntersectionObserver | null = null;
  private ro: ResizeObserver | null = null;
  private dummy = new THREE.Object3D();
  private tmpColor = new THREE.Color();

  constructor(private canvas: HTMLCanvasElement, private host: HTMLElement, theme: GlobeTheme, private cb: GlobeCallbacks) {
    this.theme = theme;
    this.renderer = new THREE.WebGLRenderer({ canvas, alpha: true, antialias: true });
    this.renderer.setPixelRatio(pixelRatio());
    this.camera.position.set(0, 0.4, 7);
    this.camera.lookAt(0, 0, 0);

    this.scene.add(new THREE.AmbientLight(0xffffff, 1.6));
    const key = new THREE.DirectionalLight(0xffffff, 1.4);
    key.position.set(3, 4, 5);
    this.scene.add(key);

    this.shell = new THREE.LineSegments(
      new THREE.WireframeGeometry(new THREE.SphereGeometry(R * 0.985, 18, 10)),
      new THREE.LineBasicMaterial({ transparent: true, opacity: 0.06 })
    );
    // The meridian splitting your audience (x < 0) from everyone else (x > 0).
    const ring = new THREE.EllipseCurve(0, 0, R * 1.03, R * 1.03).getPoints(96).map((p) => new THREE.Vector3(0, p.x, p.y));
    this.divider = new THREE.Line(new THREE.BufferGeometry().setFromPoints(ring), new THREE.LineBasicMaterial({ transparent: true, opacity: 0.3 }));
    this.group.add(this.shell, this.divider);
    this.group.rotation.set(0.25, -0.5, 0);
    this.scene.add(this.group);

    this.setTheme(theme);
    canvas.addEventListener("pointerdown", this.onDown);
    canvas.addEventListener("pointermove", this.onMove);
    canvas.addEventListener("pointerleave", this.onLeave);
    window.addEventListener("pointerup", this.onUp);

    this.ro = new ResizeObserver(() => this.resize());
    this.ro.observe(host);
    this.io = new IntersectionObserver(([e]) => {
      this.visible = e.isIntersecting;
      if (this.visible) this.start();
      else cancelAnimationFrame(this.frame);
    });
    this.io.observe(host);
    this.resize();
    this.start();
  }

  // Placeholder (no result yet): n grey nodes, so the globe previews the audience size.
  setPlaceholder(n: number) {
    this.setData(
      Array.from({ length: n }, (_, i) => ({ id: i + 1, inTarget: i % 2 === 0, shown: false, action: "never", round: 0 } as Persona)),
      []
    );
  }

  setData(personas: Persona[], edges: Edge[]) {
    this.personas = personas;
    this.selected = this.hovered = null;
    this.layout();

    if (this.nodes) {
      this.group.remove(this.nodes);
      this.nodes.dispose();
    }
    this.nodes = new THREE.InstancedMesh(this.nodeGeo, this.nodeMat, Math.max(personas.length, 1));
    this.nodes.count = personas.length;
    this.group.add(this.nodes);

    this.arcs.forEach((a) => {
      this.group.remove(a.line);
      a.line.geometry.dispose();
      (a.line.material as THREE.Material).dispose();
    });
    this.arcs = edges.flatMap((e) => {
      const a = this.positions[e.from];
      const b = this.positions[e.to];
      if (!a || !b) return [];
      const lift = R * 1.12 + a.distanceTo(b) * 0.28;
      const mid = a.clone().add(b).multiplyScalar(0.5).normalize().multiplyScalar(lift);
      const geo = new THREE.BufferGeometry().setFromPoints(new THREE.QuadraticBezierCurve3(a, mid, b).getPoints(ARC_SEGMENTS));
      const mat =
        e.type === "direct"
          ? new THREE.LineBasicMaterial({ transparent: true, opacity: 0.9 })
          : new THREE.LineDashedMaterial({ transparent: true, opacity: 0.45, dashSize: 0.06, gapSize: 0.05 });
      const line = new THREE.Line(geo, mat);
      if (e.type !== "direct") line.computeLineDistances();
      line.userData.type = e.type;
      line.visible = false;
      this.group.add(line);
      return [{ line, round: e.round, shownAt: -1 }];
    });
    this.paintArcs();
    this.setRound(this.round);
  }

  // In-target viewers take the lattice points on the left hemisphere, the rest the right.
  private layout() {
    const n = this.personas.length;
    const lattice = fibonacciSphere(n, R).sort((p, q) => p.x - q.x);
    const inT = this.personas.filter((p) => p.inTarget).length;
    let left = 0;
    let right = inT;
    this.positions = this.personas.map((p) => lattice[p.inTarget ? left++ : right++]);
  }

  setRound(round: number) {
    const now = this.clock.elapsedTime;
    if (round < this.round) this.arcs.forEach((a) => (a.shownAt = -1));
    this.round = round;
    this.arcs.forEach((a) => {
      if (a.round <= round && a.shownAt < 0) a.shownAt = this.still ? now - GROW_SECONDS : now;
      if (a.round > round) a.shownAt = -1;
    });
    this.paintNodes();
    this.kick();
  }

  setSelected(index: number | null) {
    this.selected = index;
    this.paintNodes();
    this.kick();
  }

  setTheme(theme: GlobeTheme) {
    this.theme = theme;
    const ink = theme.dark ? "#f7f7ee" : "#292919";
    (this.shell.material as THREE.LineBasicMaterial).color.set(ink);
    (this.shell.material as THREE.LineBasicMaterial).opacity = theme.dark ? 0.07 : 0.1;
    (this.divider.material as THREE.LineBasicMaterial).color.set(ink);
    this.paintNodes();
    this.paintArcs();
    this.kick();
  }

  private paintArcs() {
    this.arcs.forEach((a) => {
      const m = a.line.material as THREE.LineBasicMaterial;
      m.color.set(a.line.userData.type === "direct" ? this.theme.colors.share : this.theme.dark ? "#909083" : "#5c5c4e");
    });
  }

  private visibleAction(p: Persona): Action {
    return p.shown && p.round > 0 && p.round <= this.round ? p.action : "never";
  }

  private paintNodes() {
    const nodes = this.nodes;
    if (!nodes) return;
    this.personas.forEach((p, i) => {
      const action = this.visibleAction(p);
      let s = action === "share" ? 1.55 : action === "never" ? 0.75 : 1;
      if (i === this.hovered) s *= 1.5;
      if (i === this.selected) s *= 1.5;
      this.dummy.position.copy(this.positions[i]);
      this.dummy.scale.setScalar(s);
      this.dummy.updateMatrix();
      nodes.setMatrixAt(i, this.dummy.matrix);
      this.tmpColor.set(i === this.selected ? (this.theme.dark ? "#ffffff" : "#000000") : this.theme.colors[action]);
      nodes.setColorAt(i, this.tmpColor);
    });
    nodes.instanceMatrix.needsUpdate = true;
    if (nodes.instanceColor) nodes.instanceColor.needsUpdate = true;
  }

  private pick(e: PointerEvent): number | null {
    if (!this.nodes) return null;
    const rect = this.canvas.getBoundingClientRect();
    this.ndc.set(((e.clientX - rect.left) / rect.width) * 2 - 1, -((e.clientY - rect.top) / rect.height) * 2 + 1);
    this.raycaster.setFromCamera(this.ndc, this.camera);
    // Hits are sorted nearest first, so a front node wins over one seen through the globe.
    const hit = this.raycaster.intersectObject(this.nodes, false)[0];
    return hit?.instanceId ?? null;
  }

  private onDown = (e: PointerEvent) => {
    this.drag = { ...this.drag, active: true, moved: false, x: e.clientX, y: e.clientY, vx: 0, vy: 0 };
  };

  private onMove = (e: PointerEvent) => {
    if (this.drag.active) {
      const dx = e.clientX - this.drag.x;
      const dy = e.clientY - this.drag.y;
      if (Math.abs(dx) + Math.abs(dy) > 3) this.drag.moved = true;
      this.drag.vx = dx * 0.006;
      this.drag.vy = dy * 0.006;
      this.group.rotation.y += this.drag.vx;
      this.group.rotation.x = THREE.MathUtils.clamp(this.group.rotation.x + this.drag.vy, -1.1, 1.1);
      this.drag.x = e.clientX;
      this.drag.y = e.clientY;
      this.drag.lastInput = this.clock.elapsedTime;
      this.kick();
      return;
    }
    const idx = this.pick(e);
    if (idx !== this.hovered) {
      this.hovered = idx;
      this.canvas.style.cursor = idx !== null ? "pointer" : "grab";
      this.paintNodes();
      this.kick();
    }
    const rect = this.canvas.getBoundingClientRect();
    this.cb.onHover(idx, e.clientX - rect.left, e.clientY - rect.top);
  };

  private onUp = (e: PointerEvent) => {
    if (!this.drag.active) return;
    this.drag.active = false;
    if (!this.drag.moved && e.target === this.canvas) this.cb.onSelect(this.pick(e));
  };

  private onLeave = () => {
    if (this.hovered !== null) {
      this.hovered = null;
      this.paintNodes();
      this.kick();
    }
    this.cb.onHover(null, 0, 0);
  };

  private resize() {
    const w = this.host.clientWidth;
    const h = this.host.clientHeight;
    if (!w || !h) return;
    this.renderer.setSize(w, h, false);
    this.camera.aspect = w / h;
    this.camera.position.z = w < 520 ? 8.6 : 7;
    this.camera.updateProjectionMatrix();
    this.kick();
  }

  // With reduced motion there is no loop: render once after each change.
  private kick() {
    if (!this.still) return;
    this.updateArcs(this.clock.elapsedTime);
    this.renderer.render(this.scene, this.camera);
  }

  // Grow each revealed arc from its source over GROW_SECONDS (ease-out).
  private updateArcs(t: number) {
    this.arcs.forEach((a) => {
      a.line.visible = a.shownAt >= 0;
      if (a.line.visible) {
        const p = Math.min(1, (t - a.shownAt) / GROW_SECONDS);
        const eased = 1 - Math.pow(1 - p, 3);
        a.line.geometry.setDrawRange(0, Math.max(2, Math.round(eased * (ARC_SEGMENTS + 1))));
      }
    });
  }

  private start() {
    cancelAnimationFrame(this.frame);
    if (this.still) {
      this.kick();
      return;
    }
    const tick = () => {
      this.frame = requestAnimationFrame(tick);
      const dt = Math.min(this.clock.getDelta(), 0.05);
      const t = this.clock.elapsedTime;
      if (!this.drag.active) {
        // Inertia after a drag, then a slow idle turn once the user lets go for a bit.
        this.group.rotation.y += this.drag.vx;
        this.group.rotation.x = THREE.MathUtils.clamp(this.group.rotation.x + this.drag.vy, -1.1, 1.1);
        this.drag.vx *= 0.92;
        this.drag.vy *= 0.92;
        if (t - this.drag.lastInput > 3) this.group.rotation.y += dt * 0.12;
      }
      this.updateArcs(t);
      this.renderer.render(this.scene, this.camera);
    };
    this.clock.getDelta();
    if (this.visible) tick();
  }

  dispose() {
    cancelAnimationFrame(this.frame);
    this.io?.disconnect();
    this.ro?.disconnect();
    this.canvas.removeEventListener("pointerdown", this.onDown);
    this.canvas.removeEventListener("pointermove", this.onMove);
    this.canvas.removeEventListener("pointerleave", this.onLeave);
    window.removeEventListener("pointerup", this.onUp);
    this.arcs.forEach((a) => {
      a.line.geometry.dispose();
      (a.line.material as THREE.Material).dispose();
    });
    this.nodes?.dispose();
    this.nodeGeo.dispose();
    this.nodeMat.dispose();
    this.shell.geometry.dispose();
    (this.shell.material as THREE.Material).dispose();
    this.divider.geometry.dispose();
    (this.divider.material as THREE.Material).dispose();
    this.renderer.dispose();
  }
}
