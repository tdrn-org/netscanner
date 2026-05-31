<script lang="ts">
	import { onMount } from 'svelte';
	import { m } from '$lib/i18n.js';
	import type { ConnectionInfo, ConnectionPage } from '$lib/types.js';
	import { api } from '$lib/api.js';
	import { mockConnections } from '$lib/mocks.js';
	import StatusBadge from '$lib/components/ui/StatusBadge.svelte';
	import { countryFlag } from '$lib/flag.js';
	import { Wifi, MapPin, Monitor, Globe, ChevronDown, ChevronUp, Filter, ArrowUpDown } from '@lucide/svelte';

	let page = $state<ConnectionPage | null>(null);
	let connections = $state<ConnectionInfo[]>([]);
	let loading = $state(true);
	let loadingMore = $state(false);
	let expandedRow = $state<string | null>(null);
	let cursor = $state('');
	let hasMore = $state(false);
	let total = $state(0);

	// Filters
	let filterStatus = $state('');
	let filterService = $state('');
	let sortField = $state('last');
	let sortOrder = $state('desc');

	onMount(() => loadData());

	async function loadData(reset = true) {
		if (reset) {
			loading = true;
			cursor = '';
			connections = [];
		}
		const params: Record<string, string> = { limit: '50' };
		if (cursor) params.cursor = cursor;
		if (filterStatus) params.status = filterStatus;
		if (filterService) params.service = filterService;
		if (sortField) params.sort = sortField;
		if (sortOrder) params.order = sortOrder;

		try {
			page = await api.connections(params);
			if (reset) {
				connections = page.items;
			} else {
				connections = [...connections, ...page.items];
			}
			hasMore = page.has_more;
			total = page.total;
			if (page.next_cursor) cursor = page.next_cursor;
		} catch {
			if (reset) connections = mockConnections;
		}
		loading = false;
		loadingMore = false;
	}

	function applyFilters() {
		cursor = '';
		loadData(true);
	}

	function loadMore() {
		loadingMore = true;
		loadData(false);
	}

	function toggleSort(field: string) {
		if (sortField === field) {
			sortOrder = sortOrder === 'asc' ? 'desc' : 'asc';
		} else {
			sortField = field;
			sortOrder = 'desc';
		}
		applyFilters();
	}

	function sortIcon(field: string): string {
		if (sortField !== field) return '↕';
		return sortOrder === 'asc' ? '↑' : '↓';
	}

	function toggleRow(id: string) {
		expandedRow = expandedRow === id ? null : id;
	}

	function formatTime(ts: number): string {
		return new Date(ts / 1000).toLocaleString('de-DE', {
			day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit'
		});
	}
</script>

<div>
	<div class="mb-6 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-white">{m.connections_title()}</h1>
			<p class="mt-1 text-sm text-stone-400">{total} Einträge{filterStatus || filterService ? ' (gefiltert)' : ''}</p>
		</div>
		<button
			onclick={applyFilters}
			class="flex items-center gap-2 rounded-lg border border-slate-700 px-4 py-2 text-sm font-medium text-stone-300 hover:bg-slate-800 hover:text-white transition-colors"
			disabled={loading}
		>
			<Wifi class="h-4 w-4" />
			{m.refresh()}
		</button>
	</div>

	<!-- Filters -->
	<div class="mb-4 flex flex-wrap items-center gap-3">
		<Filter class="h-4 w-4 text-stone-500" />
		<select bind:value={filterStatus} onchange={applyFilters}
			class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs text-stone-300">
			<option value="">Alle Status</option>
			<option value="granted">Erlaubt</option>
			<option value="denied">Verweigert</option>
			<option value="error">Fehler</option>
			<option value="informational">Info</option>
		</select>
		<select bind:value={filterService} onchange={applyFilters}
			class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs text-stone-300">
			<option value="">Alle Dienste</option>
			<option value="http">http</option>
			<option value="sshd">sshd</option>
			<option value="dns">dns</option>
		</select>
		{#if filterStatus || filterService}
			<button onclick={() => { filterStatus = ''; filterService = ''; applyFilters(); }}
				class="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-1.5 text-xs text-red-400 hover:bg-red-500/20 transition-colors">
				Filter löschen
			</button>
		{/if}
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
						<th class="px-4 py-3 font-medium text-stone-400">
							<button onclick={() => toggleSort('client')} class="hover:text-white transition-colors">
								{m.connections_client()} {sortIcon('client')}
							</button>
						</th>
						<th class="px-4 py-3 font-medium text-stone-400">
							<button onclick={() => toggleSort('server')} class="hover:text-white transition-colors">
								{m.connections_server()} {sortIcon('server')}
							</button>
						</th>
						<th class="px-4 py-3 font-medium text-stone-400">{m.connections_service()}</th>
						<th class="px-4 py-3 font-medium text-stone-400">{m.connections_status()}</th>
						<th class="px-4 py-3 text-right font-medium text-stone-400">
							<button onclick={() => toggleSort('count')} class="hover:text-white transition-colors">
								{m.connections_count()} {sortIcon('count')}
							</button>
						</th>
						<th class="px-4 py-3 font-medium text-stone-400">
							<button onclick={() => toggleSort('last')} class="hover:text-white transition-colors">
								{m.connections_last_seen()} {sortIcon('last')}
							</button>
						</th>
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
							<td class="px-4 py-3 text-xs text-stone-300">
								{countryFlag(conn.client.country_code)} {conn.client.dns || conn.client.address}
							</td>
							<td class="px-4 py-3 text-xs text-stone-300">
								{countryFlag(conn.server.country_code)} {conn.server.dns || conn.server.address}
							</td>
							<td class="px-4 py-3 text-stone-300">{conn.service}</td>
							<td class="px-4 py-3"><StatusBadge status={conn.status} /></td>
							<td class="px-4 py-3 text-right font-mono text-xs text-cyan-400">{conn.count}</td>
							<td class="px-4 py-3 text-xs text-stone-400">{formatTime(conn.last)}</td>
						</tr>
						{#if expanded}
							<tr class="bg-slate-800/20">
								<td colspan="7" class="px-4 py-4">
									<div class="grid gap-4 text-sm sm:grid-cols-2">
										<div class="rounded-lg border border-slate-700/50 bg-slate-900/50 p-4">
											<div class="mb-2 flex items-center gap-2 text-xs font-medium text-stone-400">
												<Monitor class="h-3.5 w-3.5" />
												{countryFlag(conn.client.country_code)} Client {conn.client.dns || conn.client.address}
											</div>
											<div class="space-y-1.5">
												{#if conn.client.hardware_address}
													<p class="font-mono text-xs text-stone-300">
														<span class="text-stone-500">MAC:</span> {conn.client.hardware_address}
														{#if conn.client.hardware_vendor} <span class="text-stone-500">({conn.client.hardware_vendor})</span>{/if}
													</p>
												{/if}
												<p class="text-stone-300"><span class="text-stone-500">Network:</span> {conn.client.network}</p>
												{#if conn.client.city}
													<p class="flex items-center gap-1 text-stone-300">
														<MapPin class="h-3 w-3 text-stone-500" />
														{countryFlag(conn.client.country_code)} {conn.client.city}, {conn.client.country}
														<span class="text-xs text-stone-500">({conn.client.lat.toFixed(2)}, {conn.client.lng.toFixed(2)})</span>
													</p>
												{/if}
											</div>
										</div>
										<div class="rounded-lg border border-slate-700/50 bg-slate-900/50 p-4">
											<div class="mb-2 flex items-center gap-2 text-xs font-medium text-stone-400">
												<Globe class="h-3.5 w-3.5" />
												{countryFlag(conn.server.country_code)} Server {conn.server.dns || conn.server.address}
											</div>
											<div class="space-y-1.5">
												{#if conn.server.hardware_address}
													<p class="font-mono text-xs text-stone-300">
														<span class="text-stone-500">MAC:</span> {conn.server.hardware_address}
														{#if conn.server.hardware_vendor} <span class="text-stone-500">({conn.server.hardware_vendor})</span>{/if}
													</p>
												{/if}
												<p class="text-stone-300"><span class="text-stone-500">Network:</span> {conn.server.network}</p>
												{#if conn.server.city}
													<p class="flex items-center gap-1 text-stone-300">
														<MapPin class="h-3 w-3 text-stone-500" />
														{countryFlag(conn.server.country_code)} {conn.server.city}, {conn.server.country}
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

		<!-- Load more -->
		{#if hasMore}
			<div class="mt-4 text-center">
				<button
					onclick={loadMore}
					disabled={loadingMore}
					class="rounded-lg border border-slate-700 px-6 py-2.5 text-sm font-medium text-stone-300 hover:bg-slate-800 hover:text-white transition-colors disabled:opacity-50"
				>
					{loadingMore ? 'Lade...' : `${connections.length} von ${total} — Mehr laden`}
				</button>
			</div>
		{/if}
	{/if}
</div>
