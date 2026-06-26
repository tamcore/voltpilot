import { writable } from 'svelte/store';
import type { CPO } from '$lib/types/api';

const STORAGE_KEY = 'voltpilot:cpo';

// all=true means "every operator" (no operatorCode filter); operatorCode is
// empty in that case. A null store value means no choice has been made yet.
export type PreferredCPO = { operatorCode: string; operator: string; all?: boolean } | null;

// ALL_OPERATORS is the sentinel choice shown at the top of the picker.
export const ALL_OPERATORS: PreferredCPO = { operatorCode: '', operator: 'All operators', all: true };

function load(): PreferredCPO {
	if (typeof localStorage === 'undefined') return null;
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return null;
		const v = JSON.parse(raw) as PreferredCPO;
		if (v && v.all === true) return ALL_OPERATORS;
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
		chooseAll() {
			persist(ALL_OPERATORS);
			inner.set(ALL_OPERATORS);
		},
		clear() {
			persist(null);
			inner.set(null);
		}
	};
}

export const preferredCpo = createCpoStore();
