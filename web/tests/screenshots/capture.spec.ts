import { test, type Page } from '@playwright/test';

// README screenshots, captured against deterministic mocked data so the result
// is stable across runs and needs no live EnBW access. Base location: Munich.
// Paths are relative to the Playwright cwd (web/); page.screenshot() creates
// the parent directory automatically.
const OUT = '../docs/screenshots';

const CPOS = {
	cpos: [
		{ operatorCode: 'DEEBW', operator: 'EnBW', count: 14 },
		{ operatorCode: 'DEBPE', operator: 'Aral pulse', count: 6 },
		{ operatorCode: 'DEMST', operator: 'Stadtwerke München', count: 9 },
		{ operatorCode: 'DEALD', operator: 'ALDI SÜD', count: 2 }
	]
};

function chargers() {
	const mk = (
		id: string,
		address: string,
		lat: number,
		lon: number,
		distanceKm: number,
		current: string,
		maxPowerKw: number,
		avail: number,
		total: number
	) => ({
		id,
		operator: 'EnBW',
		operatorCode: 'DEEBW',
		lat,
		lon,
		distanceKm,
		address,
		maxPowerKw,
		plugTypes: current === 'ac' ? ['TYPE_2'] : ['CCS', 'TYPE_2'],
		plugTypeNames: current === 'ac' ? ['Typ 2'] : ['CCS (Typ 2)', 'Typ 2'],
		current,
		numberOfChargePoints: total,
		availableChargePoints: avail,
		available: avail > 0,
		alwaysOpen: true,
		deep_links: {
			google: `https://www.google.com/maps/dir/?api=1&destination=${lat},${lon}&travelmode=driving`,
			apple: `https://maps.apple.com/?daddr=${lat},${lon}&dirflg=d`,
			waze: `https://waze.com/ul?ll=${lat},${lon}&navigate=yes`
		}
	});
	return {
		chargers: [
			mk('200001', 'Sonnenstraße 19, 80331 München, DE', 48.1372, 11.5648, 0.8, 'dc', 300, 4, 4),
			mk('200002', 'Blumenstraße 28, 80331 München, DE', 48.1331, 11.5719, 1.1, 'both', 150, 2, 6),
			mk('200003', 'Theresienhöhe 5, 80339 München, DE', 48.1349, 11.5453, 2.3, 'dc', 200, 0, 8),
			mk('200004', 'Leopoldstraße 70, 80802 München, DE', 48.1612, 11.5861, 3.0, 'ac', 22, 3, 4)
		]
	};
}

function detail() {
	const c = chargers().chargers[0];
	return {
		...c,
		stationSummary: 'Ladepark Sonnenstraße',
		chargePoints: [
			{
				evseId: 'DE*EBW*E1001*01',
				status: 'AVAILABLE',
				available: true,
				connectors: [{ plugTypeGroup: 'CCS', plugTypeName: 'CCS (Typ 2)', maxPowerKw: 300, current: 'dc', cableAttached: true }]
			},
			{
				evseId: 'DE*EBW*E1001*02',
				status: 'AVAILABLE',
				available: true,
				connectors: [{ plugTypeGroup: 'TYPE_2', plugTypeName: 'Typ 2', maxPowerKw: 22, current: 'ac', cableAttached: false }]
			}
		]
	};
}

// A small synthetic road graph (nodes interpolated between the Munich user
// position and charger 200001) so the detail-page route preview renders
// deterministically without hitting live Overpass.
function overpassFixture() {
	const from = { lat: 48.137401, lon: 11.575879 };
	const to = { lat: 48.1372, lon: 11.5648 };
	const N = 8;
	const elements: Array<Record<string, unknown>> = [];
	const ids: number[] = [];
	for (let i = 0; i <= N; i++) {
		const id = 1000 + i;
		ids.push(id);
		elements.push({
			type: 'node',
			id,
			lat: from.lat + ((to.lat - from.lat) * i) / N,
			lon: from.lon + ((to.lon - from.lon) * i) / N
		});
	}
	elements.push({ type: 'way', id: 5000, nodes: ids, tags: { highway: 'residential' } });
	return { elements };
}

async function mockApi(page: Page) {
	await page.route('**/api/cpos**', (r) => r.fulfill({ json: CPOS }));
	await page.route('**/api/chargers?**', (r) => r.fulfill({ json: chargers() }));
	await page.route('**/api/chargers/200001**', (r) => r.fulfill({ json: detail() }));
	await page.route('**/api/interpreter**', (r) => r.fulfill({ json: overpassFixture() }));
}

test('01 — CPO picker', async ({ page }) => {
	await mockApi(page);
	await page.addInitScript(() => localStorage.clear());
	await page.goto('/');
	await page.getByTestId('cpo-all').waitFor();
	await page.screenshot({ path: `${OUT}/picker.png` });
});

test('02 — charger list', async ({ page }) => {
	await mockApi(page);
	await page.addInitScript(() =>
		localStorage.setItem('voltpilot:cpo', JSON.stringify({ operatorCode: 'DEEBW', operator: 'EnBW' }))
	);
	await page.goto('/');
	await page.getByTestId('charger-card').first().waitFor();
	await page.screenshot({ path: `${OUT}/list.png` });
});

test('03 — charger detail', async ({ page }) => {
	await mockApi(page);
	await page.addInitScript(() =>
		localStorage.setItem('voltpilot:cpo', JSON.stringify({ operatorCode: 'DEEBW', operator: 'EnBW' }))
	);
	await page.goto('/charger/200001');
	await page.getByRole('link', { name: 'Google Maps' }).waitFor();
	await page.getByTestId('route-map').waitFor();
	// Let the route compute + map tiles settle for a clean shot.
	await page.waitForTimeout(2500);
	await page.screenshot({ path: `${OUT}/detail.png` });
});
