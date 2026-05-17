# Agent Guidelines — netscanner

## Projekt
- Go-Backend (Pakete: probe, dns, geoip, ouidb, sensor)
- SvelteKit 2+ Frontend unter `internal/web/`
- Backend-API: REST auf Port 8080
- Frontend wird von Go als embedded fs ausgeliefert

## Frontend-Entwicklung
- SvelteKit, TypeScript strict, TailwindCSS
- Dark-Mode Pflicht (slate-950 Hintergrund)
- Design: Glassmorphism-Navbar, indigo-400/cyan-400 Akzente
- Events-Tabelle: progressive disclosure (Hauptspalten + Tooltips + Detail-Panel)
- Sensors: Card-Grid, responsive
- Mock-Daten in `$lib/mocks.ts` für Entwicklung ohne Backend
- Keine Erklärungen zu Tailwind-Klassen in Commit-Messages — nur Ergebnisse

## Farbpalette
- Background: slate-950
- Cards/Nav: slate-900 mit glassmorphism (backdrop-blur)
- Accents: indigo-400, cyan-400
- Text: white/stone-300
- Status: green-400 (ok), red-400 (error), amber-400 (warn)

## Konventionen
- Jede Svelte-Komponente in eigener Datei, keine Monster-Komponenten
- TypeScript-Interfaces in `$lib/types.ts`, spiegeln Go-Structs
- API-Calls zentral in `$lib/api.ts`
- Kein CSS außer Tailwind-Klassen (kein `<style>` Block)