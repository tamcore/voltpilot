// Minimal client-side road router: builds a graph from OSM (Overpass) road
// ways and runs A* over it. Intended for short (<5 km) previews only — no turn
// restrictions, no traffic, distance-weighted shortest path.

export type LatLng = { lat: number; lon: number };

export type OverpassElement = {
	type: 'node' | 'way' | 'relation';
	id: number;
	lat?: number;
	lon?: number;
	nodes?: number[];
	tags?: Record<string, string>;
};

export type Edge = { to: number; dist: number; name?: string };
export type Graph = {
	coords: Map<number, LatLng>;
	adj: Map<number, Edge[]>;
};

const EARTH_KM = 6371.0088;

export function haversineKm(a: LatLng, b: LatLng): number {
	const toRad = (d: number) => (d * Math.PI) / 180;
	const dPhi = toRad(b.lat - a.lat);
	const dLam = toRad(b.lon - a.lon);
	const phi1 = toRad(a.lat);
	const phi2 = toRad(b.lat);
	const h = Math.sin(dPhi / 2) ** 2 + Math.cos(phi1) * Math.cos(phi2) * Math.sin(dLam / 2) ** 2;
	return 2 * EARTH_KM * Math.asin(Math.sqrt(h));
}

function oneWayDirections(tags: Record<string, string> | undefined): {
	forward: boolean;
	backward: boolean;
} {
	const ow = (tags?.oneway ?? '').toLowerCase();
	if (ow === '-1' || ow === 'reverse') return { forward: false, backward: true };
	if (ow === 'yes' || ow === 'true' || ow === '1') return { forward: true, backward: false };
	// Roundabouts are implicitly one-way (forward in node order).
	if ((tags?.junction ?? '').toLowerCase() === 'roundabout') return { forward: true, backward: false };
	return { forward: true, backward: true };
}

// buildGraph turns Overpass elements (nodes + highway ways) into a routable graph.
export function buildGraph(elements: OverpassElement[]): Graph {
	const coords = new Map<number, LatLng>();
	for (const el of elements) {
		if (el.type === 'node' && el.lat !== undefined && el.lon !== undefined) {
			coords.set(el.id, { lat: el.lat, lon: el.lon });
		}
	}

	const adj = new Map<number, Edge[]>();
	const link = (a: number, b: number, name?: string) => {
		const ca = coords.get(a);
		const cb = coords.get(b);
		if (!ca || !cb) return;
		const d = haversineKm(ca, cb);
		if (!adj.has(a)) adj.set(a, []);
		adj.get(a)!.push({ to: b, dist: d, name });
	};

	for (const el of elements) {
		if (el.type !== 'way' || !el.nodes || !el.tags?.highway) continue;
		const { forward, backward } = oneWayDirections(el.tags);
		const name = el.tags.name || el.tags.ref || undefined;
		for (let i = 0; i + 1 < el.nodes.length; i++) {
			const a = el.nodes[i];
			const b = el.nodes[i + 1];
			if (forward) link(a, b, name);
			if (backward) link(b, a, name);
		}
	}
	return { coords, adj };
}

export function nearestNode(graph: Graph, p: LatLng, allowed?: Set<number>): number | null {
	let best: number | null = null;
	let bestD = Infinity;
	for (const [id, c] of graph.coords) {
		if (allowed && !allowed.has(id)) continue;
		const d = haversineKm(p, c);
		if (d < bestD) {
			bestD = d;
			best = id;
		}
	}
	return best;
}

// mainComponent returns the node ids of the largest weakly-connected component.
// Snapping endpoints into it avoids landing on a disconnected stub (e.g. a
// pedestrianised square or an orphan service spur), which would yield no route.
export function mainComponent(graph: Graph): Set<number> {
	const undirected = new Map<number, number[]>();
	const add = (a: number, b: number) => {
		if (!undirected.has(a)) undirected.set(a, []);
		undirected.get(a)!.push(b);
	};
	for (const [u, edges] of graph.adj) {
		for (const e of edges) {
			add(u, e.to);
			add(e.to, u);
		}
	}

	const seen = new Set<number>();
	let best = new Set<number>();
	for (const start of undirected.keys()) {
		if (seen.has(start)) continue;
		const comp = new Set<number>([start]);
		const stack = [start];
		seen.add(start);
		while (stack.length) {
			const u = stack.pop()!;
			for (const v of undirected.get(u) ?? []) {
				if (!seen.has(v)) {
					seen.add(v);
					comp.add(v);
					stack.push(v);
				}
			}
		}
		if (comp.size > best.size) best = comp;
	}
	return best;
}

// Tiny binary min-heap keyed by f-score, so A* stays fast on dense urban graphs.
class MinHeap {
	private ids: number[] = [];
	private keys: number[] = [];

	get size(): number {
		return this.ids.length;
	}
	push(id: number, key: number) {
		this.ids.push(id);
		this.keys.push(key);
		this.up(this.ids.length - 1);
	}
	pop(): number | undefined {
		if (this.ids.length === 0) return undefined;
		const topId = this.ids[0];
		const lastId = this.ids.pop()!;
		const lastKey = this.keys.pop()!;
		if (this.ids.length > 0) {
			this.ids[0] = lastId;
			this.keys[0] = lastKey;
			this.down(0);
		}
		return topId;
	}
	private up(i: number) {
		while (i > 0) {
			const parent = (i - 1) >> 1;
			if (this.keys[parent] <= this.keys[i]) break;
			this.swap(i, parent);
			i = parent;
		}
	}
	private down(i: number) {
		const n = this.ids.length;
		for (;;) {
			let smallest = i;
			const l = 2 * i + 1;
			const r = 2 * i + 2;
			if (l < n && this.keys[l] < this.keys[smallest]) smallest = l;
			if (r < n && this.keys[r] < this.keys[smallest]) smallest = r;
			if (smallest === i) break;
			this.swap(i, smallest);
			i = smallest;
		}
	}
	private swap(a: number, b: number) {
		[this.ids[a], this.ids[b]] = [this.ids[b], this.ids[a]];
		[this.keys[a], this.keys[b]] = [this.keys[b], this.keys[a]];
	}
}

// route returns a polyline (start → road network → goal) or null when no path
// exists. The user/charger points are prepended/appended so the line connects
// visually even though they sit off the graph nodes.
// aStar returns the sequence of graph node ids from start to goal (snapped into
// the main component), or null when unreachable.
function aStar(graph: Graph, start: LatLng, goal: LatLng): number[] | null {
	// Snap endpoints into the main connected component so we don't anchor on an
	// isolated stub (which would make A* fail even though a route exists).
	const comp = mainComponent(graph);
	const s = nearestNode(graph, start, comp.size > 0 ? comp : undefined);
	const g = nearestNode(graph, goal, comp.size > 0 ? comp : undefined);
	if (s === null || g === null) return null;

	const goalC = graph.coords.get(g)!;
	const gScore = new Map<number, number>([[s, 0]]);
	const cameFrom = new Map<number, number>();
	const open = new MinHeap();
	open.push(s, haversineKm(graph.coords.get(s)!, goalC));
	const closed = new Set<number>();

	while (open.size > 0) {
		const current = open.pop()!;
		if (current === g) {
			const nodes = [g];
			let cur = g;
			while (cameFrom.has(cur)) {
				cur = cameFrom.get(cur)!;
				nodes.push(cur);
			}
			nodes.reverse();
			return nodes;
		}
		if (closed.has(current)) continue;
		closed.add(current);

		const base = gScore.get(current)!;
		for (const e of graph.adj.get(current) ?? []) {
			const tentative = base + e.dist;
			if (tentative < (gScore.get(e.to) ?? Infinity)) {
				gScore.set(e.to, tentative);
				cameFrom.set(e.to, current);
				open.push(e.to, tentative + haversineKm(graph.coords.get(e.to)!, goalC));
			}
		}
	}
	return null;
}

// route returns a polyline (start → road network → goal) or null when no path
// exists. The user/charger points are prepended/appended so the line connects
// visually even though they sit off the graph nodes.
export function route(graph: Graph, start: LatLng, goal: LatLng): LatLng[] | null {
	const nodes = aStar(graph, start, goal);
	if (!nodes) return null;
	return [start, ...nodes.map((id) => graph.coords.get(id)!), goal];
}

// RoutedPath carries the geometry plus the street name of the segment that
// leads into each point (names[i] = name of segment points[i-1] → points[i]).
export type RoutedPath = { points: LatLng[]; names: (string | undefined)[] };

function edgeName(graph: Graph, a: number, b: number): string | undefined {
	for (const e of graph.adj.get(a) ?? []) if (e.to === b) return e.name;
	return undefined;
}

// routeWithNames is like route() but also returns the street name per segment,
// used to build turn-by-turn maneuvers.
export function routeWithNames(graph: Graph, start: LatLng, goal: LatLng): RoutedPath | null {
	const nodes = aStar(graph, start, goal);
	if (!nodes) return null;
	const points: LatLng[] = [start, ...nodes.map((id) => graph.coords.get(id)!), goal];
	const names: (string | undefined)[] = [undefined]; // start → first node: unnamed
	for (let i = 0; i + 1 < nodes.length; i++) names.push(edgeName(graph, nodes[i], nodes[i + 1]));
	names.push(undefined); // last node → goal: unnamed
	// names has points.length entries (index 0 unused as a "segment into start").
	return { points, names };
}

export function polylineKm(pts: LatLng[]): number {
	let sum = 0;
	for (let i = 0; i + 1 < pts.length; i++) sum += haversineKm(pts[i], pts[i + 1]);
	return sum;
}
