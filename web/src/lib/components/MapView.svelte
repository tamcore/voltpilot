<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import type { Charger } from '$lib/types/api';
	import 'leaflet/dist/leaflet.css';

	let {
		chargers,
		center
	}: { chargers: Charger[]; center: { lat: number; lon: number } } = $props();

	let el: HTMLDivElement;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let map: any = null;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let L: any = null;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let markersLayer: any = null;

	onMount(async () => {
		L = (await import('leaflet')).default;
		map = L.map(el, { zoomControl: true, attributionControl: true }).setView(
			[center.lat, center.lon],
			12
		);
		L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
			maxZoom: 19,
			attribution: '© OpenStreetMap'
		}).addTo(map);
		L.circleMarker([center.lat, center.lon], {
			radius: 6,
			color: '#4d7cff',
			fillColor: '#4d7cff',
			fillOpacity: 1
		}).addTo(map);
		markersLayer = L.layerGroup().addTo(map);
		render();
	});

	function render() {
		if (!map || !L || !markersLayer) return;
		markersLayer.clearLayers();
		for (const c of chargers) {
			const color = c.available ? '#2ee27a' : '#5f6c8c';
			const m = L.circleMarker([c.lat, c.lon], {
				radius: 8,
				color,
				fillColor: color,
				fillOpacity: 0.85,
				weight: 2
			});
			m.bindPopup(
				`<strong>${c.operator}</strong><br/>${Math.round(c.maxPowerKw)} kW · ${c.availableChargePoints}/${c.numberOfChargePoints} free<br/><a href="/charger/${c.id}">Details →</a>`
			);
			m.on('popupopen', () => {
				const link = document.querySelector('.leaflet-popup-content a');
				link?.addEventListener('click', (ev) => {
					ev.preventDefault();
					void goto(`/charger/${c.id}`);
				});
			});
			m.addTo(markersLayer);
		}
	}

	// Re-render markers whenever the charger list changes.
	$effect(() => {
		void chargers;
		render();
	});

	onDestroy(() => {
		if (map) {
			map.remove();
			map = null;
		}
	});
</script>

<div class="map" bind:this={el} data-testid="map"></div>

<style>
	.map {
		height: 60vh;
		min-height: 360px;
		border-radius: var(--radius-card);
		overflow: hidden;
		border: 1px solid var(--border);
	}
	:global(.leaflet-container) {
		background: var(--surface);
		font-family: var(--font-body);
	}
</style>
