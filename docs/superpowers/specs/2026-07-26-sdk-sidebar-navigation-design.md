# SDK documentation sidebar navigation

## Goal

Preserve the existing `Contents` navigation group unchanged. Remove the
separate generic top-level `Guides` navigation group and expose its remaining
guide categories directly beneath `Contents` in the sidebar.

## Sidebar order

The existing `Contents` group, including all of its entries and order, is out
of scope and remains unchanged. Directly after it, the sidebar will present
these guide entries in this exact order:

1. Concepts
2. User Guides
3. Operator Guides
4. Python SDK
5. Reference
6. Contributor Guide

The existing `Overview`, `Getting Started`, and `Interfaces` entries in the
separate `Guides` group will not be displayed. The global `Guides` dropdown
will also be removed. This does not alter similarly named entries in
`Contents`.

`Concepts`, `User Guides`, `Operator Guides`, and `Reference` are expandable
top-level groups whose first item is their existing landing page and whose
children remain their existing pages. `Python SDK` is an expandable top-level
group for the existing SDK API documentation. `Contributor Guide` is a direct
external link to the repository's `CONTRIBUTING.md`.

## Implementation boundary

Only the guide navigation declarations in `docs/sdk/index.rst` and any
necessary existing guide toctrees will change. The `Contents` declaration,
existing documents, URLs, page content, and version selector remain unchanged.
No new documentation content is introduced.

## Error handling and compatibility

All navigation targets must resolve during a Sphinx HTML build. The external
Contributor Guide link remains a normal external navigation target. Existing
guide URLs stay stable because pages are moved only in the navigation tree, not
renamed or relocated.

## Verification

Build the SDK documentation with `sphinx-build -b html . _build/html` from
`docs/sdk`, check the working diff with `git diff --check`, and inspect the
generated sidebar to confirm the approved order and expandable groups.
