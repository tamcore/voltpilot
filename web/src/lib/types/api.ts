// Mirrors the Go backend JSON shapes (internal/chargers).

export type Current = 'ac' | 'dc' | 'both';

export type DeepLinks = {
	google: string;
	apple: string;
	waze: string;
};

export type Charger = {
	id: string;
	operator: string;
	operatorCode: string;
	lat: number;
	lon: number;
	distanceKm: number;
	address?: string;
	maxPowerKw: number;
	plugTypes: string[];
	plugTypeNames: string[];
	current: Current;
	numberOfChargePoints: number;
	availableChargePoints: number;
	available: boolean;
	alwaysOpen: boolean;
	deep_links: DeepLinks;
};

export type CPO = {
	operatorCode: string;
	operator: string;
	count: number;
};

export type Connector = {
	plugTypeGroup: string;
	plugTypeName: string;
	maxPowerKw: number;
	current: Current;
	cableAttached: boolean;
};

export type ChargePoint = {
	evseId: string;
	status: string;
	available: boolean;
	connectors: Connector[];
};

export type ChargerDetail = Charger & {
	stationSummary?: string;
	chargePoints: ChargePoint[];
};

export type ChargersResponse = { chargers: Charger[] };
export type CposResponse = { cpos: CPO[] };

export type CurrentFilter = 'all' | 'ac' | 'dc';
