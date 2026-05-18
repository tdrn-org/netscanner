<script lang="ts">
	import { onMount } from 'svelte';
	import { m } from '$lib/i18n.js';
	import { api } from '$lib/api.js';
	import { FileCode, ListChecks } from '@lucide/svelte';

	let rules = $state<string[]>([]);
	let loading = $state(true);

	onMount(() => {
		loadData();
	});

	async function loadData() {
		loading = true;
		try {
			rules = await api.lmis();
		} catch {
			rules = [];
		}
		loading = false;
	}
</script>

<div>
	<div class="mb-6">
		<h1 class="text-2xl font-bold text-white">{m.rules_title()}</h1>
		<p class="mt-1 text-sm text-stone-400">{rules.length} Indizes verfügbar</p>
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-20">
			<div class="flex flex-col items-center gap-3">
				<div class="h-8 w-8 animate-spin rounded-full border-2 border-indigo-400 border-t-transparent"></div>
				<span class="text-sm text-stone-400">{m.loading()}</span>
			</div>
		</div>
	{:else if rules.length === 0}
		<div class="card flex flex-col items-center justify-center py-16">
			<ListChecks class="h-10 w-10 text-slate-600" />
			<p class="mt-3 text-sm text-stone-400">{m.rules_no_data()}</p>
		</div>
	{:else}
		<div class="grid gap-3">
			{#each rules as rule (rule)}
				<div class="card flex items-center gap-3 p-4 hover:border-slate-600/50 transition-colors">
					<div class="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-500/10">
						<FileCode class="h-4 w-4 text-indigo-400" />
					</div>
					<span class="font-mono text-sm text-stone-300">{rule}</span>
				</div>
			{/each}
		</div>
	{/if}
</div>
