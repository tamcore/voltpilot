import { test, expect, type Page } from '@playwright/test';

// Mock the backend /api so the e2e is deterministic and offline.
async function mockApi(page: Page) {
	await page.route('**/api/cpos**', (route) =>
		route.fulfill({
			json: {
				cpos: [
					{ operatorCode: 'DEBPE', operator: 'Aral pulse', count: 4 },
					{ operatorCode: 'DEBDO', operator: 'LichtBlick', count: 1 }
				]
			}
		})
	);
	await page.route('**/api/chargers?**', (route) =>
		route.fulfill({
			json: {
				chargers: [
					{
						id: '1',
						operator: 'Aral pulse',
						operatorCode: 'DEBPE',
						lat: 49.779,
						lon: 10.067,
						distanceKm: 0.2,
						maxPowerKw: 300,
						plugTypes: ['CCS'],
						plugTypeNames: ['CCS'],
						current: 'dc',
						numberOfChargePoints: 4,
						availableChargePoints: 2,
						available: true,
						alwaysOpen: true,
						deep_links: {
							google: 'https://www.google.com/maps/dir/?api=1&destination=49.779,10.067&travelmode=driving',
							apple: 'https://maps.apple.com/?daddr=49.779,10.067&dirflg=d',
							waze: 'https://waze.com/ul?ll=49.779,10.067&navigate=yes'
						}
					}
				]
			}
		})
	);
	await page.route('**/api/chargers/1**', (route) =>
		route.fulfill({
			json: {
				id: '1',
				operator: 'Aral pulse',
				operatorCode: 'DEBPE',
				lat: 49.779,
				lon: 10.067,
				distanceKm: 0.2,
				maxPowerKw: 300,
				plugTypes: ['CCS'],
				plugTypeNames: ['CCS'],
				current: 'dc',
				numberOfChargePoints: 4,
				availableChargePoints: 2,
				available: true,
				alwaysOpen: true,
				deep_links: {
					google: 'https://www.google.com/maps/dir/?api=1&destination=49.779,10.067&travelmode=driving',
					apple: 'https://maps.apple.com/?daddr=49.779,10.067&dirflg=d',
					waze: 'https://waze.com/ul?ll=49.779,10.067&navigate=yes'
				},
				stationSummary: 'Test station',
				chargePoints: [
					{
						evseId: 'DE*BPE*E1*01',
						status: 'AVAILABLE',
						available: true,
						connectors: [{ plugTypeGroup: 'CCS', plugTypeName: 'CCS', maxPowerKw: 300, current: 'dc', cableAttached: true }]
					}
				]
			}
		})
	);
}

test('first run: pick a CPO, see chargers, open detail with nav links', async ({ page }) => {
	await mockApi(page);
	await page.addInitScript(() => localStorage.clear());
	await page.goto('/');

	// CPO chooser appears; pick Aral pulse.
	await expect(page.getByTestId('cpo-option').first()).toBeVisible();
	await page.getByTestId('cpo-option').filter({ hasText: 'Aral pulse' }).click();

	// Active CPO shown, charger list rendered.
	await expect(page.getByTestId('active-cpo')).toHaveText('Aral pulse');
	await expect(page.getByTestId('charger-card')).toBeVisible();

	// Open detail; nav links are correct.
	await page.getByTestId('charger-card').click();
	const google = page.getByRole('link', { name: 'Google Maps' });
	await expect(google).toHaveAttribute('href', /google\.com\/maps\/dir.*destination=49\.779,10\.067/);
	await expect(page.getByRole('link', { name: 'Waze' })).toHaveAttribute('href', /waze\.com\/ul/);
});

test('first run: "All operators" entry selects every CPO', async ({ page }) => {
	await mockApi(page);
	await page.addInitScript(() => localStorage.clear());
	await page.goto('/');

	await expect(page.getByTestId('cpo-all')).toBeVisible();
	await page.getByTestId('cpo-all').click();

	await expect(page.getByTestId('active-cpo')).toHaveText('All operators');
	await expect(page.getByTestId('charger-card')).toBeVisible();
});

test('returning user: remembered CPO skips the chooser', async ({ page }) => {
	await mockApi(page);
	await page.addInitScript(() =>
		localStorage.setItem('voltpilot:cpo', JSON.stringify({ operatorCode: 'DEBPE', operator: 'Aral pulse' }))
	);
	await page.goto('/');
	await expect(page.getByTestId('active-cpo')).toHaveText('Aral pulse');
	await expect(page.getByTestId('charger-card')).toBeVisible();
});

test('AC/DC + available filters are present and toggle', async ({ page }) => {
	await mockApi(page);
	await page.addInitScript(() =>
		localStorage.setItem('voltpilot:cpo', JSON.stringify({ operatorCode: 'DEBPE', operator: 'Aral pulse' }))
	);
	await page.goto('/');
	await page.getByTestId('filter-current-dc').click();
	await expect(page.getByTestId('filter-current-dc')).toHaveAttribute('aria-pressed', 'true');
	await page.getByTestId('filter-available').click();
	await expect(page.getByTestId('filter-available')).toHaveAttribute('aria-pressed', 'true');
});
