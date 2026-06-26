<script lang="ts">
	import type { Charger } from '$lib/types/api';
	import CurrentBadge from './CurrentBadge.svelte';

	let { charger }: { charger: Charger } = $props();

	const plugLabel = $derived(
		(charger.plugTypeNames?.length ? charger.plugTypeNames : charger.plugTypes).join(' · ')
	);
</script>

<a class="card" href={`/charger/${charger.id}`} data-testid="charger-card">
	<div class="row top">
		<span class="operator">{charger.operator}</span>
		<span class="dist mono">{charger.distanceKm.toFixed(1)} km</span>
	</div>

	<div class="row meta">
		<CurrentBadge current={charger.current} />
		<span class="power mono">{Math.round(charger.maxPowerKw)} kW</span>
		{#if plugLabel}
			<span class="plugs">{plugLabel}</span>
		{/if}
	</div>

	<div class="row bottom">
		<span
			class="avail"
			class:ok={charger.available}
			class:none={!charger.available}
			data-testid="availability"
		>
			<span class="dot"></span>
			{charger.availableChargePoints}/{charger.numberOfChargePoints} available
		</span>
		{#if charger.alwaysOpen}
			<span class="tag">24/7</span>
		{/if}
	</div>
	{#if charger.address}
		<div class="addr">{charger.address}</div>
	{/if}
</a>

<style>
	.card {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		padding: 0.85rem 1rem;
		border-radius: var(--radius-card);
		background: var(--surface);
		border: 1px solid var(--border);
		text-decoration: none;
		color: var(--text);
		animation: rise 0.25s var(--ease-out) both;
		transition: border-color 0.18s, transform 0.12s;
	}
	.card:hover {
		border-color: var(--border-strong);
	}
	.card:active {
		transform: scale(0.99);
	}
	.row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.top {
		justify-content: space-between;
	}
	.operator {
		font-weight: 600;
		font-size: 1rem;
		color: var(--text-strong);
	}
	.dist {
		color: var(--accent);
		font-weight: 700;
		font-size: 0.95rem;
	}
	.meta {
		flex-wrap: wrap;
	}
	.power {
		font-weight: 700;
		color: var(--text);
		font-size: 0.85rem;
	}
	.plugs {
		font-size: 0.78rem;
		color: var(--muted);
	}
	.bottom {
		justify-content: space-between;
	}
	.avail {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.82rem;
		font-weight: 500;
	}
	.avail .dot {
		width: 0.55rem;
		height: 0.55rem;
		border-radius: 999px;
	}
	.avail.ok {
		color: var(--ok);
	}
	.avail.ok .dot {
		background: var(--ok);
		box-shadow: 0 0 6px var(--ok-glow);
	}
	.avail.none {
		color: var(--muted);
	}
	.avail.none .dot {
		background: var(--muted-2);
	}
	.tag {
		font-family: var(--font-display);
		font-size: 0.68rem;
		letter-spacing: 0.14em;
		color: var(--muted);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		padding: 0.1rem 0.4rem;
	}
	.addr {
		font-size: 0.76rem;
		color: var(--muted-2);
	}
</style>
