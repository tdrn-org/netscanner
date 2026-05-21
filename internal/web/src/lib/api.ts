import { base } from '$app/paths';
import type { SensorInfo, DeviceInfo, ConnectionInfo } from './types';

const BASE = `${base}/api/v1`;

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
	connections: () => get<ConnectionInfo[]>('/connection')
};
