<script lang="ts">
	import { filters } from '$lib/stores/filters';
	import type { CurrentFilter } from '$lib/types/api';

	const currents: { value: CurrentFilter; label: string }[] = [
		{ value: 'all', label: 'All' },
		{ value: 'ac', label: 'AC' },
		{ value: 'dc', label: 'DC' }
	];
</script>

<div class="filters">
	<button
		type="button"
		class="chip"
		class:active={$filters.availableOnly}
		aria-pressed={$filters.availableOnly}
		onclick={() => filters.toggleAvailable()}
		data-testid="filter-available"
	>
		<span class="dot"></span> Available only
	</button>

	<div class="seg" role="group" aria-label="Current type">
		{#each currents as c (c.value)}
			<button
				type="button"
				class="opt"
				class:active={$filters.current === c.value}
				aria-pressed={$filters.current === c.value}
				onclick={() => filters.setCurrent(c.value)}
				data-testid={`filter-current-${c.value}`}
			>
				{c.label}
			</button>
		{/each}
	</div>
</div>

<style>
	.filters {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		flex-wrap: wrap;
	}
	.chip {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.4rem 0.7rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--bg-elev);
		color: var(--muted);
		font-size: 0.82rem;
		font-weight: 500;
		transition: all 0.15s;
	}
	.chip .dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 999px;
		background: var(--muted-2);
	}
	.chip.active {
		color: var(--bg);
		background: var(--ok);
		border-color: var(--ok);
	}
	.chip.active .dot {
		background: var(--bg);
	}
	.seg {
		display: inline-flex;
		padding: 2px;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--bg-elev);
	}
	.opt {
		font-family: var(--font-display);
		font-weight: 600;
		font-size: 0.78rem;
		letter-spacing: 0.1em;
		color: var(--muted);
		padding: 0.3rem 0.7rem;
		border-radius: var(--radius-pill);
		transition: color 0.15s, background 0.15s;
	}
	.opt.active {
		background: var(--accent);
		color: var(--bg);
	}
</style>
