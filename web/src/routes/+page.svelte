<script lang="ts">
	import { geo } from '$lib/stores/geo';
	import { preferredCpo } from '$lib/stores/cpo';
	import { filters } from '$lib/stores/filters';
	import { chargersPoller } from '$lib/stores/chargers';
	import { fetchCpos } from '$lib/api/client';
	import type { CPO } from '$lib/types/api';

	import GeoStatusPanel from '$lib/components/GeoStatusPanel.svelte';
	import CpoPicker from '$lib/components/CpoPicker.svelte';
	import FilterChips from '$lib/components/FilterChips.svelte';
	import ChargerCard from '$lib/components/ChargerCard.svelte';
	import MapView from '$lib/components/MapView.svelte';

	const CPO_RADIUS_KM = 25;

	let cpos = $state<CPO[]>([]);
	let cposLoading = $state(false);
	let cposError = $state<string | null>(null);
	let cposFetchedFor = $state<string | null>(null); // coarse geo key we fetched CPOs for
	let view = $state<'list' | 'map'>('list');

	function geoKey(): string | null {
		if ($geo.status !== 'live') return null;
		return `${$geo.lat.toFixed(2)},${$geo.lon.toFixed(2)}`;
	}

	async function loadCpos() {
		if ($geo.status !== 'live') return;
		cposLoading = true;
		cposError = null;
		try {
			cpos = await fetchCpos({ lat: $geo.lat, lon: $geo.lon }, CPO_RADIUS_KM);
		} catch {
			cposError = 'Could not load operators. Check your connection and try again.';
		} finally {
			cposLoading = false;
		}
	}

	// Load the CPO chooser list once we have a location and no CPO is chosen yet.
	$effect(() => {
		const key = geoKey();
		if (!$preferredCpo && key && key !== cposFetchedFor) {
			cposFetchedFor = key;
			void loadCpos();
		}
	});

	function pick(cpo: CPO) {
		preferredCpo.choose(cpo);
	}

	function changeCpo() {
		preferredCpo.clear();
		cposFetchedFor = null;
	}

	const center = $derived(
		$geo.status === 'live' ? { lat: $geo.lat, lon: $geo.lon } : null
	);
</script>

<svelte:head><title>voltpilot</title></svelte:head>

{#if $geo.status !== 'live'}
	<GeoStatusPanel state={$geo} />
{/if}

{#if !$preferredCpo}
	<section class="hero">
		<h1 class="display">Nearest available charger</h1>
		{#if cposError}
			<p class="err">{cposError}</p>
		{/if}
		{#if center}
			<CpoPicker {cpos} loading={cposLoading} onpick={pick} />
		{/if}
	</section>
{:else}
	<section class="cpo-bar">
		<div>
			<span class="kicker">Operator</span>
			<span class="cpo-name" data-testid="active-cpo">{$preferredCpo.operator}</span>
		</div>
		<button type="button" class="change" onclick={changeCpo} data-testid="change-cpo">Change</button>
	</section>

	<div class="controls">
		<FilterChips />
		<div class="view-toggle" role="group" aria-label="View">
			<button
				type="button"
				class:active={view === 'list'}
				onclick={() => (view = 'list')}
				data-testid="view-list">List</button
			>
			<button
				type="button"
				class:active={view === 'map'}
				onclick={() => (view = 'map')}
				data-testid="view-map">Map</button
			>
		</div>
	</div>

	{#if $chargersPoller.lastError}
		<p class="err">{$chargersPoller.lastError}</p>
	{/if}

	{#if view === 'map' && center}
		{#key $preferredCpo.operatorCode}
			<MapView chargers={$chargersPoller.chargers} {center} />
		{/key}
	{:else}
		<div class="list" data-testid="charger-list">
			{#if $chargersPoller.chargers.length === 0 && $chargersPoller.loadedOnce && !$chargersPoller.loading}
				<p class="empty">No matching chargers within 25 km. Try widening the filters.</p>
			{:else if !$chargersPoller.loadedOnce && $chargersPoller.loading}
				<p class="empty">Finding chargers…</p>
			{:else}
				{#each $chargersPoller.chargers as c (c.id)}
					<ChargerCard charger={c} />
				{/each}
			{/if}
		</div>
	{/if}
{/if}

<style>
	.hero h1 {
		font-size: clamp(1.6rem, 6vw, 2.2rem);
		margin: 1rem 0 1.25rem;
		color: var(--text-strong);
	}
	.cpo-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding: 0.75rem 1rem;
		border-radius: var(--radius-card);
		background: var(--surface);
		border: 1px solid var(--border);
		margin: 0.75rem 0;
	}
	.kicker {
		display: block;
		font-family: var(--font-display);
		font-size: 0.66rem;
		letter-spacing: 0.2em;
		text-transform: uppercase;
		color: var(--muted-2);
	}
	.cpo-name {
		font-weight: 600;
		font-size: 1.05rem;
		color: var(--text-strong);
	}
	.change {
		font-family: var(--font-display);
		font-size: 0.72rem;
		letter-spacing: 0.14em;
		text-transform: uppercase;
		color: var(--accent);
		border: 1px solid color-mix(in srgb, var(--accent) 40%, transparent);
		border-radius: var(--radius-pill);
		padding: 0.4rem 0.8rem;
	}
	.controls {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
		flex-wrap: wrap;
		margin-bottom: 1rem;
	}
	.view-toggle {
		display: inline-flex;
		border: 1px solid var(--border);
		border-radius: var(--radius-pill);
		background: var(--bg-elev);
		padding: 2px;
	}
	.view-toggle button {
		font-family: var(--font-display);
		font-size: 0.74rem;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--muted);
		padding: 0.3rem 0.8rem;
		border-radius: var(--radius-pill);
	}
	.view-toggle button.active {
		background: var(--accent);
		color: var(--bg);
	}
	.list {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
	}
	.empty {
		color: var(--muted);
		text-align: center;
		padding: 2rem 1rem;
	}
	.err {
		color: var(--danger);
		font-size: 0.88rem;
	}
</style>
