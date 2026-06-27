<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount, onDestroy } from 'svelte';
	import { geo, type GeoState } from '$lib/stores/geo';
	import { fetchChargerDetail } from '$lib/api/client';
	import { fetchRoadGraph, bboxFor } from '$lib/routing/overpass';
	import { routeWithNames, haversineKm, type LatLng } from '$lib/routing/router';
	import {
		buildManeuvers,
		cumulativeKm,
		snapToRoute,
		nextManeuver,
		bearing,
		instructionText,
		formatDistance,
		type Maneuver,
		type TurnType
	} from '$lib/routing/navigation';
	import { compass } from '$lib/sensors/compass';
	import 'leaflet/dist/leaflet.css';

	const URBAN_KMH = 30;
	const ARRIVE_M = 25;
	const OFFROUTE_M = 45;
	const REROUTE_MS = 6000;

	const id = $derived($page.params.id ?? '');

	let target = $state<LatLng | null>(null);
	let operator = $state('');
	let status = $state<'locating' | 'routing' | 'rerouting' | 'navigating' | 'arrived' | 'error'>(
		'locating'
	);
	let banner = $state<{ text: string; dist: string; type: TurnType } | null>(null);
	let upcoming = $state<string | null>(null);
	let remaining = $state('');
	let etaMin = $state<number | null>(null);
	let course = $state(0);
	let muted = $state(false);
	type Orient = 'north' | 'course' | 'compass';
	let orient = $state<Orient>('north');
	const compassSupported = compass.supported;
	let compassUnsub: (() => void) | null = null;
	let lastPos: LatLng | null = null;

	const ORIENT_ICON: Record<Orient, string> = { north: 'N', course: '➤', compass: '🧭' };
	const ORIENT_LABEL: Record<Orient, string> = {
		north: 'North-up',
		course: 'Course-up',
		compass: 'Compass'
	};

	let points: LatLng[] = [];
	let cum: number[] = [];
	let maneuvers: Maneuver[] = [];
	let offRouteCount = 0;
	let lastRerouteTs = 0;
	let fetchingTarget = false;
	const announced = new Set<string>();

	// --- Leaflet ---
	let el: HTMLDivElement;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let map: any = null;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let L: any = null;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let routeLine: any = null;

	onMount(async () => {
		L = (await import('leaflet')).default;
		map = L.map(el, { zoomControl: false, attributionControl: true, dragging: false, keyboard: false });
		L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
			maxZoom: 19,
			attribution: '© OSM'
		}).addTo(map);
		map.setView([0, 0], 17);
		setTimeout(() => map && map.invalidateSize(), 50);
		void requestWakeLock();
		document.addEventListener('visibilitychange', onVisibility);
		geoUnsub = geo.subscribe(onGeo);
		// Compass heading (when enabled) overrides the GPS-derived course.
		compassUnsub = compass.subscribe((h) => {
			if (orient === 'compass' && h !== null) course = h;
		});
	});

	// Cycle North-up → Course-up → Compass (Compass only when supported).
	async function cycleOrient() {
		const order: Orient[] = compassSupported ? ['north', 'course', 'compass'] : ['north', 'course'];
		let next = order[(order.indexOf(orient) + 1) % order.length];
		if (orient === 'compass') compass.stop();
		if (next === 'compass') {
			const ok = await compass.enable();
			if (!ok) next = 'north'; // permission denied / no sensor
		}
		orient = next;
		// North-up resets to 0; Course-up re-derives on the next movement.
		if (orient !== 'compass') course = 0;
	}

	function drawRoute() {
		if (!map || !L) return;
		if (routeLine) routeLine.remove();
		routeLine = L.polyline(
			points.map((p) => [p.lat, p.lon]),
			{ color: '#34e0e0', weight: 6, opacity: 0.9 }
		).addTo(map);
	}

	function recenter(p: LatLng) {
		map?.setView([p.lat, p.lon], 17, { animate: true, duration: 0.4 });
	}

	function speak(text: string) {
		if (muted || typeof window === 'undefined' || !('speechSynthesis' in window)) return;
		const u = new SpeechSynthesisUtterance(text);
		u.lang = 'en-US';
		window.speechSynthesis.speak(u);
	}

	async function ensureTarget(pos: LatLng) {
		if (fetchingTarget || target) return;
		fetchingTarget = true;
		try {
			const d = await fetchChargerDetail(id, pos);
			target = { lat: d.lat, lon: d.lon };
			operator = d.operator;
			// Kick off routing immediately rather than waiting for the next fix.
			if (status === 'locating') void computeRoute(pos, target);
		} catch {
			status = 'error';
		} finally {
			fetchingTarget = false;
		}
	}

	async function computeRoute(from: LatLng, to: LatLng) {
		status = points.length ? 'rerouting' : 'routing';
		try {
			const graph = await fetchRoadGraph(bboxFor(from, to, 0.8));
			const rp = routeWithNames(graph, from, to);
			if (!rp) {
				status = 'error';
				return;
			}
			points = rp.points;
			cum = cumulativeKm(points);
			maneuvers = buildManeuvers(rp);
			announced.clear();
			drawRoute();
			status = 'navigating';
			update(from);
		} catch {
			status = 'error';
		}
	}

	function voiceAnnounce(m: Maneuver, distKm: number) {
		const dm = distKm * 1000;
		const key = m.pointIndex;
		if (dm <= 60 && !announced.has('n' + key)) {
			announced.add('n' + key);
			speak(instructionText(m));
		} else if (dm <= 220 && !announced.has('f' + key)) {
			announced.add('f' + key);
			speak(`In ${formatDistance(distKm)}, ${instructionText(m).toLowerCase()}`);
		}
	}

	function update(pos: LatLng) {
		if (!points.length || !target) return;

		// Course-up derives heading from movement. North-up keeps course at 0;
		// Compass updates course via its own subscription.
		if (orient === 'course' && lastPos && haversineKm(lastPos, pos) * 1000 > 3) {
			course = bearing(lastPos, pos);
		}
		lastPos = pos;

		const snap = snapToRoute(points, cum, pos);
		const total = cum[cum.length - 1];
		const remKm = Math.max(0, total - snap.distAlongKm);
		remaining = formatDistance(remKm);
		etaMin = Math.max(1, Math.round((remKm / URBAN_KMH) * 60));

		if (haversineKm(pos, target) * 1000 <= ARRIVE_M) {
			if (status !== 'arrived') speak('You have arrived.');
			status = 'arrived';
			banner = { text: 'You have arrived', dist: '', type: 'arrive' };
			recenter(pos);
			return;
		}

		// Off-route → reroute (debounced).
		if (snap.distToRouteKm * 1000 > OFFROUTE_M) offRouteCount++;
		else offRouteCount = 0;
		if (offRouteCount >= 2 && Date.now() - lastRerouteTs > REROUTE_MS) {
			lastRerouteTs = Date.now();
			offRouteCount = 0;
			speak('Rerouting.');
			void computeRoute(pos, target);
			return;
		}

		const nm = nextManeuver(maneuvers, snap.distAlongKm);
		if (nm) {
			banner = { text: instructionText(nm.maneuver), dist: formatDistance(nm.distKm), type: nm.maneuver.type };
			const idx = maneuvers.indexOf(nm.maneuver);
			const after = maneuvers[idx + 1];
			upcoming = after && after.type !== 'arrive' ? `Then ${instructionText(after)}` : null;
			voiceAnnounce(nm.maneuver, nm.distKm);
		}
		recenter(snap.snapped);
	}

	// Drive the whole flow off geolocation updates. Use an explicit store
	// subscription (not $effect) so every watchPosition tick is handled — the
	// same pattern the charger poller uses.
	let geoUnsub: (() => void) | null = null;
	function onGeo(s: GeoState) {
		if (s.status !== 'live') return;
		const pos = { lat: s.lat, lon: s.lon };
		if (!target) {
			void ensureTarget(pos);
			return;
		}
		if (status === 'locating') {
			void computeRoute(pos, target);
			return;
		}
		if (status === 'navigating' || status === 'arrived') update(pos);
	}

	// --- Wake lock ---
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let wakeLock: any = null;
	async function requestWakeLock() {
		try {
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			wakeLock = await (navigator as any).wakeLock?.request('screen');
		} catch {
			/* unsupported or denied — non-fatal */
		}
	}
	function onVisibility() {
		if (document.visibilityState === 'visible' && !wakeLock) void requestWakeLock();
	}

	function end() {
		void goto(`/charger/${id}`);
	}

	const GLYPH: Record<TurnType, string> = {
		depart: '↑',
		straight: '↑',
		'slight-left': '↖',
		'slight-right': '↗',
		left: '←',
		right: '→',
		'sharp-left': '↰',
		'sharp-right': '↱',
		uturn: '↩',
		arrive: '⚑'
	};

	onDestroy(() => {
		geoUnsub?.();
		geoUnsub = null;
		compassUnsub?.();
		compassUnsub = null;
		compass.stop();
		if (typeof document !== 'undefined') document.removeEventListener('visibilitychange', onVisibility);
		try {
			wakeLock?.release?.();
		} catch {
			/* ignore */
		}
		wakeLock = null;
		if (typeof window !== 'undefined' && 'speechSynthesis' in window) window.speechSynthesis.cancel();
		if (map) {
			map.remove();
			map = null;
		}
	});
</script>

<svelte:head><title>Navigating · voltpilot</title></svelte:head>

<div class="stage">
	<div class="rot" style:transform={`rotate(${-course}deg)`}>
		<div class="map" bind:this={el} data-testid="nav-map"></div>
	</div>
	<div class="puck" aria-hidden="true"></div>

	<div class="banner" data-testid="nav-banner">
		{#if status === 'locating'}
			<div class="line1">Waiting for location…</div>
		{:else if status === 'routing'}
			<div class="line1">Calculating route…</div>
		{:else if status === 'rerouting'}
			<div class="line1">Rerouting…</div>
		{:else if status === 'error'}
			<div class="line1">Route unavailable</div>
			<div class="line2">Use the nav apps on the previous screen.</div>
		{:else if banner}
			<div class="row">
				<span class="glyph" class:arrive={banner.type === 'arrive'}>{GLYPH[banner.type]}</span>
				<div>
					{#if banner.dist}<div class="dist mono">{banner.dist}</div>{/if}
					<div class="line1">{banner.text}</div>
					{#if upcoming && status === 'navigating'}<div class="line2">{upcoming}</div>{/if}
				</div>
			</div>
		{/if}
	</div>

	<div class="tools">
		<button
			class="tool orient"
			class:active={orient !== 'north'}
			onclick={cycleOrient}
			aria-label={`Map orientation: ${ORIENT_LABEL[orient]}`}
			title={`Orientation: ${ORIENT_LABEL[orient]} — tap to change`}
			data-testid="orient-toggle"
		>
			{ORIENT_ICON[orient]}
		</button>
		<button class="tool" onclick={() => (muted = !muted)} aria-label="Toggle voice" title="Toggle voice">
			{muted ? '🔇' : '🔊'}
		</button>
	</div>

	<div class="bottom">
		<div class="trip">
			{#if operator}<span class="op">{operator}</span>{/if}
			<span class="mono">{remaining}{etaMin !== null ? ` · ~${etaMin} min` : ''}</span>
		</div>
		<button class="end" onclick={end} data-testid="nav-end">End</button>
	</div>
</div>

<style>
	.stage {
		position: fixed;
		inset: 0;
		overflow: hidden;
		background: var(--bg);
		z-index: 50;
	}
	/* Oversized + centred so heading-up rotation never reveals empty corners. */
	.rot {
		position: absolute;
		top: -39%;
		left: -39%;
		width: 178%;
		height: 178%;
		transform-origin: center center;
		transition: transform 0.4s linear;
	}
	.map {
		width: 100%;
		height: 100%;
	}
	:global(.leaflet-container) {
		background: var(--surface);
	}
	/* Travel-direction puck pinned at the visual centre (map rotates under it). */
	.puck {
		position: absolute;
		top: 50%;
		left: 50%;
		width: 0;
		height: 0;
		border-left: 11px solid transparent;
		border-right: 11px solid transparent;
		border-bottom: 22px solid var(--cool);
		transform: translate(-50%, -50%);
		filter: drop-shadow(0 0 6px rgba(77, 124, 255, 0.6));
		z-index: 60;
	}
	.banner {
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		margin: 0.75rem;
		padding: 0.9rem 1rem;
		border-radius: var(--radius-card);
		background: color-mix(in srgb, var(--bg-elev) 92%, transparent);
		border: 1px solid var(--border-strong);
		backdrop-filter: blur(8px);
		z-index: 60;
	}
	.row {
		display: flex;
		align-items: center;
		gap: 0.9rem;
	}
	.glyph {
		font-size: 2.4rem;
		line-height: 1;
		color: var(--accent);
		min-width: 2.4rem;
		text-align: center;
	}
	.glyph.arrive {
		color: var(--ok);
	}
	.dist {
		font-size: 1.5rem;
		font-weight: 700;
		color: var(--accent);
	}
	.line1 {
		font-size: 1.1rem;
		font-weight: 600;
		color: var(--text-strong);
	}
	.line2 {
		font-size: 0.85rem;
		color: var(--muted);
		margin-top: 0.15rem;
	}
	.tools {
		position: absolute;
		bottom: 5.5rem;
		right: 0.75rem;
		z-index: 70;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.tool {
		width: 2.75rem;
		height: 2.75rem;
		border-radius: 999px;
		background: color-mix(in srgb, var(--bg-elev) 90%, transparent);
		border: 1px solid var(--border-strong);
		font-size: 1.1rem;
	}
	.tool.active {
		border-color: var(--accent);
		background: color-mix(in srgb, var(--accent) 22%, var(--bg-elev));
	}
	.tool.orient {
		font-family: var(--font-display);
		font-weight: 700;
	}
	.bottom {
		position: absolute;
		left: 0;
		right: 0;
		bottom: 0;
		margin: 0.75rem;
		padding: 0.75rem 1rem;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		border-radius: var(--radius-card);
		background: color-mix(in srgb, var(--bg-elev) 92%, transparent);
		border: 1px solid var(--border-strong);
		backdrop-filter: blur(8px);
		z-index: 60;
	}
	.trip {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
	}
	.op {
		font-size: 0.75rem;
		color: var(--muted);
	}
	.trip .mono {
		font-size: 1.05rem;
		font-weight: 700;
		color: var(--text-strong);
	}
	.end {
		min-height: 2.75rem;
		padding: 0.6rem 1.4rem;
		border-radius: var(--radius-pill);
		background: var(--danger);
		color: #fff;
		font-family: var(--font-display);
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
</style>
