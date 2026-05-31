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

	// Filtered nodes and edges
	const filteredNodes = $derived(() => {
		if (!topology) return [] as TopologyNode[];
		if (!filterNetwork && !filterStatus) return topology.nodes;

		const connectedNodes = new Set<string>();
		if (filterStatus) {
			for (const edge of topology.edges) {
				if (edge.status === filterStatus) {
					connectedNodes.add(edge.source);
					connectedNodes.add(edge.target);
				}
			}
		}

		return topology.nodes.filter(node => {
			if (filterNetwork && node.network !== filterNetwork) return false;
			if (filterStatus && !connectedNodes.has(node.id)) return false;
			return true;
		});
	});

	const filteredEdges = $derived(() => {
		if (!topology) return [] as TopologyEdge[];
		if (!filterStatus) return topology.edges;
		return topology.edges.filter(e => e.status === filterStatus);
	});

	const visible = $derived(() => {
		const n = filteredNodes;
		const e = filteredEdges;
		const ids = new Set(n.map(nd => nd.id));
		return {
			nodes: n,
			edges: e.filter(ed => ids.has(ed.source) && ids.has(ed.target))
		};
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
		{visible.nodes.length} Nodes, {visible.edges.length} Edges
	</span>
</div>

{#if loading}
	<p class="text-slate-500">Loading topology...</p>
{:else if error}
	<div class="card border-red-500/30 bg-red-500/10">
		<p class="text-red-400">{error}</p>
		<button class="btn btn-primary mt-2 text-sm" onclick={loadTopology}>Retry</button>
	</div>
{:else if topology && visible.nodes.length > 0}
	<div class="card p-2">
		<TopologyGraph nodes={visible.nodes} edges={visible.edges} />
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
