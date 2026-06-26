import { describe, it, expect } from 'vitest';
import {
	bearing,
	signedTurn,
	classifyTurn,
	cumulativeKm,
	buildManeuvers,
	snapToRoute,
	nextManeuver,
	instructionText,
	formatDistance
} from './navigation';
import type { RoutedPath } from './router';

describe('navigation geometry', () => {
	it('bearing points east and north correctly', () => {
		expect(bearing({ lat: 0, lon: 0 }, { lat: 0, lon: 1 })).toBeCloseTo(90, 0);
		expect(bearing({ lat: 0, lon: 0 }, { lat: 1, lon: 0 })).toBeCloseTo(0, 0);
	});

	it('signedTurn is positive for right, negative for left', () => {
		expect(signedTurn(0, 90)).toBeCloseTo(90); // east of north = right
		expect(signedTurn(0, 270)).toBeCloseTo(-90); // west = left
	});

	it('classifies turns', () => {
		expect(classifyTurn(5)).toBe('straight');
		expect(classifyTurn(90)).toBe('right');
		expect(classifyTurn(-90)).toBe('left');
		expect(classifyTurn(30)).toBe('slight-right');
		expect(classifyTurn(175)).toBe('uturn');
	});
});

// An L-shaped path: go east, then turn (to the) right and head south.
const lShape: RoutedPath = {
	points: [
		{ lat: 0.02, lon: 0 },
		{ lat: 0.02, lon: 0.02 },
		{ lat: 0.0, lon: 0.02 }
	],
	names: [undefined, 'East Street', 'South Street']
};

describe('maneuvers', () => {
	it('emits depart, the turn with the street name, and arrive', () => {
		const m = buildManeuvers(lShape);
		expect(m[0].type).toBe('depart');
		expect(m[m.length - 1].type).toBe('arrive');
		const turn = m.find((x) => x.type === 'right' || x.type === 'left');
		expect(turn).toBeTruthy();
		expect(turn!.type).toBe('right');
		expect(turn!.name).toBe('South Street');
	});

	it('instructionText includes the street name', () => {
		const m = buildManeuvers(lShape).find((x) => x.type === 'right')!;
		expect(instructionText(m)).toBe('Turn right onto South Street');
	});
});

describe('snapping + progress', () => {
	const pts = lShape.points;
	const cum = cumulativeKm(pts);

	it('snaps a point near the first leg', () => {
		const s = snapToRoute(pts, cum, { lat: 0.0201, lon: 0.01 });
		expect(s.segIndex).toBe(0);
		expect(s.distToRouteKm).toBeLessThan(0.02);
		expect(s.distAlongKm).toBeGreaterThan(0);
	});

	it('nextManeuver returns the upcoming turn ahead of position', () => {
		const s = snapToRoute(pts, cum, { lat: 0.02, lon: 0.005 });
		const nm = nextManeuver(buildManeuvers(lShape), s.distAlongKm);
		expect(nm).toBeTruthy();
		expect(['right', 'arrive']).toContain(nm!.maneuver.type);
		expect(nm!.distKm).toBeGreaterThan(0);
	});
});

describe('formatting', () => {
	it('formats metres and km', () => {
		expect(formatDistance(0.12)).toBe('120 m');
		expect(formatDistance(2.34)).toBe('2.3 km');
	});
});
