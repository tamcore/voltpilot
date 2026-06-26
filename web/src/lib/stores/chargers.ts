import { writable, get } from 'svelte/store';
import { fetchChargers, ApiError } from '$lib/api/client';
import { geo, type GeoState } from '$lib/stores/geo';
import { filters } from '$lib/stores/filters';
import { preferredCpo } from '$lib/stores/cpo';
import type { Charger } from '$lib/types/api';

export type ChargersState = {
	chargers: Charger[];
	loading: boolean;
	lastError: string | null;
	loadedOnce: boolean;
};

const INITIAL: ChargersState = { chargers: [], loading: false, lastError: null, loadedOnce: false };

const SEARCH_RADIUS_KM = 25;
const RESULT_LIMIT = 30;
const POLL_INTERVAL_MS = 30_000;

function createChargersStore() {
	const inner = writable<ChargersState>(INITIAL);

	let pollTimer: ReturnType<typeof setInterval> | null = null;
	let inflight: AbortController | null = null;
	let unsubs: Array<() => void> = [];
	let running = false;
	let latestGeo: GeoState = { status: 'idle' };

	async function refresh() {
		if (latestGeo.status !== 'live') return;
		const cpo = get(preferredCpo);
		if (!cpo) return; // no CPO chosen yet — nothing to list
		inflight?.abort();
		inflight = new AbortController();
		inner.update((s) => ({ ...s, loading: true, lastError: null }));
		try {
			const f = get(filters);
			const chargers = await fetchChargers({
				lat: latestGeo.lat,
				lon: latestGeo.lon,
				radiusKm: SEARCH_RADIUS_KM,
				operatorCode: cpo.all ? undefined : cpo.operatorCode,
				current: f.current,
				availableOnly: f.availableOnly,
				limit: RESULT_LIMIT,
				signal: inflight.signal
			});
			inflight = null;
			inner.set({ chargers, loading: false, lastError: null, loadedOnce: true });
		} catch (err) {
			inflight = null;
			if (err instanceof DOMException && err.name === 'AbortError') return;
			const msg = err instanceof ApiError ? err.message : 'Network error';
			inner.update((s) => ({ ...s, loading: false, lastError: msg, loadedOnce: true }));
		}
	}

	function scheduleTimer() {
		if (pollTimer) clearInterval(pollTimer);
		pollTimer = setInterval(() => void refresh(), POLL_INTERVAL_MS);
	}

	function start() {
		if (running) return;
		running = true;
		scheduleTimer();

		unsubs.push(
			geo.subscribe((s) => {
				const wasLive = latestGeo.status === 'live';
				latestGeo = s;
				if (s.status === 'live' && !wasLive) void refresh();
			})
		);
		// React to CPO / filter changes immediately (skip the initial fire).
		let cpoInit = true;
		unsubs.push(
			preferredCpo.subscribe(() => {
				if (cpoInit) {
					cpoInit = false;
					return;
				}
				inner.set(INITIAL);
				void refresh();
			})
		);
		let filterInit = true;
		unsubs.push(
			filters.subscribe(() => {
				if (filterInit) {
					filterInit = false;
					return;
				}
				void refresh();
			})
		);
	}

	function stop() {
		if (!running) return;
		running = false;
		if (pollTimer) clearInterval(pollTimer);
		pollTimer = null;
		inflight?.abort();
		inflight = null;
		for (const u of unsubs) u();
		unsubs = [];
		latestGeo = { status: 'idle' };
		inner.set(INITIAL);
	}

	return { subscribe: inner.subscribe, start, stop, refresh };
}

export const chargersPoller = createChargersStore();
