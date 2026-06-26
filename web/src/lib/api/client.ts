import type {
	Charger,
	ChargerDetail,
	ChargersResponse,
	CPO,
	CposResponse,
	CurrentFilter
} from '$lib/types/api';

export class ApiError extends Error {
	status: number;
	constructor(status: number, message: string) {
		super(message);
		this.status = status;
		this.name = 'ApiError';
	}
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(path, { ...init, headers: { Accept: 'application/json', ...init?.headers } });
	if (!res.ok) {
		let msg = res.statusText;
		try {
			const body = (await res.json()) as { error?: string };
			if (body?.error) msg = body.error;
		} catch {
			// non-JSON error body; keep statusText
		}
		throw new ApiError(res.status, msg);
	}
	return (await res.json()) as T;
}

export type ChargersParams = {
	lat: number;
	lon: number;
	radiusKm?: number;
	operatorCode?: string;
	current?: CurrentFilter;
	availableOnly?: boolean;
	limit?: number;
	signal?: AbortSignal;
};

export async function fetchChargers(p: ChargersParams): Promise<Charger[]> {
	const q = new URLSearchParams({ lat: String(p.lat), lon: String(p.lon) });
	if (p.radiusKm) q.set('radiusKm', String(p.radiusKm));
	if (p.operatorCode) q.set('operatorCode', p.operatorCode);
	if (p.current && p.current !== 'all') q.set('current', p.current);
	if (p.availableOnly) q.set('availableOnly', 'true');
	if (p.limit) q.set('limit', String(p.limit));
	const res = await request<ChargersResponse>(`/api/chargers?${q.toString()}`, { signal: p.signal });
	return res.chargers ?? [];
}

export async function fetchCpos(
	pos: { lat: number; lon: number },
	radiusKm?: number,
	signal?: AbortSignal
): Promise<CPO[]> {
	const q = new URLSearchParams({ lat: String(pos.lat), lon: String(pos.lon) });
	if (radiusKm) q.set('radiusKm', String(radiusKm));
	const res = await request<CposResponse>(`/api/cpos?${q.toString()}`, { signal });
	return res.cpos ?? [];
}

export function fetchChargerDetail(
	id: string,
	pos: { lat: number; lon: number },
	signal?: AbortSignal
): Promise<ChargerDetail> {
	const q = new URLSearchParams({ lat: String(pos.lat), lon: String(pos.lon) });
	return request<ChargerDetail>(`/api/chargers/${encodeURIComponent(id)}?${q.toString()}`, { signal });
}
