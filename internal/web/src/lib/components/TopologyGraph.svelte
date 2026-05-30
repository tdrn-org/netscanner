<script lang="ts">
	import { onMount } from 'svelte';
	import type { TopologyNode, TopologyEdge } from '$lib/types';

	interface Props {
		nodes: TopologyNode[];
		edges: TopologyEdge[];
		height?: number;
	}

	let { nodes, edges, height = 500 }: Props = $props();

	// Physics state — positions and velocities
	type SimNode = { x: number; y: number; vx: number; vy: number };
	let simNodes = $state<SimNode[]>([]);
	let running = $state(true);
	let selectedNode = $state<string | null>(null);

	// Group nodes by network for background regions
	function getNetworkGroups(): Map<string, TopologyNode[]> {
		const map = new Map<string, TopologyNode[]>();
		for (const node of nodes) {
			const net = node.network || 'unknown';
			if (!map.has(net)) map.set(net, []);
			map.get(net)!.push(node);
		}
		return map;
	}

	let networkGroups = $derived(getNetworkGroups());

	const networkColors: Record<string, string> = {
		'private': 'rgba(99,102,241,0.08)',
		'public': 'rgba(239,68,68,0.05)',
		'global-unicast': 'rgba(239,68,68,0.06)',
		'loopback': 'rgba(34,211,238,0.05)',
		'unknown': 'rgba(148,163,184,0.03)'
	};

	const networkLabels: Record<string, string> = {
		'private': 'Private',
		'public': 'Internet',
		'global-unicast': 'Internet',
		'loopback': 'Local',
		'unknown': ''
	};

	const w = 900;
	const h = $derived(height);
	const cx = $derived(w / 2);
	const cy = $derived(h / 2);

	onMount(() => {
		// Initialize in a circle
		simNodes = nodes.map((_, i) => {
			const angle = (2 * Math.PI * i) / nodes.length;
			const r = Math.min(cx, cy) * 0.6;
			return { x: cx + r * Math.cos(angle), y: cy + r * Math.sin(angle), vx: 0, vy: 0 };
		});
		simulate();
		return () => { running = false; };
	});

	function simulate() {
		if (!running) return;
		const alpha = 0.3;
		const centeringForce = 0.005;
		const repulsion = 2000;
		const attraction = 0.01;
		const damping = 0.85;

		// Apply forces
		for (let i = 0; i < simNodes.length; i++) {
			const a = simNodes[i];

			// Centering
			a.vx += (cx - a.x) * centeringForce;
			a.vy += (cy - a.y) * centeringForce;

			// Repulsion from other nodes
			for (let j = i + 1; j < simNodes.length; j++) {
				const b = simNodes[j];
				let dx = b.x - a.x;
				let dy = b.y - a.y;
				const dist = Math.max(Math.sqrt(dx * dx + dy * dy), 1);
				const force = repulsion / (dist * dist);
				dx = (dx / dist) * force;
				dy = (dy / dist) * force;
				a.vx -= dx;
				a.vy -= dy;
				b.vx += dx;
				b.vy += dy;
			}

			// Attraction along edges
			for (const edge of edges) {
				const si = nodes.findIndex(n => n.id === edge.source);
				const ti = nodes.findIndex(n => n.id === edge.target);
				if (si === i || ti === i) {
					const other = si === i ? simNodes[ti] : simNodes[si];
					if (other) {
						const dx = other.x - a.x;
						const dy = other.y - a.y;
						const dist = Math.max(Math.sqrt(dx * dx + dy * dy), 1);
						a.vx += dx * attraction;
						a.vy += dy * attraction;
					}
				}
			}
		}

		// Apply velocities with damping
		for (const n of simNodes) {
			n.vx *= damping;
			n.vy *= damping;
			n.x += n.vx;
			n.y += n.vy;
			// Keep in bounds
			n.x = Math.max(30, Math.min(w - 30, n.x));
			n.y = Math.max(20, Math.min(h - 20, n.y));
		}

		requestAnimationFrame(simulate);
	}

	function nodeRadius(node: TopologyNode): number {
		return Math.max(12, Math.min(30, 10 + node.connectionCount * 3));
	}

	function statusColor(status: string): string {
		switch (status) {
			case 'granted': return '#22c55e';
			case 'denied': return '#eab308';
			case 'error': return '#ef4444';
			default: return '#94a3b8';
		}
	}

	function typeColor(type: string): string {
		switch (type) {
			case 'server': return '#818cf8';
			case 'client': return '#22d3ee';
			case 'both': return '#c084fc';
			default: return '#94a3b8';
		}
	}

	// Convert 2-letter country code to flag emoji (Unicode Regional Indicators)
	function countryFlag(code: string): string {
		if (!code || code.length !== 2) return '';
		const a = '🇦'.codePointAt(0)!;
		return String.fromCodePoint(a + code.charCodeAt(0) - 65, a + code.charCodeAt(1) - 65);
	}
</script>

<svg viewBox="0 0 {w} {h}" class="w-full h-auto rounded-lg" style="max-height:{height}px; background:#0f172a">
	<!-- Network regions -->
	{#each [...networkGroups] as [network, networkNodes] (network)}
		{@const ns = networkNodes.map(n => nodes.findIndex(x => x.id === n.id)).filter(i => i >= 0 && simNodes[i])}
		{#if ns.length >= 1}
			{@const xs = ns.map(i => simNodes[i].x)}
			{@const ys = ns.map(i => simNodes[i].y)}
			{@const minX = Math.min(...xs) - (ns.length === 1 ? 50 : 60)}
			{@const minY = Math.min(...ys) - (ns.length === 1 ? 35 : 40)}
			{@const maxX = Math.max(...xs) + (ns.length === 1 ? 50 : 60)}
			{@const maxY = Math.max(...ys) + (ns.length === 1 ? 35 : 40)}
			<rect x={minX} y={minY} width={maxX - minX} height={maxY - minY}
				rx="12" fill={networkColors[network] || networkColors['unknown']}
				stroke={network === 'private' ? 'rgba(99,102,241,0.2)' : 'rgba(148,163,184,0.1)'}
				stroke-width="1" stroke-dasharray={network === 'public' ? '4 2' : 'none'}
			/>
			<text x={minX + 8} y={minY + 16} fill="rgba(148,163,184,0.5)" font-size="9">
				{networkLabels[network] || network}
			</text>
		{/if}
	{/each}

	<!-- Edges -->
	{#each edges as edge (edge.source + edge.target + edge.service)}
		{@const si = nodes.findIndex(n => n.id === edge.source)}
		{@const ti = nodes.findIndex(n => n.id === edge.target)}
		{#if si >= 0 && ti >= 0 && simNodes[si] && simNodes[ti]}
			<line
				x1={simNodes[si].x} y1={simNodes[si].y}
				x2={simNodes[ti].x} y2={simNodes[ti].y}
				stroke={statusColor(edge.status)} stroke-opacity="0.4"
				stroke-width={Math.max(1, Math.min(3, edge.count / 5))}
			/>
		{/if}
	{/each}

	<!-- Nodes -->
	{#each nodes as node, i (node.id)}
		{@const sn = simNodes[i]}
		{#if sn}
			{@const r = nodeRadius(node)}
			<circle
				cx={sn.x} cy={sn.y} r={r}
				fill={typeColor(node.type)}
				fill-opacity="0.8" stroke={selectedNode === node.id ? '#fff' : typeColor(node.type)}
				stroke-width={selectedNode === node.id ? 2 : 1}
				class="cursor-pointer transition-all hover:fill-opacity-100"
				onclick={() => selectedNode = selectedNode === node.id ? null : node.id}
				onkeydown={(e) => { if (e.key === 'Enter') selectedNode = selectedNode === node.id ? null : node.id; }}
				role="button"
				tabindex="0"
			/>
			<text
				x={sn.x} y={sn.y + r + 14}
				text-anchor="middle" fill="#cbd5e1"
				font-size="10" class="pointer-events-none"
			>
				{countryFlag(node.countryCode || '')} {node.dns || node.label}
			</text>
		{/if}
	{/each}
</svg>

<!-- Selected node detail -->
{#if selectedNode}
	{@const node = nodes.find(n => n.id === selectedNode)}
	{#if node}
		<div class="mt-4 card p-4">
			<h3 class="text-sm font-semibold text-accent mb-2">
				{countryFlag(node.countryCode || '')} {node.dns || node.label}
			</h3>
			<div class="grid grid-cols-2 gap-2 text-xs text-slate-400">
				<div>Address: <span class="text-slate-200">{node.address}</span></div>
				<div>Type: <span class="text-slate-200">{node.type}</span></div>
				{#if node.hardwareAddress}
					<div>MAC: <span class="text-slate-200 font-mono">{node.hardwareAddress}{#if node.hardwareVendor} <span class="text-slate-500">({node.hardwareVendor})</span>{/if}</span></div>
				{/if}
				<div>Network: <span class="text-slate-200">{node.network}</span></div>
				{#if node.countryCode}<div>Location: <span class="text-slate-200">{countryFlag(node.countryCode)} {node.countryCode} ({node.lat.toFixed(1)},{node.lng.toFixed(1)})</span></div>{/if}
				<div>Connections: <span class="text-slate-200">{node.connectionCount}</span></div>
			</div>
			<!-- Connected edges -->
			<div class="mt-3 pt-3 border-t border-slate-700">
				<div class="text-xs text-slate-500 mb-1">Edges:</div>
				{#each edges.filter(e => e.source === node.id || e.target === node.id) as edge}
					{@const other = nodes.find(n => n.id === (edge.source === node.id ? edge.target : edge.source))}
					<div class="flex items-center gap-2 text-xs py-1">
						<span class="inline-block w-2 h-2 rounded-full" style="background:{statusColor(edge.status)}"></span>
						<span class="text-slate-300">{edge.service}</span>
						<span class="text-slate-500">→</span>
						<span class="text-slate-400">{other?.label ?? edge.target}</span>
						<span class="text-slate-600 ml-auto">{edge.count}x</span>
					</div>
				{/each}
			</div>
		</div>
	{/if}
{/if}
