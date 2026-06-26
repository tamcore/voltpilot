// Pure turn-by-turn helpers over a routed polyline: maneuver extraction,
// snapping live position to the route, and instruction text. No DOM, no state.

import { haversineKm, type LatLng, type RoutedPath } from './router';

export type TurnType =
	| 'depart'
	| 'straight'
	| 'slight-left'
	| 'slight-right'
	| 'left'
	| 'right'
	| 'sharp-left'
	| 'sharp-right'
	| 'uturn'
	| 'arrive';

export type Maneuver = {
	point: LatLng;
	pointIndex: number;
	type: TurnType;
	name?: string; // street proceeded onto after the maneuver
	distFromStartKm: number;
};

const DEG = Math.PI / 180;

// bearing from a to b in degrees, clockwise from north [0,360).
export function bearing(a: LatLng, b: LatLng): number {
	const phi1 = a.lat * DEG;
	const phi2 = b.lat * DEG;
	const dLon = (b.lon - a.lon) * DEG;
	const y = Math.sin(dLon) * Math.cos(phi2);
	const x = Math.cos(phi1) * Math.sin(phi2) - Math.sin(phi1) * Math.cos(phi2) * Math.cos(dLon);
	return (Math.atan2(y, x) / DEG + 360) % 360;
}

// signedTurn: how much you turn going from bearing `from` to `to`.
// Positive = right (clockwise), negative = left, in (-180, 180].
export function signedTurn(from: number, to: number): number {
	let d = ((to - from + 540) % 360) - 180;
	if (d === -180) d = 180;
	return d;
}

export function classifyTurn(deg: number): TurnType {
	const a = Math.abs(deg);
	if (a < 18) return 'straight';
	const right = deg > 0;
	if (a < 50) return right ? 'slight-right' : 'slight-left';
	if (a < 130) return right ? 'right' : 'left';
	if (a < 160) return right ? 'sharp-right' : 'sharp-left';
	return 'uturn';
}

// cumulativeKm[i] = distance along the polyline from the start to point i.
export function cumulativeKm(points: LatLng[]): number[] {
	const cum = [0];
	for (let i = 1; i < points.length; i++) cum.push(cum[i - 1] + haversineKm(points[i - 1], points[i]));
	return cum;
}

// buildManeuvers turns a routed path into depart → turns → arrive.
export function buildManeuvers(path: RoutedPath): Maneuver[] {
	const { points, names } = path;
	if (points.length < 2) return [];
	const cum = cumulativeKm(points);
	const firstName = names.find((n) => n) ?? undefined;
	const out: Maneuver[] = [
		{ point: points[0], pointIndex: 0, type: 'depart', name: firstName, distFromStartKm: 0 }
	];

	for (let i = 1; i + 1 < points.length; i++) {
		const bIn = bearing(points[i - 1], points[i]);
		const bOut = bearing(points[i], points[i + 1]);
		const type = classifyTurn(signedTurn(bIn, bOut));
		if (type === 'straight') continue;
		out.push({
			point: points[i],
			pointIndex: i,
			type,
			name: names[i + 1] ?? names[i],
			distFromStartKm: cum[i]
		});
	}

	out.push({
		point: points[points.length - 1],
		pointIndex: points.length - 1,
		type: 'arrive',
		distFromStartKm: cum[cum.length - 1]
	});
	return out;
}

export type Snap = {
	segIndex: number;
	snapped: LatLng;
	distAlongKm: number;
	distToRouteKm: number;
};

// Project p onto segment a→b using a local equirectangular frame; returns the
// closest point and the fraction t∈[0,1] along the segment.
function projectOnSegment(p: LatLng, a: LatLng, b: LatLng): { t: number; closest: LatLng } {
	const latRef = (a.lat * DEG + b.lat * DEG) / 2;
	const kmPerDegLat = 111.32;
	const kmPerDegLon = 111.32 * Math.cos(latRef);
	const ax = a.lon * kmPerDegLon;
	const ay = a.lat * kmPerDegLat;
	const bx = b.lon * kmPerDegLon;
	const by = b.lat * kmPerDegLat;
	const px = p.lon * kmPerDegLon;
	const py = p.lat * kmPerDegLat;
	const dx = bx - ax;
	const dy = by - ay;
	const len2 = dx * dx + dy * dy;
	let t = len2 === 0 ? 0 : ((px - ax) * dx + (py - ay) * dy) / len2;
	t = Math.max(0, Math.min(1, t));
	return { t, closest: { lat: a.lat + (b.lat - a.lat) * t, lon: a.lon + (b.lon - a.lon) * t } };
}

// snapToRoute finds the nearest point on the polyline to pos.
export function snapToRoute(points: LatLng[], cum: number[], pos: LatLng): Snap {
	let best: Snap = { segIndex: 0, snapped: points[0], distAlongKm: 0, distToRouteKm: Infinity };
	for (let i = 0; i + 1 < points.length; i++) {
		const { closest } = projectOnSegment(pos, points[i], points[i + 1]);
		const d = haversineKm(pos, closest);
		if (d < best.distToRouteKm) {
			const along = cum[i] + haversineKm(points[i], closest);
			best = { segIndex: i, snapped: closest, distAlongKm: along, distToRouteKm: d };
		}
	}
	return best;
}

// nextManeuver returns the upcoming maneuver (first whose distance-along is
// ahead of the current position) and the distance to it in km.
export function nextManeuver(
	maneuvers: Maneuver[],
	distAlongKm: number
): { maneuver: Maneuver; distKm: number } | null {
	for (const m of maneuvers) {
		if (m.type === 'depart') continue;
		if (m.distFromStartKm >= distAlongKm - 0.005) {
			return { maneuver: m, distKm: Math.max(0, m.distFromStartKm - distAlongKm) };
		}
	}
	return null;
}

const TURN_LABEL: Record<TurnType, string> = {
	depart: 'Head out',
	straight: 'Continue straight',
	'slight-left': 'Slight left',
	'slight-right': 'Slight right',
	left: 'Turn left',
	right: 'Turn right',
	'sharp-left': 'Sharp left',
	'sharp-right': 'Sharp right',
	uturn: 'Make a U-turn',
	arrive: 'Arrive'
};

export function turnLabel(t: TurnType): string {
	return TURN_LABEL[t];
}

// instructionText: short banner text, e.g. "Turn left onto Sonnenstraße".
export function instructionText(m: Maneuver): string {
	if (m.type === 'arrive') return 'Arrive at the charger';
	const base = turnLabel(m.type);
	return m.name ? `${base} onto ${m.name}` : base;
}

export function formatDistance(km: number): string {
	const m = km * 1000;
	if (m < 1000) return `${Math.max(0, Math.round(m / 10) * 10)} m`;
	return `${km.toFixed(1)} km`;
}
