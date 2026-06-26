<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { LatLng } from '$lib/routing/router';
	import 'leaflet/dist/leaflet.css';

	let {
		user,
		target,
		route = null
	}: { user: LatLng; target: LatLng; route?: LatLng[] | null } = $props();

	let el: HTMLDivElement;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let map: any = null;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let L: any = null;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let lineLayer: any = null;

	onMount(async () => {
		L = (await import('leaflet')).default;
		map = L.map(el, { zoomControl: false, attributionControl: true, dragging: true });
		L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
			maxZoom: 19,
			attribution: '© OpenStreetMap'
		}).addTo(map);
		L.circleMarker([user.lat, user.lon], {
			radius: 6,
			color: '#4d7cff',
			fillColor: '#4d7cff',
			fillOpacity: 1,
			weight: 2
		})
			.addTo(map)
			.bindTooltip('You');
		L.circleMarker([target.lat, target.lon], {
			radius: 7,
			color: '#34e0e0',
			fillColor: '#34e0e0',
			fillOpacity: 0.9,
			weight: 2
		})
			.addTo(map)
			.bindTooltip('Charger');
		draw();
	});

	function draw() {
		if (!map || !L) return;
		if (lineLayer) {
			lineLayer.remove();
			lineLayer = null;
		}
		const pts = route ?? [user, target];
		const latlngs = pts.map((p) => [p.lat, p.lon]);
		lineLayer = L.polyline(latlngs, {
			color: '#34e0e0',
			weight: 4,
			opacity: 0.9,
			// Dashed when we only have a straight-line fallback (no real route).
			dashArray: route ? undefined : '6 8'
		}).addTo(map);
		map.fitBounds(lineLayer.getBounds(), { padding: [28, 28], maxZoom: 16 });
	}

	$effect(() => {
		void route;
		draw();
	});

	onDestroy(() => {
		if (map) {
			map.remove();
			map = null;
		}
	});
</script>

<div class="map" bind:this={el} data-testid="route-map"></div>

<style>
	.map {
		height: 240px;
		border-radius: var(--radius-card);
		overflow: hidden;
		border: 1px solid var(--border);
		margin-bottom: 1rem;
	}
	:global(.leaflet-container) {
		background: var(--surface);
		font-family: var(--font-body);
	}
</style>
