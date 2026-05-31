import { defineConfig } from '@playwright/test';

export default defineConfig({
	testDir: './e2e',
	retries: 0,
	use: {
		baseURL: 'http://localhost:9123',
		headless: true,
		chromiumSandbox: false,
	},
	webServer: {
		command: 'cd ../.. && go run ./cmd/netscanner/ --config ./internal/web/e2e/test-config.toml',
		url: 'http://localhost:9123/api/v1/ping',
		reuseExistingServer: false,
		timeout: 30000,
	},
});
