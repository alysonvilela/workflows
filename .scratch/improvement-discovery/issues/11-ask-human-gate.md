# 11 — `ask_human` Gate

**What to build:** when the Company Context does not answer something the reasoning needs, the Node stops and asks. The consultant sees the question as a form, answers it, and that branch resumes where it stopped — while the siblings never stopped at all.

**Blocked by:** 09

**Status:** ready-for-agent

- [ ] `request_information` creates an `ask_human` Node with the JSON Schema in `input`, and the branch goes `blocked`
- [ ] The emitted schema is restricted to a flat object with `string`, `number`, `boolean` or `enum` properties; a schema outside the subset is a retry against the model, not an error shown to the consultant
- [ ] The Run reads `awaiting_input` while any Gate is open and returns to `running` when none is
- [ ] Form rendered with RJSF under a custom theme built on the `@coss` components, without modifying the primitives
- [ ] The answer is validated against the schema on the server, not only on the client
- [ ] Answering unblocks only that Node; siblings were running the whole time
- [ ] Submitting the same answer twice takes effect once
- [ ] Dismissing a Gate lets the Node proceed on the conservative assumption, stated in its output
- [ ] A single inbox listing every open Gate across all Runs, showing how long each has waited
- [ ] A blocked Run consumes no processing at all while it waits
- [ ] Behavioural tests covering open, answer, dismiss and double-answer
