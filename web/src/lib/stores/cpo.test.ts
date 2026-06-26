import { describe, it, expect, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { preferredCpo } from './cpo';

describe('preferredCpo store', () => {
	beforeEach(() => {
		localStorage.clear();
		preferredCpo.clear();
	});

	it('starts null', () => {
		expect(get(preferredCpo)).toBeNull();
	});

	it('remembers a chosen CPO and persists it', () => {
		preferredCpo.choose({ operatorCode: 'DEBPE', operator: 'Aral pulse', count: 4 });
		expect(get(preferredCpo)).toEqual({ operatorCode: 'DEBPE', operator: 'Aral pulse' });
		expect(localStorage.getItem('voltpilot:cpo')).toContain('DEBPE');
	});

	it('clears the chosen CPO', () => {
		preferredCpo.choose({ operatorCode: 'DEBPE', operator: 'Aral pulse', count: 4 });
		preferredCpo.clear();
		expect(get(preferredCpo)).toBeNull();
		expect(localStorage.getItem('voltpilot:cpo')).toBeNull();
	});
});
