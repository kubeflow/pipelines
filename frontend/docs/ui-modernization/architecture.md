# UI component ownership and styling

## Component roles

| Directory | Responsibility |
| --- | --- |
| `components/ui` | Shared accessible primitives and their visual variants. |
| `components/shell` | Application layout, theme, page chrome and deployment navigation framing. |
| `components/navigation` | Command search and navigation controls. |
| `components/tables` | Resource-table presentation, pagination controls and Runs presentation. |
| `components/inspection` | Reusable inspection fields, sections and tabs. |
| `components/pipelines` | Pipeline cards and pipeline form/layout presentation. |
| `components/runs` | Run status, recent-run health and recurring-run controls. |

Existing graph, viewer and task-tab components keep their established role directories. New components belong with their responsibility, not a migration phase. The residual `atoms` directory is retained compatibility code, not a second destination for new primitives. Storybook identifiers remain stable when source files move.

## Styling boundary

Use Tailwind utilities and CVA variants for small shared primitives such as buttons and inputs. Use colocated stylesheets for complex page, table, graph and inspection layouts. Both consume the shared theme tokens; avoid parallel declarations for the same visual property across a primitive's utility classes and a caller's stylesheet. CVA selects utility variants rather than defining a separate theme. Keep class merging inside the shared primitive boundary.

`src/tailwind.css` is the single owner of the global preflight/reset and layer order. Theme and reset rules are layered; generated utilities intentionally remain unlayered after the reset so that explicit primitive utilities retain their existing precedence alongside component styles. Do not add page-specific resets or compensate for conflicts with escalating specificity. Extend a primitive's supported variant or change its owning stylesheet instead. The global reset, component styles and utility overrides must be considered together when changing the build or Tailwind imports.

## Page and chrome ownership

| Concern | Owner and contract |
| --- | --- |
| Resource reads, mutations and loading | The page/controller or its query hook owns requests, pending state, stale-response protection, recovery and payloads. Presentation must not start duplicate requests or independently mirror those states. |
| Toolbar actions | The page defines available actions, icons, variants, eligibility and callbacks through `Buttons` / `PageChromeTypes`. `PageChrome` renders that contract without inferring behavior from action names. Tests exercise the real action definitions through the renderer. |
| Banners, dialogs and notifications | The routed page publishes its messages through the existing chrome callbacks. The route wrapper owns their presentation state and isolates it between pages. Child operations clear only the errors they own; successful child reads must not erase unrelated page errors. |
| Table state | `CustomTable` owns filtering, sorting, page size, cursors and request generations. `ResourceTable` and page-specific table components render those values and dispatch user actions; they do not predict generations or shadow encoded filters. |
| Navigation | The router owns resource paths and the URL owns top-level inspection selection. User navigation pushes history; automatic invalid-selection cleanup replaces it. Local state is reserved for ephemeral controls without a URL contract. |
| Theme and portals | The theme provider owns effective theme and the shared portal class. Components consume that context rather than rereading storage or duplicating theme logic. |

Class-based `Page` controllers and their functional wrappers remain supported in this migration. The end state is one explicit owner per concern using these same contracts, regardless of component syntax. When a page needs an ownership change, migrate that page as a complete slice: remove the duplicate state/adapter responsibility, preserve resource scope and recovery, and verify navigation, mutation and stale-request behavior. A class-to-function conversion alone does not resolve an ownership problem and is not a prerequisite for this UI release.

See [workflow contracts](workflows.md) for keyboard, form, namespace and functional compatibility rules. Generated qualification results belong in [retained evidence archives](evidence-archive.md), not in source directories.

## Component names

Name route entry points `<Resource>Page` (for example `RunDetailsPage`, `PipelinesPage`, `CreateRunPage`, and `UploadPipelinePage`). Name their contents by responsibility: `RunDetailsView`, `CreateRunForm`, `PipelineSelectorDialog`, or `RunParametersForm`. A `View` may coordinate interactions; the suffix does not imply that it is stateless. Route URLs are independent of these implementation names.

Use `WithContext` for small wrappers whose job is injecting namespace/query context, and `Controller` for an exported page controller behind a route wrapper. Use `Router` only for actual route selection, not data loading or an Active/Archived tab control. Avoid presentation names such as `Modern`, `Enhanced`, or `V2` when there is only one implementation. API schemas and format-specific utilities retain meaningful version labels such as `V2beta1Run`.

The two feature-flagged query-based alternative page implementations live in `pages/query` as `CreateExperimentQueryPage` and `RecurringRunDetailsQueryPage`. They own requests, mutations and chrome, so they are not reusable form/view components. Their distinguishing names and existing feature flag remain until a separate ownership migration retires one implementation. Class pages otherwise remain unchanged.
