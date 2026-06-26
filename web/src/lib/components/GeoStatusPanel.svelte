<script lang="ts">
	import type { GeoState } from '$lib/stores/geo';
	let { state }: { state: GeoState } = $props();

	const message = $derived(
		state.status === 'pending'
			? 'Locating you…'
			: state.status === 'permission-denied'
				? 'Location permission denied. Enable it to see chargers near you.'
				: state.status === 'unavailable'
					? 'Location is unavailable on this device.'
					: ''
	);
</script>

{#if message}
	<div class="panel" class:warn={state.status !== 'pending'} role="status">
		{message}
	</div>
{/if}

<style>
	.panel {
		padding: 0.7rem 1rem;
		border-radius: var(--radius-card);
		background: var(--surface);
		border: 1px solid var(--border);
		color: var(--muted);
		font-size: 0.88rem;
	}
	.panel.warn {
		border-color: color-mix(in srgb, var(--amber) 45%, transparent);
		color: var(--amber);
	}
</style>
