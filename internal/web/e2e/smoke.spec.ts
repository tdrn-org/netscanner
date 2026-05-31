import { test, expect } from '@playwright/test';

const pages = [
	{ path: '/', name: 'Dashboard' },
	{ path: '/connections', name: 'Connections' },
	{ path: '/topology', name: 'Topology' },
	{ path: '/sensors', name: 'Sensors' },
];

for (const page of pages) {
	test(`${page.name} loads without console errors`, async ({ page: p }) => {
		const errors: string[] = [];
		p.on('pageerror', err => errors.push(err.message));

		await p.goto(page.path, { waitUntil: 'networkidle' });

		// Verify page is not empty
		await expect(p.locator('nav')).toBeVisible({ timeout: 5000 });

		// Filter expected noise
		const realErrors = errors.filter(e =>
			!e.includes('favicon') && !e.includes('404') && !e.includes('third-party')
		);
		expect(realErrors, `${page.name}: ${realErrors.join('; ')}`).toEqual([]);
	});
}

test('Topology has SVG graph with nodes', async ({ page: p }) => {
	await p.goto('/topology', { waitUntil: 'networkidle' });
	await p.waitForTimeout(4000);
	await expect(p.locator('svg circle').first()).toBeVisible({ timeout: 5000 });
});

test('Connections shows data or empty state', async ({ page: p }) => {
	await p.goto('/connections', { waitUntil: 'networkidle' });
	// Either a table or "No connections" message
	await expect(p.locator('table, .card').first()).toBeVisible({ timeout: 8000 });
});
