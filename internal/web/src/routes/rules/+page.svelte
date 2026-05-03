<script>
	import { resolve } from '$app/paths';
</script>

<nav>
	<div class="flex items-center">
		<div>
			<ul class="m-2 flex flex-col gap-x-3 gap-y-1 lg:m-0 lg:flex-row lg:items-center">
				<li>
					<a href={resolve('/')}>Events</a>
				</li>
				<li>
					<a href={resolve('/sensors/')}>Sensors</a>
				</li>
				<li>
					<a href={resolve('/rules/')}>Rules</a>
				</li>
			</ul>
		</div>
	</div>
</nav>
