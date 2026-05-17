# AGENT.md — netscanner Web Frontend (`internal/web/`)

## Projektkontext

NetScanner ist ein Netzwerk-Aktivitäts-Monitor. Das Go-Backend sammelt Verbindungsdaten
von Sensoren (Syslog, Accesslog, ...) und stellt sie via REST-API bereit.
Dieses Frontend visualisiert diese Daten in Echtzeit.

Das Frontend liegt in `internal/web/` und wird vom Go-Backend über `embed.FS` als
statisches Bundle ausgeliefert. Der Build-Output landet in `internal/web/build/`
(siehe `Makefile` → `clean`-Target: `rm -rf "internal/web/build"`).

---

## Stack

| Technologie       | Version / Hinweis                                      |
|-------------------|--------------------------------------------------------|
| SvelteKit         | **Svelte 5** – ausschließlich Runes-API                |
| Adapter           | `@sveltejs/adapter-static`, output: `build/`           |
| CSS               | **TailwindCSS v4** – kein `tailwind.config.js`         |
| Lokalisierung     | **Paraglide JS** (`@inlang/paraglide-sveltekit`)       |
| Paketmanager      | `npm` (Makefile nutzt `$(NPM) --prefix internal/web`)  |
| TypeScript        | strict mode                                            |

---

## Build-Integration mit Go

### Output-Verzeichnis

Das SvelteKit-Build muss nach **`internal/web/build/`** ausgeben:

```js
// svelte.config.js
import adapter from '@sveltejs/adapter-static';

export default {
  kit: {
    adapter: adapter({
      pages: 'build',
      assets: 'build',
      fallback: 'index.html',  // SPA-Fallback: Go leitet alle Nicht-API-Pfade hierher
      precompress: false,
    }),
  },
};
```

### Go embed.FS

Go bindet `internal/web/build/` so ein:

```go
// internal/web/web.go (bereits vorhanden – NICHT modifizieren)
//go:embed build
var staticFiles embed.FS
```

Go leitet alle Requests, die nicht mit `/api/` beginnen, auf `build/index.html` um.
Das ermöglicht clientseitiges SvelteKit-Routing.

### Build-Befehl (vom Makefile)

```bash
npm --no-progress --no-color --no-fund run --prefix internal/web build
```

Eigene Entwicklungsbefehle (aus `internal/web/`):

```bash
npm run dev    # Entwicklungsserver (Vite, HMR)
npm run build  # Produktions-Build → build/
npm run check  # svelte-check + TypeScript
```

---

## REST-API – Vollständige Referenz

Base Path: `/api/v1`
Backend-Port in Entwicklung: `9123` (konfigurierbar)

### Endpunkte

| Method | Path                   | Response-Typ           | Beschreibung                        |
|--------|------------------------|------------------------|-------------------------------------|
| GET    | `/api/v1/ping`         | `text/plain` `"ok"`    | Health-Check                        |
| GET    | `/api/v1/sensor`       | `SensorInfo[]`         | Alle laufenden Sensoren + Stats     |
| GET    | `/api/v1/rules/lmi`    | `string[]`             | Log-Matcher-Index-Namen             |
| GET    | `/api/v1/device/{id}`  | `DeviceInfo`           | Gerätedaten per ID                  |
| GET    | `/api/v1/connection`   | `ConnectionInfo[]`     | Alle geloggten Verbindungen         |

### TypeScript-Interfaces (spiegeln Go-Structs 1:1)

Diese Interfaces müssen in `$lib/types.ts` definiert sein und exakt den Go-Structs entsprechen.
Felder dürfen **nicht** umbenannt oder weggelassen werden.

```ts
// $lib/types.ts

export interface SensorInfo {
  name: string;           // Eindeutiger Name des Sensors
  type: string;           // Typ: syslog | accesslog | ...
  event_counter: number;  // Events seit Start
}

export interface DeviceInfo {
  id: string;
  address: string;
  network: string;
  hardware_address: string;
  hardware_vendor: string;
  dns: string;
  lat: number;
  lng: number;
  city: string;
  country: string;
  country_code: string;
}

export interface ConnectionInfo {
  id: string;
  server: DeviceInfo;
  client: DeviceInfo;
  service: string;
  status: 'granted' | 'denied' | 'error' | 'informational';
  count: number;
  first: number;  // Unix timestamp (ms)
  last: number;   // Unix timestamp (ms)
}
```

### API-Client (`$lib/api.ts`)

Alle API-Calls laufen zentral über `$lib/api.ts`. Keine direkten `fetch`-Aufrufe in Komponenten.

```ts
// $lib/api.ts
import type { SensorInfo, DeviceInfo, ConnectionInfo } from './types.js';

const BASE = '/api/v1';

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`);
  if (!res.ok) throw new Error(`HTTP ${res.status}: ${path}`);
  return res.json() as Promise<T>;
}

export const api = {
  ping: ()                    => get<string>('/ping'),
  sensors: ()                 => get<SensorInfo[]>('/sensor'),
  lmis: ()                    => get<string[]>('/rules/lmi'),
  device: (id: string)        => get<DeviceInfo>(`/device/${id}`),
  connections: ()             => get<ConnectionInfo[]>('/connection'),
};
```

### Mock-Daten für Entwicklung ohne Backend

```ts
// $lib/mocks.ts
import type { SensorInfo, ConnectionInfo } from './types.js';

export const mockSensors: SensorInfo[] = [
  { name: 'syslog-local', type: 'syslog', event_counter: 1024 },
  { name: 'nginx-access', type: 'accesslog', event_counter: 8731 },
];

export const mockConnections: ConnectionInfo[] = [
  {
    id: 'c001',
    server: { id: 's1', address: '192.168.1.1', network: '192.168.1.0/24',
              hardware_address: 'aa:bb:cc:dd:ee:ff', hardware_vendor: 'Cisco',
              dns: 'router.local', lat: 48.1, lng: 11.5,
              city: 'Munich', country: 'Germany', country_code: 'DE' },
    client: { id: 'c1', address: '10.0.0.5', network: '10.0.0.0/8',
              hardware_address: '', hardware_vendor: '',
              dns: '', lat: 37.7, lng: -122.4,
              city: 'San Francisco', country: 'United States', country_code: 'US' },
    service: 'HTTPS',
    status: 'granted',
    count: 42,
    first: Date.now() - 86400000,
    last: Date.now(),
  },
];
```

---

## Design-System

### Farbpalette (verbindlich)

| Token             | Tailwind-Klasse      | Beschreibung                    |
|-------------------|----------------------|---------------------------------|
| Background        | `bg-slate-950`       | Haupt-Hintergrund               |
| Cards / Nav       | `bg-slate-900`       | Karten, Navbar                  |
| Glassmorphism     | `backdrop-blur-md`   | Navbar-Effekt                   |
| Accent Primary    | `text-indigo-400`    | Primärakzente, aktive States    |
| Accent Secondary  | `text-cyan-400`      | Sekundärakzente, Highlights     |
| Text Primary      | `text-white`         | Haupttext                       |
| Text Muted        | `text-stone-300`     | Sekundärtext, Labels            |
| Status OK         | `text-green-400`     | Status: granted / ok            |
| Status Error      | `text-red-400`       | Status: denied / error          |
| Status Warn       | `text-amber-400`     | Status: informational / warn    |
| Border            | `border-slate-700`   | Trennlinien, Card-Borders       |

**Dark Mode ist obligatorisch** – das gesamte UI ist dunkel (kein Light-Mode-Toggle nötig).

### Navbar

- Glassmorphism: `bg-slate-900/80 backdrop-blur-md`
- `sticky top-0 z-50`
- Aktiver Link: `text-indigo-400`
- Inaktiver Link: `text-stone-300 hover:text-white`

### Verbindungsstatus-Badge

```svelte
<!-- $lib/components/ui/StatusBadge.svelte -->
<script lang="ts">
  import type { ConnectionInfo } from '$lib/types.js';
  let { status }: { status: ConnectionInfo['status'] } = $props();

  const classes: Record<ConnectionInfo['status'], string> = {
    granted:       'text-green-400 bg-green-400/10',
    denied:        'text-red-400 bg-red-400/10',
    error:         'text-red-400 bg-red-400/10',
    informational: 'text-amber-400 bg-amber-400/10',
  };
</script>
<span class="px-2 py-0.5 rounded text-xs font-mono {classes[status]}">{status}</span>
```

---

## Svelte 5 – Pflichtregeln

Svelte 4 Syntax ist **verboten**. Der Agent muss ausschließlich Runes verwenden.

### Props

```svelte
<!-- ✅ Korrekt -->
<script lang="ts">
  let { title, count = 0 }: { title: string; count?: number } = $props();
</script>

<!-- ❌ Falsch: export let -->
```

### State & Derived

```svelte
<script lang="ts">
  let connections = $state<ConnectionInfo[]>([]);
  let loading = $state(true);

  let grantedCount = $derived(connections.filter(c => c.status === 'granted').length);

  $effect(() => {
    api.connections().then(data => {
      connections = data;
      loading = false;
    });
  });
</script>
```

### Events: Callback-Props statt `createEventDispatcher`

```svelte
<!-- ✅ Korrekt -->
<script lang="ts">
  let { onSelect }: { onSelect: (id: string) => void } = $props();
</script>
<button onclick={() => onSelect(item.id)}>...</button>
```

### Snippets statt `<slot>`

```svelte
<script lang="ts">
  import type { Snippet } from 'svelte';
  let { children }: { children: Snippet } = $props();
</script>
<div>{@render children()}</div>
```

---

## TailwindCSS v4

**Keine `tailwind.config.js` erstellen.** Konfiguration ausschließlich in `src/app.css`:

```css
/* src/app.css */
@import "tailwindcss";

@theme {
  --color-accent-primary: theme(colors.indigo.400);
  --color-accent-secondary: theme(colors.cyan.400);
}
```

Regeln:
- Kein `<style>`-Block in Svelte-Komponenten – ausschließlich Tailwind-Klassen im Template
- `@apply` nur in `app.css` für globale Basis-Styles (Scrollbar, Body, etc.)
- Kein CSS außerhalb von `app.css`

---

## Paraglide – Lokalisierung

Alle sichtbaren Texte müssen über Paraglide-Nachrichten laufen. Keine hardcodierten Strings im Template.

### Nachrichten verwenden

```svelte
<script lang="ts">
  import * as m from '$lib/paraglide/messages.js';
</script>

<h1>{m.page_connections_title()}</h1>
<p>{m.sensor_event_count({ count: sensor.event_counter })}</p>
```

### Nachrichten-Dateien

Jede neue Nachricht muss in **beide** Dateien eingetragen werden:

`messages/de.json`:
```json
{
  "page_connections_title": "Verbindungen",
  "sensor_event_count": "{count} Events"
}
```

`messages/en.json`:
```json
{
  "page_connections_title": "Connections",
  "sensor_event_count": "{count} events"
}
```

`src/paraglide/` ist **generierter Code** – niemals manuell editieren.

---

## Projektstruktur

```
internal/web/
├── src/
│   ├── app.css                      # TailwindCSS v4 Entry (@theme hier)
│   ├── app.html
│   ├── hooks.server.ts              # Paraglide handle()
│   ├── lib/
│   │   ├── api.ts                   # Zentraler API-Client (alle fetch-Calls)
│   │   ├── types.ts                 # TypeScript-Interfaces (spiegeln Go-Structs)
│   │   ├── mocks.ts                 # Mock-Daten für Dev ohne Backend
│   │   ├── i18n.ts                  # Paraglide i18n Export
│   │   ├── paraglide/               # GENERIERT – nicht editieren
│   │   └── components/
│   │       ├── ui/                  # Generische Primitives (StatusBadge, Spinner, ...)
│   │       └── layout/              # Navbar, PageLayout
│   └── routes/
│       ├── +layout.svelte           # Root-Layout: Navbar + Paraglide
│       ├── +layout.ts               # export const ssr = false
│       ├── +page.svelte             # Dashboard / Startseite
│       ├── connections/
│       │   └── +page.svelte         # Verbindungstabelle
│       └── sensors/
│           └── +page.svelte         # Sensor-Card-Grid
├── messages/
│   ├── de.json
│   └── en.json
├── build/                           # Go embed.FS liest hier – gitignored
├── static/
├── svelte.config.js
├── vite.config.ts
├── project.inlang/
│   └── settings.json
└── package.json
```

---

## Seitenkonzept

### `/` – Dashboard

- Übersicht: Sensor-Anzahl, aktive Verbindungen, Status-Verteilung (granted/denied)
- Letzte N Verbindungen als kompakte Tabelle

### `/connections` – Verbindungstabelle

Daten von `GET /api/v1/connection`:

- Hauptspalten: Zeitstempel (`last`), Client-IP, Server-IP, Service, Status-Badge, Count
- Progressive Disclosure:
  - Hover/Tooltip: vollständige `DeviceInfo` (DNS, Vendor, Geo)
  - Klick auf Zeile: Detail-Panel (Slide-in oder Expand) mit allen Feldern
- Zeitstempel: `first`/`last` sind Unix-Millisekunden → `new Date(ts).toLocaleString()`

### `/sensors` – Sensor-Übersicht

Daten von `GET /api/v1/sensor`:

- Card-Grid (responsive: 1/2/3 Spalten)
- Pro Card: Name, Typ, Event-Counter mit Live-Refresh (Polling alle 10s via `$effect`)

---

## Wichtige Konventionen

- **Keine Monster-Komponenten** – jede Komponente hat eine einzige Verantwortung
- **Kein direktes `fetch` in Komponenten** – immer `$lib/api.ts` nutzen
- **Alle Interfaces in `$lib/types.ts`** – Go-Structs 1:1 abbilden, JSON-Feldnamen beibehalten
- **`$lib/`-Alias** – niemals relative `../../`-Imports
- **`ssr = false`** im Root-`+layout.ts` – kein SSR, reine SPA
- **`npm`** – das Makefile nutzt `npm`, kein pnpm oder yarn einführen

---

## Häufige Fehler – VERMEIDEN

| ❌ Falsch                                    | ✅ Richtig                                          |
|----------------------------------------------|-----------------------------------------------------|
| `export let foo` (Svelte 4)                  | `let { foo } = $props()`                            |
| `$: bar = x * 2`                             | `let bar = $derived(x * 2)`                         |
| `<slot />`                                   | `{@render children()}`                              |
| `createEventDispatcher()`                    | Callback-Props                                      |
| `pages: 'dist'` im Adapter                   | `pages: 'build'` (Go erwartet `internal/web/build`) |
| `tailwind.config.js` erstellen               | Konfiguration via `@theme` in `app.css`             |
| `<style>` in Svelte-Komponenten              | Ausschließlich Tailwind-Klassen im Template         |
| Hardcodierte Texte im Template               | Immer `m.message_key()` aus Paraglide               |
| `fetch('/api/v1/...')` direkt in Komponente  | `api.connections()` aus `$lib/api.ts`               |
| `src/paraglide/` editieren                   | Generierter Code – nur `messages/*.json` editieren  |

---

## Checkliste vor jedem Commit

- [ ] Kein Svelte-4-Syntax (kein `export let`, kein `$:`, kein `<slot>`)
- [ ] Alle UI-Texte über `m.key()` aus Paraglide, Schlüssel in `de.json` + `en.json`
- [ ] `svelte.config.js` → `adapter-static` mit `pages: 'build'`
- [ ] `src/routes/+layout.ts` exportiert `export const ssr = false`
- [ ] Keine direkten `fetch`-Calls in Komponenten (nur `$lib/api.ts`)
- [ ] TypeScript-Interfaces in `$lib/types.ts`, JSON-Keys unverändert (snake_case)
- [ ] Kein `<style>`-Block in Svelte-Komponenten
- [ ] `npm run check` ohne TypeScript-Fehler
- [ ] `npm run build` erstellt `internal/web/build/` erfolgreich
- [ ] `make build` (vom Repo-Root) läuft durch (Go + Frontend zusammen)
