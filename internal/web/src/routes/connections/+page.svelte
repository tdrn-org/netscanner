<script lang="ts">
	import { onMount } from 'svelte';
	import { m } from '$lib/i18n.js';
	import type { ConnectionInfo } from '$lib/types.js';
	import { api } from '$lib/api.js';
	import { mockConnections } from '$lib/mocks.js';
	import StatusBadge from '$lib/components/ui/StatusBadge.svelte';
	import { Wifi, MapPin, Monitor, Globe, ChevronDown, ChevronUp } from '@lucide/svelte';

	let connections = $state<ConnectionInfo[]>([]);
	let loading = $state(true);
	let expandedRow = $state<string | null>(null);

	onMount(() => {
		loadData();
	});

	async function loadData() {
		loading = true;
		try {
			connections = await api.connections();
		} catch {
			connections = mockConnections;
		}
		loading = false;
	}

	function toggleRow(id: string) {
		expandedRow = expandedRow === id ? null : id;
	}

	function formatTime(ts: number): string {
		return new Date(ts).toLocaleString('de-DE', {
			day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit'
		});
	}
</script>

<div>
	<div class="mb-6 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-white">{m.connections_title()}</h1>
			<p class="mt-1 text-sm text-stone-400">{connections.length} Einträge</p>
		</div>
		<button
			onclick={loadData}
			class="flex items-center gap-2 rounded-lg border border-slate-700 px-4 py-2 text-sm font-medium text-stone-300 hover:bg-slate-800 hover:text-white transition-colors"
			disabled={loading}
		>
			<Wifi class="h-4 w-4" />
			{m.refresh()}
		</button>
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-20">
			<div class="flex flex-col items-center gap-3">
				<div class="h-8 w-8 animate-spin rounded-full border-2 border-indigo-400 border-t-transparent"></div>
				<span class="text-sm text-stone-400">{m.loading()}</span>
			</div>
		</div>
	{:else if connections.length === 0}
		<div class="card flex flex-col items-center justify-center py-16">
			<Wifi class="h-10 w-10 text-slate-600" />
			<p class="mt-3 text-sm text-stone-400">{m.connections_no_data()}</p>
		</div>
	{:else}
		<div class="card overflow-hidden">
			<table class="w-full text-left text-sm">
				<thead>
					<tr class="border-b border-slate-700/50 bg-slate-900">
						<th class="w-8 px-4 py-3"></th>
						<th class="px-4 py-3 font-medium text-stone-400">{m.connections_client()}</th>
						<th class="px-4 py-3 font-medium text-stone-400">{m.connections_server()}</th>
						<th class="px-4 py-3 font-medium text-stone-400">{m.connections_service()}</th>
						<th class="px-4 py-3 font-medium text-stone-400">{m.connections_status()}</th>
						<th class="px-4 py-3 text-right font-medium text-stone-400">{m.connections_count()}</th>
						<th class="px-4 py-3 font-medium text-stone-400">{m.connections_last_seen()}</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-700/30">
					{#each connections as conn (conn.id)}
						{@const expanded = expandedRow === conn.id}
						<tr
							class="cursor-pointer hover:bg-slate-800/50 transition-colors {expanded ? 'bg-slate-800/30' : ''}"
							onclick={() => toggleRow(conn.id)}
						>
							<td class="px-4 py-3">
								{#if expanded}
									<ChevronUp class="h-4 w-4 text-indigo-400" />
								{:else}
									<ChevronDown class="h-4 w-4 text-stone-500" />
								{/if}
							</td>
							<td class="px-4 py-3 font-mono text-xs text-stone-300">{conn.client.address}</td>
							<td class="px-4 py-3 font-mono text-xs text-stone-300">{conn.server.address}</td>
							<td class="px-4 py-3 text-stone-300">{conn.service}</td>
							<td class="px-4 py-3"><StatusBadge status={conn.status} /></td>
							<td class="px-4 py-3 text-right font-mono text-xs text-cyan-400">{conn.count}</td>
							<td class="px-4 py-3 text-xs text-stone-400">{formatTime(conn.last)}</td>
						</tr>
						{#if expanded}
							<tr class="bg-slate-800/20">
								<td colspan="7" class="px-4 py-4">
									<div class="grid gap-4 text-sm sm:grid-cols-2">
										<!-- Client Details -->
										<div class="rounded-lg border border-slate-700/50 bg-slate-900/50 p-4">
											<div class="mb-2 flex items-center gap-2 text-xs font-medium text-stone-400">
												<Monitor class="h-3.5 w-3.5" />
												Client {conn.client.address}
											</div>
											<div class="space-y-1.5">
												{#if conn.client.dns}
													<p class="text-stone-300"><span class="text-stone-500">DNS:</span> {conn.client.dns}</p>
												{/if}
												{#if conn.client.hardware_vendor}
													<p class="text-stone-300"><span class="text-stone-500">Vendor:</span> {conn.client.hardware_vendor}</p>
												{/if}
												{#if conn.client.hardware_address}
													<p class="font-mono text-xs text-stone-300"><span class="text-stone-500">MAC:</span> {conn.client.hardware_address}</p>
												{/if}
												<p class="text-stone-300"><span class="text-stone-500">Network:</span> {conn.client.network}</p>
												{#if conn.client.city}
													<p class="flex items-center gap-1 text-stone-300">
														<MapPin class="h-3 w-3 text-stone-500" />
														{conn.client.city}, {conn.client.country}
														<span class="text-xs text-stone-500">({conn.client.lat.toFixed(2)}, {conn.client.lng.toFixed(2)})</span>
													</p>
												{/if}
											</div>
										</div>
										<!-- Server Details -->
										<div class="rounded-lg border border-slate-700/50 bg-slate-900/50 p-4">
											<div class="mb-2 flex items-center gap-2 text-xs font-medium text-stone-400">
												<Globe class="h-3.5 w-3.5" />
												Server {conn.server.address}
											</div>
											<div class="space-y-1.5">
												{#if conn.server.dns}
													<p class="text-stone-300"><span class="text-stone-500">DNS:</span> {conn.server.dns}</p>
												{/if}
												{#if conn.server.hardware_vendor}
													<p class="text-stone-300"><span class="text-stone-500">Vendor:</span> {conn.server.hardware_vendor}</p>
												{/if}
												{#if conn.server.hardware_address}
													<p class="font-mono text-xs text-stone-300"><span class="text-stone-500">MAC:</span> {conn.server.hardware_address}</p>
												{/if}
												<p class="text-stone-300"><span class="text-stone-500">Network:</span> {conn.server.network}</p>
												{#if conn.server.city}
													<p class="flex items-center gap-1 text-stone-300">
														<MapPin class="h-3 w-3 text-stone-500" />
														{conn.server.city}, {conn.server.country}
														<span class="text-xs text-stone-500">({conn.server.lat.toFixed(2)}, {conn.server.lng.toFixed(2)})</span>
													</p>
												{/if}
											</div>
										</div>
									</div>
									<div class="mt-3 flex gap-4 text-xs text-stone-400">
										<span>First: {formatTime(conn.first)}</span>
										<span>Count: {conn.count}</span>
									</div>
								</td>
							</tr>
						{/if}
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>
