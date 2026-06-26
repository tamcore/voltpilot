import '@testing-library/jest-dom/vitest';

// jsdom does not always expose a working localStorage (depends on origin).
// Provide a simple in-memory shim so store persistence logic is testable.
if (typeof globalThis.localStorage === 'undefined') {
	const store = new Map<string, string>();
	const shim: Storage = {
		get length() {
			return store.size;
		},
		clear: () => store.clear(),
		getItem: (k: string) => (store.has(k) ? (store.get(k) as string) : null),
		key: (i: number) => Array.from(store.keys())[i] ?? null,
		removeItem: (k: string) => void store.delete(k),
		setItem: (k: string, v: string) => void store.set(k, String(v))
	};
	Object.defineProperty(globalThis, 'localStorage', { value: shim, configurable: true });
}
