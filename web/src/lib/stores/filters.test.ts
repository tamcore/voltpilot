import { describe, it, expect, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { filters } from './filters';

describe('filters store', () => {
	beforeEach(() => {
		localStorage.clear();
		// reset to defaults
		filters.setCurrent('all');
		if (get(filters).availableOnly) filters.toggleAvailable();
	});

	it('defaults to all + not available-only', () => {
		expect(get(filters)).toEqual({ availableOnly: false, current: 'all' });
	});

	it('toggles availableOnly and persists', () => {
		filters.toggleAvailable();
		expect(get(filters).availableOnly).toBe(true);
		expect(localStorage.getItem('voltpilot:filters')).toContain('"availableOnly":true');
	});

	it('sets current type', () => {
		filters.setCurrent('dc');
		expect(get(filters).current).toBe('dc');
	});
});
