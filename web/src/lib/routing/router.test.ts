import { describe, it, expect } from 'vitest';
import { buildGraph, route, nearestNode, polylineKm, haversineKm, type OverpassElement } from './router';

// A small 4-node grid:  1 — 2
//                       |   |
//                       3 — 4
const grid: OverpassElement[] = [
	{ type: 'node', id: 1, lat: 0, lon: 0 },
	{ type: 'node', id: 2, lat: 0, lon: 0.01 },
	{ type: 'node', id: 3, lat: -0.01, lon: 0 },
	{ type: 'node', id: 4, lat: -0.01, lon: 0.01 },
	{ type: 'way', id: 10, nodes: [1, 2], tags: { highway: 'residential' } },
	{ type: 'way', id: 11, nodes: [1, 3], tags: { highway: 'residential' } },
	{ type: 'way', id: 12, nodes: [3, 4], tags: { highway: 'residential' } },
	{ type: 'way', id: 13, nodes: [2, 4], tags: { highway: 'residential' } }
];

describe('router', () => {
	it('builds a bidirectional graph and links neighbours', () => {
		const g = buildGraph(grid);
		expect(g.coords.size).toBe(4);
		// node 1 connects to 2 and 3 (both directions present)
		expect(g.adj.get(1)?.map((e) => e.to).sort()).toEqual([2, 3]);
		expect(g.adj.get(4)?.map((e) => e.to).sort()).toEqual([2, 3]);
	});

	it('honours oneway=yes (forward only)', () => {
		const g = buildGraph([
			{ type: 'node', id: 1, lat: 0, lon: 0 },
			{ type: 'node', id: 2, lat: 0, lon: 0.01 },
			{ type: 'way', id: 9, nodes: [1, 2], tags: { highway: 'residential', oneway: 'yes' } }
		]);
		expect(g.adj.get(1)?.map((e) => e.to)).toEqual([2]);
		expect(g.adj.get(2) ?? []).toEqual([]);
	});

	it('honours oneway=-1 (reverse only)', () => {
		const g = buildGraph([
			{ type: 'node', id: 1, lat: 0, lon: 0 },
			{ type: 'node', id: 2, lat: 0, lon: 0.01 },
			{ type: 'way', id: 9, nodes: [1, 2], tags: { highway: 'residential', oneway: '-1' } }
		]);
		expect(g.adj.get(2)?.map((e) => e.to)).toEqual([1]);
		expect(g.adj.get(1) ?? []).toEqual([]);
	});

	it('ignores non-highway ways', () => {
		const g = buildGraph([
			{ type: 'node', id: 1, lat: 0, lon: 0 },
			{ type: 'node', id: 2, lat: 0, lon: 0.01 },
			{ type: 'way', id: 9, nodes: [1, 2], tags: { waterway: 'river' } }
		]);
		expect(g.adj.size).toBe(0);
	});

	it('finds the nearest node', () => {
		const g = buildGraph(grid);
		expect(nearestNode(g, { lat: -0.009, lon: 0.0001 })).toBe(3);
	});

	it('routes start → goal and brackets with the exact endpoints', () => {
		const g = buildGraph(grid);
		const start = { lat: 0.0001, lon: -0.0001 }; // near node 1
		const goal = { lat: -0.0101, lon: 0.0101 }; // near node 4
		const line = route(g, start, goal)!;
		expect(line).not.toBeNull();
		expect(line[0]).toEqual(start);
		expect(line[line.length - 1]).toEqual(goal);
		// start + 3 graph nodes (1→2→4 or 1→3→4) + goal
		expect(line.length).toBe(5);
	});

	it('snaps past a disconnected stub into the main component', () => {
		// A tiny isolated 2-node service stub sits right next to the start, but
		// the route must snap to the connected grid instead of the dead stub.
		const withStub: OverpassElement[] = [
			...grid,
			{ type: 'node', id: 98, lat: 0.00004, lon: -0.00004 },
			{ type: 'node', id: 99, lat: 0.00006, lon: -0.00006 },
			{ type: 'way', id: 50, nodes: [98, 99], tags: { highway: 'service' } }
		];
		const g = buildGraph(withStub);
		const line = route(g, { lat: 0, lon: 0 }, { lat: -0.0101, lon: 0.0101 });
		expect(line).not.toBeNull();
		expect(line!.length).toBeGreaterThanOrEqual(4);
	});

	it('returns null when the graph is empty', () => {
		expect(route(buildGraph([]), { lat: 0, lon: 0 }, { lat: 1, lon: 1 })).toBeNull();
	});

	it('polylineKm sums segment lengths', () => {
		const a = { lat: 0, lon: 0 };
		const b = { lat: 0, lon: 0.01 };
		expect(polylineKm([a, b])).toBeCloseTo(haversineKm(a, b), 6);
	});
});
