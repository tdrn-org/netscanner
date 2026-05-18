<script lang="ts">
	import type { ConnectionInfo } from '$lib/types.js';
	import { m } from '$lib/i18n.js';

	let { status }: { status: ConnectionInfo['status'] } = $props();

	const config: Record<ConnectionInfo['status'], { cls: string; label: () => string }> = {
		granted:       { cls: 'badge text-green-400 bg-green-400/10 border border-green-400/20', label: () => m.status_granted() },
		denied:        { cls: 'badge text-red-400 bg-red-400/10 border border-red-400/20', label: () => m.status_denied() },
		error:         { cls: 'badge text-red-400 bg-red-400/10 border border-red-400/20', label: () => m.status_error() },
		informational: { cls: 'badge text-amber-400 bg-amber-400/10 border border-amber-400/20', label: () => m.status_informational() }
	};

	let current = $derived(config[status]);
</script>

<span class={current.cls}>{current.label()}</span>
