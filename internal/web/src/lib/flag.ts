// Convert 2-letter country code to flag emoji (Unicode Regional Indicators)
export function countryFlag(code: string): string {
	if (!code || code.length !== 2) return '';
	const a = '🇦'.codePointAt(0)!;
	return String.fromCodePoint(a + code.charCodeAt(0) - 65, a + code.charCodeAt(1) - 65);
}
