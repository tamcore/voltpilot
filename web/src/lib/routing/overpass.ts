import { buildGraph, type Graph, type OverpassElement } from './router';

// Drivable highway classes we route over.
const HIGHWAYS =
	'motorway|trunk|primary|secondary|tertiary|unclassified|residential|living_street|service|road|motorway_link|trunk_link|primary_link|secondary_link|tertiary_link';

// Public Overpass endpoints (CORS-enabled). Tried in order.
const ENDPOINTS = [
	'https://overpass-api.de/api/interpreter',
	'https://overpass.kumi.systems/api/interpreter'
];

export type BBox = { minLat: number; minLon: number; maxLat: number; maxLon: number };

function query(b: BBox): string {
	const bbox = `${b.minLat},${b.minLon},${b.maxLat},${b.maxLon}`;
	return `[out:json][timeout:25];(way["highway"~"^(${HIGHWAYS})$"](${bbox}););out body;>;out skel qt;`;
}

// fetchRoadGraph downloads the road network in the bbox from Overpass and
// builds a routable graph. Throws if all endpoints fail.
export async function fetchRoadGraph(b: BBox, signal?: AbortSignal): Promise<Graph> {
	const body = 'data=' + encodeURIComponent(query(b));
	let lastErr: unknown;
	for (const ep of ENDPOINTS) {
		try {
			const res = await fetch(ep, {
				method: 'POST',
				headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
				body,
				signal
			});
			if (!res.ok) {
				lastErr = new Error(`overpass ${res.status}`);
				continue;
			}
			const json = (await res.json()) as { elements?: OverpassElement[] };
			return buildGraph(json.elements ?? []);
		} catch (err) {
			if (err instanceof DOMException && err.name === 'AbortError') throw err;
			lastErr = err;
		}
	}
	throw lastErr instanceof Error ? lastErr : new Error('overpass: all endpoints failed');
}

// bboxFor returns a bounding box covering both points plus a padding margin
// (km) so the route isn't clipped at the edges.
export function bboxFor(
	a: { lat: number; lon: number },
	b: { lat: number; lon: number },
	padKm = 0.6
): BBox {
	const padLat = padKm / 111.32;
	const midLat = (a.lat + b.lat) / 2;
	const cosLat = Math.max(Math.cos((midLat * Math.PI) / 180), 0.01);
	const padLon = padKm / (111.32 * cosLat);
	return {
		minLat: Math.min(a.lat, b.lat) - padLat,
		minLon: Math.min(a.lon, b.lon) - padLon,
		maxLat: Math.max(a.lat, b.lat) + padLat,
		maxLon: Math.max(a.lon, b.lon) + padLon
	};
}
