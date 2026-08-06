# 14 — Discovery Run graph and node inspector

**What to build:** the consultant sees the reasoning as a tree, not as a log — able to watch a Run happen, spot where it stalled, and click a Node to understand why it concluded what it concluded.

**Blocked by:** 09

**Status:** ready-for-agent

The design is variant E from the prototype in `frontend/src/prototype-company/`, the chosen layout. The graph occupies the centre canvas next to the Company Context; the right-hand inspector changes shape with the selected node kind — a form for `ask_human`, a drop area for `ask_document`, stack/deliverables/risks for `spec_solution`, the reason for a discard, impact/effort/priority for an `assess_opportunity`. Nodes are colour-coded by kind, with a legend.

- [ ] Tree drawn with React Flow, automatic layout, `synthesize` pinned to the far end
- [ ] Each Node's status visible on the node itself
- [ ] Blocked Nodes highlighted, so what needs an answer is findable without reading every node
- [ ] Discarded branches dimmed rather than hidden — rejected reasoning stays auditable
- [ ] Delivered spec Nodes marked distinctly
- [ ] Clicking a Node opens the inspector with its input, output and error
- [ ] Run controls on the canvas: optional theme, `opportunity_count`, and the Single/Discovery label derived from it
- [ ] The graph updates by itself while the Run progresses, by polling
- [ ] Route skeleton while loading, not a spinner
