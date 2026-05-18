import type { SensorInfo, ConnectionInfo } from './types.js';

export const mockSensors: SensorInfo[] = [
	{ name: 'accesslog/accesslog1#1', type: 'accesslog', event_counter: 8472 },
	{ name: 'syslog/syslog1#1', type: 'syslog', event_counter: 3156 }
];

export const mockConnections: ConnectionInfo[] = [
	{
		id: 'c001',
		server: {
			id: 's1', address: '10.1.1.1', network: '10.1.0.0/16',
			hardware_address: 'aa:bb:cc:dd:ee:01', hardware_vendor: 'Cisco',
			dns: 'gateway.local', lat: 48.1351, lng: 11.5820,
			city: 'Munich', country: 'Germany', country_code: 'DE'
		},
		client: {
			id: 'c1', address: '192.168.1.42', network: '192.168.1.0/24',
			hardware_address: '11:22:33:44:55:66', hardware_vendor: 'Intel',
			dns: 'workstation.local', lat: 48.1351, lng: 11.5820,
			city: 'Munich', country: 'Germany', country_code: 'DE'
		},
		service: 'HTTPS',
		status: 'granted',
		count: 142,
		first: Date.now() - 86400000,
		last: Date.now()
	},
	{
		id: 'c002',
		server: {
			id: 's1', address: '10.1.1.1', network: '10.1.0.0/16',
			hardware_address: 'aa:bb:cc:dd:ee:01', hardware_vendor: 'Cisco',
			dns: 'gateway.local', lat: 48.1351, lng: 11.5820,
			city: 'Munich', country: 'Germany', country_code: 'DE'
		},
		client: {
			id: 'c2', address: '10.5.1.89', network: '10.5.0.0/16',
			hardware_address: '', hardware_vendor: '',
			dns: '', lat: 37.9430, lng: 23.7110,
			city: 'Paleo Faliro', country: 'Greece', country_code: 'GR'
		},
		service: 'SSH',
		status: 'denied',
		count: 3,
		first: Date.now() - 3600000,
		last: Date.now()
	},
	{
		id: 'c003',
		server: {
			id: 's2', address: '10.2.0.1', network: '10.2.0.0/16',
			hardware_address: 'dd:ee:ff:00:11:22', hardware_vendor: 'Netgear',
			dns: 'vpn-gw.local', lat: 48.1351, lng: 11.5820,
			city: 'Munich', country: 'Germany', country_code: 'DE'
		},
		client: {
			id: 'c3', address: '172.16.0.10', network: '172.16.0.0/12',
			hardware_address: '', hardware_vendor: '',
			dns: 'mobile.lan', lat: 0, lng: 0,
			city: '', country: '', country_code: ''
		},
		service: 'DNS',
		status: 'informational',
		count: 872,
		first: Date.now() - 7200000,
		last: Date.now() - 60000
	}
];
