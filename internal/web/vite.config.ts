import adapter from '@sveltejs/adapter-static';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

// Build-time placeholder for the base path. The Go server (internal/web/web.go)
// substitutes every occurrence with the runtime base path (derived from
// server.public_url), so a single build can be hosted under any prefix.
// Must stay in sync with basePathPlaceholder in internal/web/web.go.
const BASE_PATH_PLACEHOLDER = '/__NETSCANNER_BASE_PATH__';

export default defineConfig({
	plugins: [
		sveltekit({
			compilerOptions: {
				runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true)
			},
			adapter: adapter({
				pages: 'build',
				assets: 'build',
				fallback: 'index.html',
				precompress: false,
				strict: true
			}),
			paths: {
				base: BASE_PATH_PLACEHOLDER
			}
		}),
		tailwindcss()
	]
});
