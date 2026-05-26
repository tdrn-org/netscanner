<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { m } from '$lib/i18n.js';
	import type { SensorInfo, ConnectionInfo } from '$lib/types.js';
	import { api } from '$lib/api.js';
	import { mockSensors, mockConnections } from '$lib/mocks.js';
	import StatusBadge from '$lib/components/ui/StatusBadge.svelte';
	import { Activity, Wifi, Shield, ArrowUpRight } from '@lucide/svelte';

	let sensors = $state<SensorInfo[]>([]);
	let connections = $state<ConnectionInfo[]>([]);
	let loading = $state(true);
	let error = $state(false);

	onMount(() => {
		loadData();
	});

	async function loadData() {
		loading = true;
		error = false;
		try {
			const [s, c] = await Promise.all([api.sensors(), api.connections()]);
			sensors = s;
			connections = c;
		} catch {
			sensors = mockSensors;
			connections = mockConnections;
		}
		loading = false;
	}

	let activeSensorCount = $derived(sensors.length);
	let totalConnectionCount = $derived(connections.length);
	let grantedCount = $derived(connections.filter(c => c.status === 'granted').length);
	let deniedCount = $derived(connections.filter(c => c.status === 'denied').length);
	let recentConnections = $derived(connections.slice(0, 5));

	function formatTime(ts: number): string {
		// Backend sends UnixMicro (µs), JS Date expects milliseconds
		return new Date(ts / 1000).toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' });
	}

	function shortAddr(addr: string): string {
		return addr.length > 18 ? addr.slice(0, 18) + '…' : addr;
	}
</script>

{#if loading}
	<div class="flex items-center justify-center py-20">
		<div class="flex flex-col items-center gap-3">
			<div class="h-8 w-8 animate-spin rounded-full border-2 border-indigo-400 border-t-transparent"></div>
			<span class="text-sm text-stone-400">{m.loading()}</span>
		</div>
	</div>
{:else}
	<!-- Stats Grid -->
	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
		<div class="card p-5">
			<div class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-500/10">
					<Shield class="h-5 w-5 text-indigo-400" />
				</div>
				<div>
					<p class="text-xs font-medium text-stone-400">{m.dashboard_active_sensors()}</p>
					<p class="text-2xl font-bold text-white">{activeSensorCount}</p>
				</div>
			</div>
		</div>

		<div class="card p-5">
			<div class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-cyan-500/10">
					<Wifi class="h-5 w-5 text-cyan-400" />
				</div>
				<div>
					<p class="text-xs font-medium text-stone-400">{m.dashboard_total_connections()}</p>
					<p class="text-2xl font-bold text-white">{totalConnectionCount}</p>
				</div>
			</div>
		</div>

		<div class="card p-5">
			<div class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-green-500/10">
					<ArrowUpRight class="h-5 w-5 text-green-400" />
				</div>
				<div>
					<p class="text-xs font-medium text-stone-400">{m.status_granted()}</p>
					<p class="text-2xl font-bold text-green-400">{grantedCount}</p>
				</div>
			</div>
		</div>

		<div class="card p-5">
			<div class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-red-500/10">
					<Activity class="h-5 w-5 text-red-400" />
				</div>
				<div>
					<p class="text-xs font-medium text-stone-400">{m.status_denied()}</p>
					<p class="text-2xl font-bold text-red-400">{deniedCount}</p>
				</div>
			</div>
		</div>
	</div>

	<!-- Recent Connections -->
	<div class="mt-8">
		<div class="mb-4 flex items-center justify-between">
			<h2 class="text-lg font-semibold text-white">{m.dashboard_recent_connections()}</h2>
			<a href={resolve('/connections/')} class="text-sm font-medium text-indigo-400 hover:text-indigo-300 transition-colors">Alle anzeigen →</a>
		</div>

		<div class="card overflow-hidden">
			<table class="w-full text-left text-sm">
				<thead>
					<tr class="border-b border-slate-700/50">
						<th class="px-4 py-3 font-medium text-stone-400">{m.connections_client()}</th>
						<th class="px-4 py-3 font-medium text-stone-400">{m.connections_server()}</th>
						<th class="px-4 py-3 font-medium text-stone-400">{m.connections_service()}</th>
						<th class="px-4 py-3 font-medium text-stone-400">{m.connections_status()}</th>
						<th class="px-4 py-3 font-medium text-stone-400">{m.connections_last_seen()}</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-700/30">
					{#each recentConnections as conn (conn.id)}
						<tr class="hover:bg-slate-800/50 transition-colors">
							<td class="px-4 py-3 font-mono text-xs text-stone-300">{shortAddr(conn.client.address)}</td>
							<td class="px-4 py-3 font-mono text-xs text-stone-300">{shortAddr(conn.server.address)}</td>
							<td class="px-4 py-3 text-stone-300">{conn.service}</td>
							<td class="px-4 py-3"><StatusBadge status={conn.status} /></td>
							<td class="px-4 py-3 text-xs text-stone-400">{formatTime(conn.last)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
			{#if recentConnections.length === 0}
				<div class="px-4 py-8 text-center text-sm text-stone-400">{m.connections_no_data()}</div>
			{/if}
		</div>
	</div>
{/if}
