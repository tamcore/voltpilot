import { writable } from 'svelte/store';
import type { CPO } from '$lib/types/api';

const STORAGE_KEY = 'voltpilot:cpo';

export type PreferredCPO = { operatorCode: string; operator: string } | null;

function load(): PreferredCPO {
	if (typeof localStorage === 'undefined') return null;
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return null;
		const v = JSON.parse(raw) as PreferredCPO;
		if (v && typeof v.operatorCode === 'string' && v.operatorCode) return v;
		return null;
	} catch {
		return null;
	}
}

function persist(v: PreferredCPO) {
	if (typeof localStorage === 'undefined') return;
	if (v) localStorage.setItem(STORAGE_KEY, JSON.stringify(v));
	else localStorage.removeItem(STORAGE_KEY);
}

function createCpoStore() {
	const inner = writable<PreferredCPO>(load());
	return {
		subscribe: inner.subscribe,
		choose(cpo: CPO | PreferredCPO) {
			const v: PreferredCPO = cpo
				? { operatorCode: cpo.operatorCode, operator: cpo.operator }
				: null;
			persist(v);
			inner.set(v);
		},
		clear() {
			persist(null);
			inner.set(null);
		}
	};
}

export const preferredCpo = createCpoStore();
