<script lang="ts">
	import type { CPO } from '$lib/types/api';

	let {
		cpos,
		loading = false,
		onpick,
		onall
	}: { cpos: CPO[]; loading?: boolean; onpick: (cpo: CPO) => void; onall: () => void } = $props();

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
			<li>
				<button type="button" class="cpo all" onclick={() => onall()} data-testid="cpo-all">
					<span class="name">All operators</span>
					<span class="count mono">{cpos.reduce((s, c) => s + c.count, 0)}</span>
				</button>
			</li>
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
		min-height: 3rem;
		padding: 0.8rem 1rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: var(--bg-elev);
		color: var(--text);
		font: inherit;
		font-size: 1rem;
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
		min-height: 3.5rem;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
		padding: 1rem 1.15rem;
		border-radius: var(--radius-card);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--text);
		transition: border-color 0.15s, background 0.15s;
	}
	.cpo:hover,
	.cpo:focus-visible {
		border-color: var(--accent);
		outline: none;
	}
	.cpo.all {
		border-color: color-mix(in srgb, var(--accent) 55%, transparent);
		background: color-mix(in srgb, var(--accent) 10%, var(--surface));
	}
	.cpo.all .name {
		color: var(--accent);
	}
	.name {
		font-weight: 600;
		font-size: 1.05rem;
	}
	.count {
		font-size: 0.78rem;
		color: var(--muted);
		background: var(--bg-elev);
		border-radius: var(--radius-pill);
		padding: 0.1rem 0.5rem;
	}
</style>
