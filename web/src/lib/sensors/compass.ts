import { writable } from 'svelte/store';

// Compass heading from the device magnetometer, exposed as a store of degrees
// (0–360, clockwise from north — the direction the top of the phone points),
// or null when unavailable/disabled. iOS needs a user-gesture permission grant;
// Android uses the absolute-orientation event. Falls back to nothing (the nav
// view then uses GPS-course) when no magnetometer or permission denied.

export function normalizeDeg(d: number): number {
	return ((d % 360) + 360) % 360;
}

// circularLowPass smooths heading along the shortest arc to avoid 359°→0° jumps.
export function circularLowPass(prev: number | null, next: number, alpha = 0.25): number {
	if (prev === null) return normalizeDeg(next);
	const diff = ((next - prev + 540) % 360) - 180;
	return normalizeDeg(prev + alpha * diff);
}

// androidHeadingFromAlpha converts an absolute-orientation alpha (degrees,
// counter-clockwise from north) plus the screen rotation into a compass
// heading (clockwise from north).
export function androidHeadingFromAlpha(alpha: number, screenAngle = 0): number {
	return normalizeDeg(360 - alpha + screenAngle);
}

type CompassEvent = DeviceOrientationEvent & { webkitCompassHeading?: number };

function createCompass() {
	const supported =
		typeof window !== 'undefined' && typeof window.DeviceOrientationEvent !== 'undefined';
	const heading = writable<number | null>(null);
	let smoothed: number | null = null;
	let eventName: 'deviceorientationabsolute' | 'deviceorientation' = 'deviceorientation';
	let started = false;

	function onEvent(ev: Event) {
		const e = ev as CompassEvent;
		let h: number | null = null;
		if (typeof e.webkitCompassHeading === 'number' && !Number.isNaN(e.webkitCompassHeading)) {
			h = e.webkitCompassHeading; // iOS: already clockwise-from-north
		} else if (e.absolute && typeof e.alpha === 'number') {
			const screenAngle =
				(typeof screen !== 'undefined' && screen.orientation && screen.orientation.angle) || 0;
			h = androidHeadingFromAlpha(e.alpha, screenAngle);
		}
		if (h === null) return;
		smoothed = circularLowPass(smoothed, h);
		heading.set(smoothed);
	}

	function start() {
		if (started || !supported) return;
		started = true;
		eventName = 'ondeviceorientationabsolute' in window ? 'deviceorientationabsolute' : 'deviceorientation';
		window.addEventListener(eventName, onEvent, true);
	}

	// enable requests permission where required (iOS) and starts listening.
	// Returns whether the compass is now active.
	async function enable(): Promise<boolean> {
		if (!supported) return false;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		const D = window.DeviceOrientationEvent as any;
		try {
			if (typeof D.requestPermission === 'function') {
				const res = await D.requestPermission();
				if (res !== 'granted') return false;
			}
		} catch {
			return false;
		}
		start();
		return true;
	}

	function stop() {
		if (started) {
			window.removeEventListener(eventName, onEvent, true);
			started = false;
		}
		smoothed = null;
		heading.set(null);
	}

	return { subscribe: heading.subscribe, enable, stop, supported };
}

export const compass = createCompass();
