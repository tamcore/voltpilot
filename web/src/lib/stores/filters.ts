import { writable } from 'svelte/store';
import type { CurrentFilter } from '$lib/types/api';

const STORAGE_KEY = 'voltpilot:filters';

export type Filters = {
	availableOnly: boolean;
	current: CurrentFilter;
};

const DEFAULT: Filters = { availableOnly: false, current: 'all' };

function load(): Filters {
	if (typeof localStorage === 'undefined') return { ...DEFAULT };
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return { ...DEFAULT };
		const v = JSON.parse(raw) as Partial<Filters>;
		return {
			availableOnly: Boolean(v.availableOnly),
			current: v.current === 'ac' || v.current === 'dc' ? v.current : 'all'
		};
	} catch {
		return { ...DEFAULT };
	}
}

function persist(f: Filters) {
	if (typeof localStorage === 'undefined') return;
	localStorage.setItem(STORAGE_KEY, JSON.stringify(f));
}

function createFiltersStore() {
	const inner = writable<Filters>(load());
	return {
		subscribe: inner.subscribe,
		toggleAvailable() {
			inner.update((f) => {
				const next = { ...f, availableOnly: !f.availableOnly };
				persist(next);
				return next;
			});
		},
		setCurrent(c: CurrentFilter) {
			inner.update((f) => {
				const next = { ...f, current: c };
				persist(next);
				return next;
			});
		}
	};
}

export const filters = createFiltersStore();
