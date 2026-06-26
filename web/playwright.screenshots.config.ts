import { defineConfig } from '@playwright/test';

// Dedicated config for regenerating the README screenshots. Builds the static
// app, serves it with `vite preview`, and captures a few mobile-sized shots.
// The /api calls are mocked inside the spec, so no backend is needed.
const PORT = 4179;

export default defineConfig({
	testDir: 'tests/screenshots',
	timeout: 60_000,
	fullyParallel: false,
	reporter: 'list',
	use: {
		baseURL: `http://127.0.0.1:${PORT}`,
		// Mobile emulation on Chromium (already installed; no extra browser).
		browserName: 'chromium',
		viewport: { width: 390, height: 844 },
		deviceScaleFactor: 3,
		isMobile: true,
		hasTouch: true,
		permissions: ['geolocation'],
		// Munich — Marienplatz.
		geolocation: { latitude: 48.137401, longitude: 11.575879 },
		locale: 'de-DE',
		colorScheme: 'dark'
	},
	webServer: {
		command: `npm run build && npx vite preview --port ${PORT} --strictPort`,
		url: `http://127.0.0.1:${PORT}`,
		reuseExistingServer: !process.env.CI,
		timeout: 120_000
	}
});
