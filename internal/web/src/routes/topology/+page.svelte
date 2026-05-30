<script lang="ts">
	import { onMount } from 'svelte';
	import type { Topology } from '$lib/types';
	import { api } from '$lib/api';
	import TopologyGraph from '$lib/components/TopologyGraph.svelte';

	let topology = $state<Topology | null>(null);
	let loading = $state(true);
	let error = $state('');

	onMount(() => loadTopology());

	async function loadTopology() {
		loading = true;
		error = '';
		try {
			topology = await api.topology();
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Netscanner — Topology</title>
</svelte:head>

<div class="mb-6">
	<h1 class="text-2xl font-bold text-accent mb-1">Network Topology</h1>
	<p class="text-sm text-slate-500">Force-directed graph of recorded connections</p>
</div>

{#if loading}
	<p class="text-slate-500">Loading topology...</p>
{:else if error}
	<div class="card border-red-500/30 bg-red-500/10">
		<p class="text-red-400">{error}</p>
		<button class="btn btn-primary mt-2 text-sm" onclick={loadTopology}>Retry</button>
	</div>
{:else if topology && topology.nodes.length > 0}
	<div class="card p-2">
		<TopologyGraph nodes={topology.nodes} edges={topology.edges} />
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
