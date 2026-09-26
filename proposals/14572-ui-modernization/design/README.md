# Handoff: Kubeflow Pipelines UI modernization

## Overview
A full visual and flow redesign of the KFP frontend (`kubeflow/pipelines/frontend`, master as of 2026-09-26). It replaces the MUI v5 / Emotion / typestyle presentation layer with **shadcn/ui on Base UI + Tailwind v4**, adds light/dark themes, a collapsible text+icon nav rail, a ⌘K command palette, a single-page New Run form, a side-panel task inspector on run detail, and a proper Compare view. Scope covers every screen in the current nav: Pipelines, Experiments, Runs, Run detail, Compare, Recurring runs, Artifacts, New run. Executions is intentionally excluded (master already redirects it to Runs).

Migration plan: one coordinated release cutover with verified development stages. See the [KEP](../README.md) for the governing test, compatibility, migration, and rollback criteria.

## About the design files
Files in `prototype/` are **design references built in HTML**, not production code. Recreate them in the KFP frontend using its existing stack (React 19, Vite, TanStack Query, React Router v8, `@xyflow/react`, Vitest + Testing Library) plus shadcn/ui (Base UI variant) and Tailwind v4. Keep all data hooks, `lib/`, generated API clients (`apisv2beta1`, `generated/`), and graph layout logic; replace only presentation.

Open [the prototype](prototype/KFP%20Modern.dc.html) in a browser using the preview instructions below. Its main flows are clickable; mock actions and data are illustrative. Tweaks: `navStyle` = `rail` (target) or `current` (legacy comparison), `theme` = `light`/`dark`.

## Fidelity
**High-fidelity.** Colors, type, spacing, radii and pictured states define the visual target in [tokens.css](tokens.css). Apply the KEP accessibility and compatibility criteria when implementing them. Mock data (runs, pipelines, params, logs) is illustrative only; wire to real APIs. Existing behavior supplies omitted flows and states, as specified in the KEP.

## Global layout
- App: full-viewport flex row. Nav rail (left) + `main` (flex:1, column, `overflow:hidden`; each page scrolls its own content area).
- Page header pattern: `padding: 28px 32px 0` (lists) or `20px 32px 0` (detail pages with breadcrumb). H1 22px / 600 / letter-spacing -0.02em. Subtitle 14px `--muted-foreground`, margin-top 2px. Primary action top-right.
- Content area: `padding: 0 32px 32px`, scrolls. Tables live in a card: `--card` bg, 1px `--border`, radius 12px, `overflow:hidden`; horizontal scroll via `min-width` (720–900px) on the card.

## Screens

### Nav rail (`SideNav.tsx`, replaces entirely)
- Width 236px expanded / 64px collapsed, `transition: width .2s ease`. Bg `--card`, right border `--border`. Collapsed state persists (LocalStorage, as today).
- Header 56px: 28×28 logo tile (radius 8, bg `--primary`, Kubeflow mark from `icons/kubeflowLogo.tsx` filled `--primary-foreground`, 18px) + "Pipelines" 14/600 + namespace 11px muted beneath.
- Search button: margin `4px 10px 10px`, height 34, radius 8, 1px border, bg `--muted`, muted text "Search" 13px + `⌘K` kbd (JetBrains Mono 11px, 1px `--border-strong`, radius 4). Opens palette.
- Items: height 36, padding 0 9px, radius 8, gap 12, 500 weight. 18px **stroke icons (1.6px)**, label, right-aligned count in mono 11px muted. Active: bg `--primary-soft`, fg `--primary`. Inactive: transparent, `--foreground-2`. Hover: `--muted`. Items: Pipelines, Experiments, Runs, Recurring runs, Artifacts. Runs is active for Run detail, Compare and New run too (mirror `SideNav.tsx` `_highlightRunsButton` logic).
- Footer (border-top): theme toggle ("Dark mode"/"Light mode"), Collapse (chevron rotates 45° ↔ -135°), version line "v{X} · Docs · GitHub" 11px muted.
- Collapsed: icons only, `title` tooltips, no counts.

### Runs (`AllRunsList.tsx`, `AllRunsAndArchive.tsx`, `CustomTable.tsx`) — screenshot 01, 10
- Header: "Runs" + summary "{n} running · {n} failed in the last 48 h". Actions: **Compare** (secondary; shows "Compare (n)", turns `--primary-soft` bg / `--primary` border+text when ≥2 selected; with <2 selected shows toast "Select at least two runs to compare") and **+ New run** (primary, height 34, radius 8, 600).
- Filter row (`padding:18px 32px 12px`): status chips All/Running/Succeeded/Failed/Pending/Canceled — pill, height 30, 13px/500, 7px status dot, mono count 11px at 70% opacity; selected = `--muted` bg + `--border-strong`. Right: filter input 280×32, radius 8, focus border `--primary`.
- Table columns: `28px | minmax(220px,2fr) | 110px | minmax(160px,1.4fr) | minmax(140px,1.2fr) | 90px | 150px`, gap 16, row padding 12px 16px. Header 12px/600 uppercase, letter-spacing .04em, muted.
  - Checkbox: 18×18, radius 5, 1.5px `--border-strong`; checked = `--primary` fill + white check. Click doesn't open row. Selected row bg `--primary-soft`.
  - Run cell: **history sparkline** (last 5 runs of same pipeline: 4×14px bars, radius 2, gap 2, status color; pending at 40%) + name (500, ellipsis) + run id (mono 12 muted).
  - Status: 8px dot + label 13/500 in status color; Running dot pulses.
  - Pipeline "name · vN" (version muted), Experiment, Duration (mono 13), Started (relative, 13 muted).
  - Row hover `--muted`; whole row opens run detail.
- **Empty state** (filters yield nothing): centered, padding 48px, "No runs match" 15/600, "Try another status or clear the filter." 13 muted, "Clear filters" secondary button.
- Archived tab: keep as a secondary chip/tab; not redesigned beyond sharing this table.

### Run detail (`RunDetailsV2.tsx`, `components/graph/*`, `components/tabs/*`) — screenshots 02, 03, 11
- Breadcrumb 13px muted: Runs / {experiment}. Title row: 12px status dot (pulses if running) + H1 (ellipsis) + status pill (radius full, 12/600, soft bg + status fg). Actions right: Clone, Retry (enabled only when Failed), Terminate (red text only when Running), Archive — secondary buttons height 32.
- Meta strip: border-top + border-bottom, padding 12px 0, gap 28, wraps. Each: label 11/600 uppercase .05em muted, value 14/500. Fields: Pipeline (link) + version, Started, Duration (mono), Tasks (derived: "5 done · 1 failed · 1 skipped"), Run ID (mono 13).
- **Failure banner** (Failed runs only): margin-top 14, padding 12px 14px, radius 10, bg `--status-failed-soft`, 1px `--status-failed` border. 18px red circle "!" + "Task **{task}** failed · exit code N" 13/600 + first error line mono 13 ellipsis. Right: "View logs" solid red button → selects failed node, opens Logs tab.
- Tabs: Graph / Details / Pipeline spec — height 36, 2px bottom border `--primary` when active, active text `--foreground`, inactive muted.
- **Graph**: canvas bg `--background` with dot grid (`radial-gradient(var(--border-strong) 1px, transparent 1px)` 22px). Use existing React Flow; restyle `ExecutionNode`: 200×56, radius 10, 1.5px `--border`, bg `--card`, `--shadow-card`; 10px status dot + name 13/500 + duration mono 11 muted. Selected: border `--primary` + `0 0 0 3px var(--primary-soft)`. Edges: bezier 1.5px; completed `--status-succeeded` @60%; into failed/skipped `--status-failed` @60%; into running dashed 6/6 animated; pending `--border-strong`. Legend bottom-left in a small card.
- **Task side panel** (replaces current bottom/side drawer): 380px right, border-left, bg `--card`. Header: "TASK" label + name 16/600 + close ×. Tabs Info / Logs / Events (height 32, 13px).
  - Info: 2-col grid (110px labels) Status, Duration, Component, Image, Pod (mono, break-all). "Input parameters" list (k/v rows, mono values). "Output artifacts" as clickable rows (8px accent square, name 13/500, type mono 11).
  - Logs: mono 12/1.6, pre-wrap. Keep `LogViewer.tsx` virtualization.
  - Events: time (mono muted) + message rows.
  - Esc closes panel.
- Details tab: key/value card max-width 720, 160px label column. Spec tab: YAML in card, mono 12.5/1.6 (keep Ace/`Editor.tsx` read-only if preferred, themed).

### Compare (`CompareV2.tsx`) — screenshot 04
- Breadcrumb Runs / Compare; H1 "Compare {n} runs"; subtitle; "Change selection" secondary → back to Runs keeping selection.
- One card. Grid `minmax(160px,1fr) repeat(n, minmax(180px,1.4fr))`. Header row: per run status (12/500 + dot), name 600 ellipsis, "id · duration" mono 12; clicking opens run.
- Sections with header rows (11/600 uppercase, bg `--background`): **Parameters**, **Metrics**, **Run** (Pipeline, Experiment, Duration, Started). Row label mono 13 with 6px dot (accent=param, green=metric, neutral=run). Rows where values differ: bg `--status-warning-soft`, values 600 `--foreground`. Missing values "—".
- Keep existing metric visualizations (ROC, confusion matrix, scalar tables) below this table as cards styled per tokens.

### Pipelines (`PipelineList.tsx`, `PipelineDetails.tsx`) — screenshot 05
- Header + "+ Upload pipeline". Card grid `repeat(auto-fill, minmax(300px,1fr))`, gap 14. Card: padding 16/18, radius 12; name 600 ellipsis, description 12 muted; version tag (mono 11, `--muted` bg, radius 5). **Run-health bars**: last 14 runs, flex bars height 26, success 100% height green @55%, failure 40% height red. Footer: "{n} runs · {pct} success" / updated. Hover: `--border-strong` + shadow. Click → pipeline detail (keep current graph + versions; restyle per Run detail).

### Experiments (`ExperimentList.tsx`, `ExperimentDetails.tsx`) — screenshot 06
- Table rows: name 500 | success/failure stacked bar (6px, radius 3, green then red over `--status-neutral-soft`) | "{n} runs" | last run relative. Row → experiment runs (reuse Runs table filtered).

### Recurring runs (`AllRecurringRunsList.tsx`, `RecurringRunDetailsV2*`, `lib/TriggerUtils.ts`) — screenshot 07
- Columns: Schedule (name + "{experiment} · max {n} concurrent") | Enabled (switch 36×20, green when on, calls enable/disable API inline) | Trigger (human text + cron mono 12) | Pipeline + version | Next run | Status (Active/Scheduled/Expired/Disabled dot+label). "+ New schedule" opens New run in recurring mode.

### Artifacts (`ArtifactList.tsx`, `ArtifactDetails.tsx`) — screenshot 08
- Type chips (All/Model/Dataset/Metrics/HTML + counts). Columns: artifact (10px type-colored square: Model=primary, Dataset=green, Metrics=warning, HTML=neutral; name + "id N" mono) | type tag | URI (mono 12.5 ellipsis) | produced-by task · run | created. Row → run detail with that task selected. Keep `ArtifactPreview` / `NativeArtifactLineage` on the details page, restyled.

### New run (`NewRunV2.tsx`, `NewRunSwitcher.tsx`, `NewRunParametersV2.tsx`, `PipelinesDialogV2.tsx`) — screenshot 09
- **One page, no dialogs.** Grid `minmax(0,1fr) 300px`, gap 24, max-width 1040. Left: four numbered cards (22px accent circle + title 600):
  1. **Pipeline** — selectable cards grid (`minmax(200px,1fr)`), selected = 1.5px `--primary` + `--primary-soft`. Version pills (mono) + "Always use latest" checkbox. Replaces `PipelinesDialogV2`.
  2. **Run details** — Run name (auto "Run of {pipeline} {version} ({5-char id})"), Experiment select, Description (optional), Service account (mono), Pipeline root (mono). Inputs height 34, radius 8, bg `--background`.
  3. **Schedule** — segmented Run once / Recurring (3px padded track, active segment `--card` + subtle shadow). Recurring reveals Trigger (Cron / Periodic), cron expression, Max concurrent runs, Catch up checkbox, and a plain-English preview "Every day at 22:00 UTC · next run …".
  4. **Parameters** — rows: name (mono 13) | type (mono 11 muted, e.g. NUMBER_INTEGER) | input. Keep `NewRunParametersV2` validation.
- Right: sticky Summary card (Pipeline, Experiment, Mode, Params) + primary "Start run"/"Create schedule" + Cancel.
- On submit: navigate to Runs (or Recurring runs) and show toast "Started "{name}"" with "View run" action.

### Command palette (new) — screenshot 12
- ⌘K / Ctrl+K toggles from anywhere; Esc closes. Overlay `rgba(10,12,20,.45)`, panel 560px, top 14vh, radius 14, shadow. Input 52px 15px. Results max-height 360: rows 40px — kind label (11/600 uppercase muted, 76px) | label | hint. Sources: runs, pipelines, experiments (server search via existing list APIs with filter, debounced), actions (Create new run, Toggle theme). Use shadcn `Command`.

### Toasts (new)
- Bottom-center, 24px from bottom, radius 10, bg `--foreground`, text `--background`, 13px; green 8px dot, message, optional action (accent 600), ×. Auto-dismiss 6s. Use shadcn `Sonner`.

## Interactions & behavior
- Running indicators pulse (`kfp-pulse` 1.6s); running edges animate (`kfp-dash` 1s linear).
- Nav collapse 200ms width; chevron rotate.
- Hover: rows/buttons `--muted`; cards `--border-strong` + shadow.
- Focus: inputs border `--primary`; add visible `ring-2 ring-[--primary-soft]` on all interactive elements (not in prototype; required).
- Loading: skeleton rows (same row heights) for tables, skeleton nodes for graph. Errors from APIs: reuse `Banner.tsx` semantics styled as the failure banner (warning variant for non-fatal).
- Responsive: below 1024px the rail auto-collapses; below 900px the task panel becomes a full-height overlay sheet; tables scroll horizontally.

## State
- Theme: `light`/`dark`/`system`, default `system`, stored in LocalStorage, applied as `.dark` on `<html>`. Configure the Tailwind v4 class-driven dark variant; follow system changes when the stored preference is `system`.
- Nav collapsed: LocalStorage (existing key).
- Runs: status filter + text filter → existing list filter API (`apisv2beta1/filter`); selected run ids (for Compare) in URL `?runlist=` as today.
- Run detail: preserve existing `?task=` selected-node and `?view=` semantics. The mock tab labels are visual guidance, not a replacement URL contract; panel tab can remain local.
- New run: form state local; recurring mode via `?recurring=1`.
- All data via existing TanStack Query hooks (`hooks/queryKeys.ts`).

## Design tokens
See `tokens.css` (light + dark). Summary:
- Accent `#2563d9` (dark `#6b9cf5`). Surfaces `#f5f6f9` / `#ffffff` / `#f0f1f5`. Border `#e3e5ec` / `#d3d6df`. Text `#15171e` / `#4a4f5c` / `#737889`.
- Status: succeeded `#1f8a4c`, running `#2563d9`, failed `#c9302c`, warning `#b8700a`, neutral `#8a8f9d` (each with a soft bg).
- Radii 5 / 8 / 10 / 12 / 14 (palette) / full. Spacing: 4-pt base; page gutters 32; card padding 16–20; table row 12×16.
- Type: Public Sans 400/500/600 (OFL, bundle locally); JetBrains Mono 400/500 for ids, durations, params, URIs, logs. Scale: 11 (labels, uppercase .05em), 12, 13, 14 (body), 15, 16, 22 (H1, -0.02em).

## Assets
- Kubeflow mark: paths from `frontend/src/icons/kubeflowLogo.tsx`.
- Nav icons: 18px, 1.6px stroke, drawn in the prototype (`icon()` in the logic class). Use Lucide equivalents: `Workflow` (Pipelines), `FlaskConical` (Experiments), `PlayCircle` (Runs), `Repeat` (Recurring), `Package` (Artifacts).
- Legacy icons (`icons/*.tsx`, GitHub mark) only used in the `current` comparison mode.

## Files
- `prototype/KFP Modern.dc.html` — clickable prototype (all screens). Needs `support.js` beside it.
- [KEP](../README.md) — component choice, alternatives, compatibility, and migration plan.
- `tokens.css` — theme variables.
- `screenshots/` — 12 captures (light + dark).

## Preview and scope

From this `design/` directory, run:

```sh
python3 -m http.server 8000 --bind 127.0.0.1
```

Open <http://localhost:8000/prototype/KFP%20Modern.dc.html>. The prototype runtime
loads React/Babel from unpkg and fonts from Google Fonts, so its interactive preview
requires network access. The checked-in screenshots are available offline. This
runtime and its dependencies are design-preview assets only; the production UI
continues to use the repository toolchain and bundled fonts.

The KEP is the source for component-stack decisions and delivery criteria. The
original separate stack-comparison document is superseded by the KEP. Counts,
histories, and success rates in the mock are examples; implement only accurate,
bounded existing-API representations under the KEP's statistics rules.

Unpictured upload/version, archive, details, shared, and deployment-specific pages
retain their existing capabilities. Include confirmations, pagination/sorting,
permission states, first-use empty/loading/error states, keyboard focus, reduced
motion, and rich graph/artifact behavior in implementation verification. Artifact
rows must retain access to artifact details, preview, and lineage as well as the
producing task. Keep existing return destinations and supported form options.
