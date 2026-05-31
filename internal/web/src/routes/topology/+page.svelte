<script lang="ts">
	import { onMount } from 'svelte';
	import type { Topology, TopologyNode, TopologyEdge } from '$lib/types';
	import { api } from '$lib/api';
	import TopologyGraph from '$lib/components/TopologyGraph.svelte';
	import { Filter } from '@lucide/svelte';

	let topology = $state<Topology | null>(null);
	let loading = $state(true);
	let error = $state('');

	// Filters
	let filterNetwork = $state('');
	let filterStatus = $state('');

	// Available networks (collected from data)
	let networks = $state<string[]>([]);

	onMount(() => loadTopology());

	async function loadTopology() {
		loading = true;
		error = '';
		try {
			topology = await api.topology();
			// Collect unique networks
			const netSet = new Set(topology.nodes.map(n => n.network).filter(Boolean));
			networks = [...netSet].sort();
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	// Filtered nodes and edges — plain state, computed via $effect
	let filteredNodes = $state<TopologyNode[]>([]);
	let filteredEdges = $state<TopologyEdge[]>([]);

	$effect(() => {
		if (!topology) {
			filteredNodes = [];
			filteredEdges = [];
			return;
		}
		// Network filter: only apply to the CLIENT side of edges.
		// Servers always stay visible so internal infrastructure remains in focus.
		// A node appears if it's a server in any visible edge, or a client in a matching edge.
		const n = (filterNetwork || filterStatus)
			? (() => {
				const connectedNodes = new Set<string>();
				if (filterStatus) {
					for (const edge of topology.edges) {
						if (edge.status === filterStatus) {
							connectedNodes.add(edge.source);
							connectedNodes.add(edge.target);
						}
					}
				}
				// When network filter is active, find which nodes to show based on edges
				const visibleNodes = new Set<string>();
				if (filterNetwork) {
					for (const edge of topology.edges) {
						const edgeOk = !filterStatus || edge.status === filterStatus;
						if (!edgeOk) continue;
						// Client must match the network filter
						const client = topology.nodes.find(nd => nd.id === edge.source);
						if (client && client.network === filterNetwork) {
							visibleNodes.add(edge.source);
							visibleNodes.add(edge.target); // server always shown
						}
					}
					// Also include nodes that are servers (type=server/both) — always visible
					for (const node of topology.nodes) {
						if (node.type === 'server' || node.type === 'both') {
							visibleNodes.add(node.id);
						}
					}
					return topology.nodes.filter(node => visibleNodes.has(node.id));
				}
				return topology.nodes.filter(node => {
					if (filterStatus && !connectedNodes.has(node.id)) return false;
					return true;
				});
			})()
			: topology.nodes;

		const e = filterStatus
			? topology.edges.filter(ed => ed.status === filterStatus)
			: topology.edges;

		filteredNodes = n;
		filteredEdges = e;
	});

	function clearFilters() {
		filterNetwork = '';
		filterStatus = '';
	}
</script>

<svelte:head>
	<title>Netscanner — Topology</title>
</svelte:head>

<div class="mb-6">
	<h1 class="text-2xl font-bold text-accent mb-1">Network Topology</h1>
	<p class="text-sm text-slate-500">Force-directed graph of recorded connections</p>
</div>

<!-- Filters -->
<div class="mb-4 flex flex-wrap items-center gap-3">
	<Filter class="h-4 w-4 text-stone-500" />
	<select bind:value={filterNetwork}
		class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs text-stone-300">
		<option value="">Alle Netzwerke</option>
		{#each networks as net}
			<option value={net}>{net}</option>
		{/each}
	</select>
	<select bind:value={filterStatus}
		class="rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-xs text-stone-300">
		<option value="">Alle Status</option>
		<option value="granted">Erlaubt</option>
		<option value="denied">Verweigert</option>
		<option value="error">Fehler</option>
		<option value="informational">Info</option>
	</select>
	{#if filterNetwork || filterStatus}
		<button onclick={clearFilters}
			class="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-1.5 text-xs text-red-400 hover:bg-red-500/20 transition-colors">
			Filter löschen
		</button>
	{/if}
	<span class="text-xs text-stone-600 ml-2">
		{filteredNodes.length} Nodes, {filteredEdges.length} Edges
	</span>
</div>

{#if loading}
	<p class="text-slate-500">Loading topology...</p>
{:else if error}
	<div class="card border-red-500/30 bg-red-500/10">
		<p class="text-red-400">{error}</p>
		<button class="btn btn-primary mt-2 text-sm" onclick={loadTopology}>Retry</button>
	</div>
{:else if topology && filteredNodes.length > 0}
	{@const visibleEdges = filteredEdges.filter(e => {
		const ids = new Set(filteredNodes.map(n => n.id));
		return ids.has(e.source) && ids.has(e.target);
	})}
	<div class="card p-2">
		<TopologyGraph nodes={filteredNodes} edges={visibleEdges} />
	</div>
	<div class="mt-4 grid grid-cols-4 gap-3 text-xs text-slate-500">
		<div class="flex items-center gap-2"><span class="w-3 h-3 rounded-full" style="background:#818cf8"></span> Server</div>
		<div class="flex items-center gap-2"><span class="w-3 h-3 rounded-full" style="background:#22d3ee"></span> Client</div>
		<div class="flex items-center gap-2"><span class="w-3 h-3 rounded-full" style="background:#c084fc"></span> Both</div>
		<div class="flex items-center gap-2"><span class="w-3 h-3 rounded-full" style="background:#22c55e"></span> Granted</div>
		<div class="flex items-center gap-2"><span class="w-3 h-3 rounded-full" style="background:#eab308"></span> Denied</div>
		<div class="flex items-center gap-2"><span class="w-3 h-3 rounded-full" style="background:#ef4444"></span> Error</div>
		<div class="flex items-center gap-2"><span class="w-3 h-3 rounded-full" style="background:#94a3b8"></span> Info</div>
	</div>
{:else}
	<div class="card text-center py-12">
		<p class="text-slate-500">No topology data available yet.</p>
		<p class="text-slate-600 text-sm mt-1">Connections will appear as the netscanner records network activity.</p>
	</div>
{/if}
