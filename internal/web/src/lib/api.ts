import { resolve } from '$app/paths';
import type { SensorInfo, DeviceInfo, ConnectionInfo, ConnectionPage, Topology } from './types.js';

// The REST API is served by the Go backend below the base path of the web
// document. SvelteKit 3 no longer exports the base path itself, so it is
// derived from the root route.
const BASE = `${resolve('/').replace(/\/$/, '')}/api/v1`;

async function get<T>(path: string): Promise<T> {
	const res = await fetch(`${BASE}${path}`);
	if (!res.ok) throw new Error(`HTTP ${res.status}: ${path}`);
	return res.json() as Promise<T>;
}

export const api = {
	ping: () => get<string>('/ping'),
	sensors: () => get<SensorInfo[]>('/sensor'),
	lmis: () => get<string[]>('/rules/lmi'),
	device: (id: string) => get<DeviceInfo>(`/device/${id}`),
	connections: (params?: Record<string, string>) => {
		const qs = params ? '?' + new URLSearchParams(params).toString() : '';
		return get<ConnectionPage>(`/connection${qs}`);
	},
	topology: () => get<Topology>('/topology')
};
