<script lang="ts">
	import { page } from '$app/stores';
	import { geo, distanceKm } from '$lib/stores/geo';
	import { fetchChargerDetail, ApiError } from '$lib/api/client';
	import type { ChargerDetail } from '$lib/types/api';
	import CurrentBadge from '$lib/components/CurrentBadge.svelte';

	let detail = $state<ChargerDetail | null>(null);
	let error = $state<string | null>(null);
	let fetchedId = $state<string | null>(null);

	const id = $derived($page.params.id ?? '');

	async function load() {
		if (!id || $geo.status !== 'live') return;
		try {
			error = null;
			detail = await fetchChargerDetail(id, { lat: $geo.lat, lon: $geo.lon });
			fetchedId = id;
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Could not load this charger.';
		}
	}

	// Load once geolocation is live (so distance is accurate) and the id is known.
	$effect(() => {
		if (id && id !== fetchedId && $geo.status === 'live') void load();
	});

	const liveDistanceKm = $derived(
		detail && $geo.status === 'live'
			? distanceKm({ lat: $geo.lat, lon: $geo.lon }, { lat: detail.lat, lon: detail.lon })
			: detail?.distanceKm ?? null
	);

	function statusLabel(s: string): string {
		const v = s.toUpperCase();
		if (v === 'AVAILABLE') return 'Available';
		if (v === 'OCCUPIED') return 'In use';
		if (v === 'OUT_OF_SERVICE') return 'Out of service';
		return 'Unknown';
	}
</script>

<svelte:head><title>{detail ? detail.operator : 'Charger'} · voltpilot</title></svelte:head>

<a class="back" href="/">← Back</a>

{#if error}
	<p class="err">{error}</p>
{:else if !detail}
	<p class="loading">Loading charger…</p>
{:else}
	<header class="head">
		<h1>{detail.operator}</h1>
		{#if detail.address}<p class="addr">{detail.address}</p>{/if}
	</header>

	<div class="stats">
		<div class="stat">
			<span class="num mono">{liveDistanceKm !== null ? liveDistanceKm.toFixed(1) : '—'}</span>
			<span class="unit">km away</span>
		</div>
		<div class="stat">
			<span class="num mono">{Math.round(detail.maxPowerKw)}</span>
			<span class="unit">kW max</span>
		</div>
		<div class="stat">
			<span class="num mono" class:ok={detail.available}
				>{detail.availableChargePoints}/{detail.numberOfChargePoints}</span
			>
			<span class="unit">available</span>
		</div>
		<div class="stat"><CurrentBadge current={detail.current} /></div>
	</div>

	<div class="section-label">Navigate</div>
	<div class="nav-buttons">
		<a class="btn primary" href={detail.deep_links.google} target="_blank" rel="noopener">Google Maps</a>
		<a class="btn" href={detail.deep_links.apple} target="_blank" rel="noopener">Apple Maps</a>
		<a class="btn" href={detail.deep_links.waze} target="_blank" rel="noopener">Waze</a>
	</div>

	{#if detail.chargePoints.length}
		<div class="section-label">Charge points</div>
		<ul class="cps">
			{#each detail.chargePoints as cp (cp.evseId)}
				<li class="cp">
					<div class="cp-top">
						<span class="evse mono">{cp.evseId}</span>
						<span
							class="status"
							class:ok={cp.available}
							class:busy={!cp.available}
							data-testid="cp-status">{statusLabel(cp.status)}</span
						>
					</div>
					<div class="connectors">
						{#each cp.connectors as cn (cn.plugTypeName + cn.maxPowerKw)}
							<span class="conn">
								<CurrentBadge current={cn.current} />
								{cn.plugTypeName} · {Math.round(cn.maxPowerKw)} kW
							</span>
						{/each}
					</div>
				</li>
			{/each}
		</ul>
	{/if}
{/if}

<style>
	.back {
		display: inline-block;
		margin: 0.5rem 0 1rem;
		font-family: var(--font-display);
		font-size: 0.78rem;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--muted);
		text-decoration: none;
	}
	.head h1 {
		margin: 0;
		font-size: clamp(1.4rem, 5vw, 2rem);
		color: var(--text-strong);
	}
	.addr {
		color: var(--muted);
		margin: 0.25rem 0 0;
	}
	.stats {
		display: flex;
		flex-wrap: wrap;
		gap: 0.75rem;
		margin: 1.25rem 0;
	}
	.stat {
		flex: 1 1 5rem;
		min-width: 5rem;
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		align-items: flex-start;
		padding: 0.75rem 0.9rem;
		border-radius: var(--radius-card);
		background: var(--surface);
		border: 1px solid var(--border);
	}
	.num {
		font-size: 1.4rem;
		font-weight: 700;
		color: var(--text-strong);
	}
	.num.ok {
		color: var(--ok);
	}
	.unit {
		font-size: 0.72rem;
		text-transform: uppercase;
		letter-spacing: 0.1em;
		color: var(--muted-2);
	}
	.nav-buttons {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
	}
	.btn {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 3.5rem;
		padding: 1rem 1.15rem;
		border-radius: var(--radius-card);
		border: 1px solid var(--border-strong);
		background: var(--surface);
		color: var(--text-strong);
		text-decoration: none;
		font-weight: 600;
		font-size: 1.05rem;
	}
	.btn.primary {
		background: var(--accent);
		color: var(--bg);
		border-color: var(--accent);
	}
	.cps {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.cp {
		padding: 0.7rem 0.9rem;
		border-radius: var(--radius-card);
		background: var(--surface);
		border: 1px solid var(--border);
	}
	.cp-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
	}
	.evse {
		font-size: 0.74rem;
		color: var(--muted);
	}
	.status {
		font-size: 0.78rem;
		font-weight: 600;
	}
	.status.ok {
		color: var(--ok);
	}
	.status.busy {
		color: var(--muted);
	}
	.connectors {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		margin-top: 0.5rem;
	}
	.conn {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.8rem;
		color: var(--text);
	}
	.loading,
	.err {
		padding: 2rem 1rem;
		text-align: center;
		color: var(--muted);
	}
	.err {
		color: var(--danger);
	}
</style>
