import { describe, it, expect, vi, afterEach } from 'vitest';
import { fetchChargers, fetchCpos, ApiError } from './client';

function mockFetchOnce(status: number, body: unknown) {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => ({
			ok: status >= 200 && status < 300,
			status,
			statusText: 'x',
			json: async () => body
		}))
	);
}

afterEach(() => vi.unstubAllGlobals());

describe('api client', () => {
	it('builds the chargers query and unwraps the array', async () => {
		const f = vi.fn(async (..._args: unknown[]) => ({
			ok: true,
			status: 200,
			json: async () => ({ chargers: [{ id: '1' }] })
		}));
		vi.stubGlobal('fetch', f);
		const res = await fetchChargers({
			lat: 49.778,
			lon: 10.066,
			operatorCode: 'DEBPE',
			current: 'dc',
			availableOnly: true,
			radiusKm: 10
		});
		expect(res).toHaveLength(1);
		const url = (f.mock.calls[0][0] as string);
		expect(url).toContain('operatorCode=DEBPE');
		expect(url).toContain('current=dc');
		expect(url).toContain('availableOnly=true');
	});

	it('omits current when set to all', async () => {
		const f = vi.fn(async (..._args: unknown[]) => ({
			ok: true,
			status: 200,
			json: async () => ({ chargers: [] })
		}));
		vi.stubGlobal('fetch', f);
		await fetchChargers({ lat: 1, lon: 2, current: 'all' });
		expect((f.mock.calls[0][0] as string)).not.toContain('current=');
	});

	it('throws ApiError on non-2xx', async () => {
		mockFetchOnce(502, { error: 'upstream down' });
		await expect(fetchCpos({ lat: 1, lon: 2 })).rejects.toBeInstanceOf(ApiError);
	});
});
