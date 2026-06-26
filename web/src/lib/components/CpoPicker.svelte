<script lang="ts">
	import type { CPO } from '$lib/types/api';

	let {
		cpos,
		loading = false,
		onpick
	}: { cpos: CPO[]; loading?: boolean; onpick: (cpo: CPO) => void } = $props();

	let query = $state('');
	const filtered = $derived(
		query.trim()
			? cpos.filter((c) => c.operator.toLowerCase().includes(query.trim().toLowerCase()))
			: cpos
	);
</script>

<div class="picker">
	<p class="lead">Choose your charge point operator. We'll remember it and take you straight to its nearest available charger next time.</p>

	<input
		class="search"
		type="search"
		placeholder="Filter operators…"
		bind:value={query}
		aria-label="Filter operators"
	/>

	{#if loading}
		<p class="hint">Looking for operators near you…</p>
	{:else if cpos.length === 0}
		<p class="hint">No operators found nearby. Try moving the map or check your connection.</p>
	{:else}
		<ul class="list">
			{#each filtered as c (c.operatorCode)}
				<li>
					<button type="button" class="cpo" onclick={() => onpick(c)} data-testid="cpo-option">
						<span class="name">{c.operator}</span>
						<span class="count mono">{c.count}</span>
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<style>
	.picker {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}
	.lead {
		color: var(--muted);
		font-size: 0.9rem;
		margin: 0.25rem 0 0;
	}
	.search {
		width: 100%;
		padding: 0.65rem 0.85rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--bg-elev);
		color: var(--text);
		font: inherit;
	}
	.search:focus-visible {
		outline: 2px solid var(--accent);
		outline-offset: 1px;
	}
	.hint {
		color: var(--muted-2);
		font-size: 0.85rem;
	}
	.list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}
	.cpo {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
		padding: 0.75rem 1rem;
		border-radius: var(--radius-card);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--text);
		transition: border-color 0.15s, background 0.15s;
	}
	.cpo:hover {
		border-color: var(--accent);
	}
	.name {
		font-weight: 600;
	}
	.count {
		font-size: 0.78rem;
		color: var(--muted);
		background: var(--bg-elev);
		border-radius: var(--radius-pill);
		padding: 0.1rem 0.5rem;
	}
</style>
