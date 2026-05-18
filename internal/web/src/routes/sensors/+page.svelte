<script lang="ts">
	import { onMount } from 'svelte';
	import { m } from '$lib/i18n.js';
	import type { SensorInfo } from '$lib/types.js';
	import { api } from '$lib/api.js';
	import { mockSensors } from '$lib/mocks.js';
	import { Shield, RefreshCw, FileText, Server } from '@lucide/svelte';

	let sensors = $state<SensorInfo[]>([]);
	let loading = $state(true);
	let pollInterval: ReturnType<typeof setInterval>;

	onMount(() => {
		loadData();
		pollInterval = setInterval(loadData, 10000);
		return () => clearInterval(pollInterval);
	});

	async function loadData() {
		try {
			sensors = await api.sensors();
		} catch {
			sensors = mockSensors;
		}
		loading = false;
	}

	function sensorIcon(type: string) {
		if (type === 'accesslog') return FileText;
		if (type === 'syslog') return Server;
		return Shield;
	}

	function sensorColor(type: string): string {
		if (type === 'accesslog') return 'text-cyan-400 bg-cyan-500/10';
		if (type === 'syslog') return 'text-indigo-400 bg-indigo-500/10';
		return 'text-emerald-400 bg-emerald-500/10';
	}
</script>

<div>
	<div class="mb-6 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-white">{m.sensors_title()}</h1>
			<p class="mt-1 text-sm text-stone-400">
				{sensors.length} {sensors.length === 1 ? 'Sensor' : 'Sensoren'} aktiv
				<span class="ml-2 inline-flex items-center gap-1 rounded-full bg-green-400/10 px-2 py-0.5 text-xs text-green-400">
					<span class="h-1.5 w-1.5 rounded-full bg-green-400 animate-pulse"></span>
					{m.live()}
				</span>
			</p>
		</div>
		<button
			onclick={loadData}
			class="flex items-center gap-2 rounded-lg border border-slate-700 px-4 py-2 text-sm font-medium text-stone-300 hover:bg-slate-800 hover:text-white transition-colors"
		>
			<RefreshCw class="h-4 w-4" />
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
	{:else if sensors.length === 0}
		<div class="card flex flex-col items-center justify-center py-16">
			<Shield class="h-10 w-10 text-slate-600" />
			<p class="mt-3 text-sm text-stone-400">{m.sensors_no_data()}</p>
		</div>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each sensors as sensor (sensor.name)}
				{@const Icon = sensorIcon(sensor.type)}
				{@const color = sensorColor(sensor.type)}
				<div class="card p-5 hover:border-slate-600/50 transition-colors">
					<div class="flex items-start justify-between">
						<div class="flex h-12 w-12 items-center justify-center rounded-lg {color}">
							<Icon class="h-6 w-6" />
						</div>
					</div>
					<div class="mt-4">
						<h3 class="text-sm font-semibold text-white">{sensor.name}</h3>
						<p class="mt-1 flex items-center gap-2 text-xs text-stone-400">
							<span>{m.sensors_type()}:</span>
							<span class="rounded bg-slate-800 px-1.5 py-0.5 font-mono text-stone-300">{sensor.type}</span>
						</p>
					</div>
					<div class="mt-4 flex items-baseline justify-between border-t border-slate-700/50 pt-4">
						<span class="text-xs text-stone-400">{m.sensors_events()}</span>
						<span class="text-2xl font-bold text-white tabular-nums">{sensor.event_counter.toLocaleString('de-DE')}</span>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
