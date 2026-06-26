import { writable } from 'svelte/store';

export type GeoState =
	| { status: 'idle' }
	| { status: 'unavailable' }
	| { status: 'permission-denied' }
	| { status: 'pending' }
	| {
			status: 'live';
			lat: number;
			lon: number;
			accuracy: number;
			timestamp: number;
	  };

function createGeoStore() {
	const inner = writable<GeoState>({ status: 'idle' });
	let watchId: number | null = null;

	function start() {
		if (typeof navigator === 'undefined' || !('geolocation' in navigator)) {
			inner.set({ status: 'unavailable' });
			return;
		}
		if (watchId !== null) return;
		inner.set({ status: 'pending' });
		watchId = navigator.geolocation.watchPosition(
			(pos) => {
				inner.set({
					status: 'live',
					lat: pos.coords.latitude,
					lon: pos.coords.longitude,
					accuracy: pos.coords.accuracy,
					timestamp: pos.timestamp
				});
			},
			(err) => {
				inner.set({
					status: err.code === err.PERMISSION_DENIED ? 'permission-denied' : 'unavailable'
				});
			},
			{ enableHighAccuracy: true, maximumAge: 5000, timeout: 30000 }
		);
	}

	function stop() {
		if (watchId !== null && typeof navigator !== 'undefined' && 'geolocation' in navigator) {
			navigator.geolocation.clearWatch(watchId);
		}
		watchId = null;
	}

	return { subscribe: inner.subscribe, start, stop };
}

export const geo = createGeoStore();

// distanceKm returns the great-circle distance between two points in km.
export function distanceKm(
	a: { lat: number; lon: number },
	b: { lat: number; lon: number }
): number {
	const R = 6371.0088;
	const toRad = (d: number) => (d * Math.PI) / 180;
	const dPhi = toRad(b.lat - a.lat);
	const dLam = toRad(b.lon - a.lon);
	const phi1 = toRad(a.lat);
	const phi2 = toRad(b.lat);
	const h =
		Math.sin(dPhi / 2) ** 2 + Math.cos(phi1) * Math.cos(phi2) * Math.sin(dLam / 2) ** 2;
	return 2 * R * Math.asin(Math.sqrt(h));
}
