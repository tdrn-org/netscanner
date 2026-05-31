// Mirror Go structs from Swagger — snake_case field names match JSON exactly.

export interface SensorInfo {
	name: string;
	type: string;
	event_counter: number;
}

export interface DeviceInfo {
	id: string;
	address: string;
	network: string;
	hardware_address: string;
	hardware_vendor: string;
	dns: string;
	lat: number;
	lng: number;
	city: string;
	country: string;
	country_code: string;
}

export interface ConnectionInfo {
	id: string;
	server: DeviceInfo;
	client: DeviceInfo;
	service: string;
	status: 'granted' | 'denied' | 'error' | 'informational';
	count: number;
	first: number;
	last: number;
}

export interface ConnectionPage {
	items: ConnectionInfo[];
	next_cursor: string;
	has_more: boolean;
	total: number;
}

export interface TopologyNode {
	id: string;
	label: string;
	address: string;
	hardwareVendor?: string;
	hardwareAddress?: string;
	dns?: string;
	network: string;
	countryCode?: string;
	lat: number;
	lng: number;
	type: 'client' | 'server' | 'both';
	connectionCount: number;
}

export interface TopologyEdge {
	source: string;
	target: string;
	service: string;
	status: string;
	count: number;
}

export interface Topology {
	nodes: TopologyNode[];
	edges: TopologyEdge[];
}
