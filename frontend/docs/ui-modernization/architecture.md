# UI component ownership and styling

## Component roles

| Directory               | Responsibility                                                            |
| ----------------------- | ------------------------------------------------------------------------- |
| `components/ui`         | Shared accessible primitives and their visual variants.                   |
| `components/shell`      | Application layout, theme, page chrome and deployment navigation framing. |
| `components/navigation` | Command search and navigation controls.                                   |
| `components/tables`     | Resource-table presentation, pagination controls and Runs presentation.   |
| `components/inspection` | Reusable inspection fields, sections and tabs.                            |
| `components/pipelines`  | Pipeline collection and pipeline form/layout presentation.                     |
| `components/runs`       | Run status, recent-run health and recurring-run controls.                 |

Existing graph, viewer and task-tab components keep their established role directories. New components belong with their responsibility, not a migration phase. The residual `atoms` directory is retained compatibility code, not a second destination for new primitives. Storybook identifiers remain stable when source files move.

## Styling boundary

Use Tailwind utilities and CVA variants for small shared primitives such as buttons and inputs. Use colocated stylesheets for complex page, table, graph and inspection layouts. Both consume the shared theme tokens; avoid parallel declarations for the same visual property across a primitive's utility classes and a caller's stylesheet. CVA selects utility variants rather than defining a separate theme. Keep class merging inside the shared primitive boundary.

`src/tailwind.css` is the single owner of the global preflight/reset and layer order. Theme and reset rules are layered; generated utilities intentionally remain unlayered after the reset so that explicit primitive utilities retain their existing precedence alongside component styles. Do not add page-specific resets or compensate for conflicts with escalating specificity. Extend a primitive's supported variant or change its owning stylesheet instead. The global reset, component styles and utility overrides must be considered together when changing the build or Tailwind imports.

## Page and chrome ownership

| Concern                               | Owner and contract                                                                                                                                                                                                                                                         |
| ------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Resource reads, mutations and loading | The page/controller or its query hook owns requests, pending state, stale-response protection, recovery and payloads. Presentation must not start duplicate requests or independently mirror those states.                                                                 |
| Toolbar actions                       | The page defines available actions, icons, variants, eligibility and callbacks through `Buttons` / `PageChromeTypes`. `PageChrome` renders that contract without inferring behavior from action names. Tests exercise the real action definitions through the renderer.    |
| Banners, dialogs and notifications    | The routed page publishes its messages through the existing chrome callbacks. The route wrapper owns their presentation state and isolates it between pages. Child operations clear only the errors they own; successful child reads must not erase unrelated page errors. |
| Table state                           | `CustomTable` owns filtering, sorting, page size, cursors and request generations. `ResourceTable` and page-specific table components render those values and dispatch user actions; they do not predict generations or shadow encoded filters.                            |
| Navigation                            | The router owns resource paths and the URL owns top-level inspection selection. User navigation pushes history; automatic invalid-selection cleanup replaces it. Local state is reserved for ephemeral controls without a URL contract.                                    |
| Theme and portals                     | The theme provider owns effective theme and the shared portal class. Components consume that context rather than rereading storage or duplicating theme logic.                                                                                                             |

Class-based `Page` controllers and their functional wrappers remain supported in this migration. The end state is one explicit owner per concern using these same contracts, regardless of component syntax. When a page needs an ownership change, migrate that page as a complete slice: remove the duplicate state/adapter responsibility, preserve resource scope and recovery, and verify navigation, mutation and stale-request behavior. A class-to-function conversion alone does not resolve an ownership problem and is not a prerequisite for this UI release.

See [workflow contracts](workflows.md) for keyboard, form, namespace and functional compatibility rules. Generated qualification results belong in [retained evidence archives](evidence-archive.md), not in source directories.

## Component names

Name route entry points `<Resource>Page` (for example `RunDetailsPage`, `PipelinesPage`, `CreateRunPage`, and `UploadPipelinePage`). Name their contents by responsibility: `RunDetailsView`, `CreateRunForm`, `PipelineSelectorDialog`, or `RunParametersForm`. A `View` may coordinate interactions; the suffix does not imply that it is stateless. Route URLs are independent of these implementation names.

Use `WithContext` for small wrappers whose job is injecting namespace/query context, and `Controller` for an exported page controller behind a route wrapper. Use `Router` only for actual route selection, not data loading or an Active/Archived tab control. Avoid presentation names such as `Modern`, `Enhanced`, or `V2` when there is only one implementation. API schemas and format-specific utilities retain meaningful version labels such as `V2beta1Run`.

The two feature-flagged query-based alternative page implementations live in `pages/query` as `CreateExperimentQueryPage` and `RecurringRunDetailsQueryPage`. They own requests, mutations and chrome, so they are not reusable form/view components. Their distinguishing names and existing feature flag remain until a separate ownership migration retires one implementation. Class pages otherwise remain unchanged.

## Page layout

Use a compact workbench layout across resource pages. `PageChrome` owns the title, primary actions, breadcrumbs and 16px page gutter. Align collection controls, tabs and content to that gutter; nested collections inherit the parent content width without adding another gutter.

List pages place filters on the left and view/sort controls on the right in one wrapping control strip. Selection, rows and pagination share the same collection surface. Use a 52px minimum list header, 48px filter strip, and 40px minimum single-line rows with one comfortable default; allow content and responsive wrapping to grow naturally. Keep pagination compact and stable during filtering. Validate density with same-viewport screenshots and the distance from the page top to the first data row, not just shared CSS. Pipelines use a table with a linked latest-version column. Version management belongs on the pipeline detail page, not in an expanded list row.

Detail pages use the same header and tab rhythm, with metadata grouped into readable sections. Graphs and timelines retain a large dedicated workspace and task inspectors; metadata views use responsive fields rather than narrow nested cards. Special-purpose content can differ, but spacing and control placement should not depend on the resource type.

Collections use one compact layout with unchanged readable text sizes. There is no density preference; browser zoom controls overall scaling. Column resizing and remembered column widths remain independent. Namespace-wide import/export belongs in the shell Workspace menu. Pipeline detail pages put the version selector beside the title and full version management in the Versions tab.


## Page layout contract

Collection and detail pages share Public Sans: body and table values 13px/20px,
secondary labels 12px/18px, section headings 16px/24px, and page titles 22px/30px.
Use monospace only for code, structured output, and identifiers where character
alignment is useful; ordinary names and type labels use the body face.
Page gutters are 16px, section gaps 12px, and control gaps 8px. Detail surfaces
use a 16px inset and a consistent label/value grid. Graphs, timelines, editors,
and comparison matrices retain their domain-specific layouts within that frame.

CollectionToolbar owns the search and view-control strip. ResourceTable owns
cell padding, header styling, selection, scrolling, and pagination chrome.
Rows are 32px minimum without reducing text size; wrapped content can grow.
Declare metadata widths on each Column alongside its renderer: dates 190px,
status 120px, duration 100px, and other metadata 160–200px. Leave primary names
flexible and reserve at least 240px for them in the table minimum width. Wide
schemas scroll horizontally instead of squeezing names or wrapping every cell.

Show selection only when the collection offers an operation on selected items.
Use the shared 18px checkbox in a 40px selection column; picker dialogs may use
radio selection. Clicking a selectable row toggles selection; links and other
controls perform their own actions. Read-only lists do not show dummy checkboxes.
Primary names use regular-weight foreground text; secondary metadata uses regular
body text with a quieter color. Differences in columns reflect resource semantics, not independent
page styling choices.

### Expanded collection previews

Expanded pipelines and experiments show at most five recent versions or runs in a compact list, with shared typography and spacing. Previews have no selection, sort controls or pagination. Their links open full detail-page management; pipeline versions use `?tab=versions` with search, sorting, pagination and bulk deletion. Top-level pipeline selection applies only to pipelines. Version labels must not infer a default from a display name: the v2 API has no default-version field.
