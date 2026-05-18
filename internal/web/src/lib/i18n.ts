// Lightweight i18n — reads directly from JSON message files.
// Zero dependencies, no build step needed.

import de from '../../messages/de.json';
import en from '../../messages/en.json';

type Messages = Record<string, string>;
type Locale = 'de' | 'en';

const messages: Record<Locale, Messages> = { de, en };

let currentLocale: Locale = 'de';

function detectLocale(): Locale {
	if (typeof navigator !== 'undefined') {
		const lang = navigator.language.slice(0, 2);
		if (lang === 'de') return 'de';
	}
	return 'en';
}

export function initLocale(locale?: Locale) {
	currentLocale = locale ?? detectLocale();
}

export function t(key: string, params?: Record<string, string | number>): string {
	let text = messages[currentLocale]?.[key] ?? messages.en[key] ?? key;
	if (params) {
		for (const [k, v] of Object.entries(params)) {
			text = text.replace(`{${k}}`, String(v));
		}
	}
	return text;
}

export function getLocale(): Locale {
	return currentLocale;
}

// Re-export as `m` for drop-in compatibility with Paraglide-style imports
export const m = new Proxy({} as Record<string, () => string>, {
	get(_target, key: string) {
		return () => t(key);
	}
});

// Auto-init on import
initLocale();
