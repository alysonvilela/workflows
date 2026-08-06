# 03 — Company CRUD

**What to build:** the consultant creates, renames and deletes Companies from the interface, and switches between them. This is the first slice with user value: without a Company nothing else in the system exists.

**Blocked by:** 02

**Status:** ready-for-agent

The screen comes from the prototype in `frontend/src/prototype-company/`. **Variant E — "Cockpit" — is the chosen design**, picked over an inbox-first layout (F) and a plan-first layout (G) because it keeps document management, the Discovery Run graph and the node inspector visible at once, which is what the work actually needs. Rewrite it under production conditions; the prototype was written without tests or error handling, so do not promote it as-is.

Layout E, in short: a header with a Company selector, a left column holding the drop area plus the Company Context accordion, a centre canvas holding the run bar and the graph, and a right inspector whose shape changes with the selected node.

- [ ] Create a Company from the frontend, with the form visible immediately — no intermediate modal
- [ ] List Companies with their Document counts
- [ ] Rename a Company; the `slug` is derived from the name, never typed
- [ ] Delete a Company, with confirmation, cascading to its Documents
- [ ] Validation errors inline in the form; toasts reserved for notifications
- [ ] Matching REST endpoints, with list responses using the project's pagination contract
- [ ] Behavioural tests at Seam 1 covering create, rename, delete and list
