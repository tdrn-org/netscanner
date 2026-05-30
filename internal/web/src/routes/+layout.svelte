<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import { m } from '$lib/i18n.js';
	import type { Snippet } from 'svelte';
		import { Activity, Wifi, Shield, GitBranch } from '@lucide/svelte';

let { children }: { children: Snippet } = $props();

const navItems = [
	{ href: '/', label: m.nav_dashboard(), icon: Activity },
	{ href: '/connections/', label: m.nav_connections(), icon: Wifi },
	{ href: '/topology/', label: 'Topology', icon: GitBranch },
	{ href: '/sensors/', label: m.nav_sensors(), icon: Shield }
];

	function isActive(path: string): boolean {
		const resolved = resolve(path);
		if (path === '/') return $page.url.pathname === resolved;
		return $page.url.pathname.startsWith(resolved);
	}
</script>

<div class="min-h-screen bg-slate-950">
	<!-- Navbar -->
	<nav class="glass sticky top-0 z-50">
		<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
			<div class="flex h-16 items-center justify-between">
				<!-- Logo -->
				<a href={resolve('/')} class="flex items-center gap-2.5 text-white no-underline">
					<span class="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-500/20">
						<Activity class="h-5 w-5 text-indigo-400" />
					</span>
					<span class="text-lg font-semibold tracking-tight">{m.app_name()}</span>
				</a>

				<!-- Navigation Links -->
				<div class="flex items-center gap-1">
					{#each navItems as item}
						<a
							href={resolve(item.href)}
							class="flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors {isActive(item.href)
								? 'bg-indigo-500/10 text-indigo-400'
								: 'text-stone-300 hover:bg-slate-800 hover:text-white'}"
						>
							<item.icon class="h-4 w-4" />
							<span class="hidden sm:inline">{item.label}</span>
						</a>
					{/each}
				</div>
			</div>
		</div>
	</nav>

	<!-- Page Content -->
	<main class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
		{@render children()}
	</main>
</div>
