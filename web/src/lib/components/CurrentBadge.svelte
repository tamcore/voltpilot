<script lang="ts">
	import type { Current } from '$lib/types/api';
	let { current }: { current: Current } = $props();
	const label = $derived(current === 'both' ? 'AC·DC' : current.toUpperCase());
</script>

<span class="badge" class:dc={current === 'dc'} class:both={current === 'both'}>{label}</span>

<style>
	.badge {
		font-family: var(--font-mono);
		font-weight: 700;
		font-size: 0.68rem;
		letter-spacing: 0.06em;
		padding: 0.12rem 0.42rem;
		border-radius: var(--radius-sm);
		background: color-mix(in srgb, var(--cool) 18%, transparent);
		color: var(--cool);
		border: 1px solid color-mix(in srgb, var(--cool) 40%, transparent);
	}
	.badge.dc,
	.badge.both {
		background: color-mix(in srgb, var(--accent) 16%, transparent);
		color: var(--accent);
		border-color: color-mix(in srgb, var(--accent) 45%, transparent);
	}
</style>
