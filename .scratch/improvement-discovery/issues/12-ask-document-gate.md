# 12 — `ask_document` Gate

**What to build:** a Node asks for a file nobody uploaded — the cost spreadsheet, the CRM export. The consultant uploads it, it joins the Company Context like any other Document, and the branch resumes already seeing the new data.

**Blocked by:** 11, 06

**Status:** ready-for-agent

This is what makes the plan measurable: without the client's real numbers, the spec is guesswork.

- [ ] `request_document` creates an `ask_document` Node carrying the question title and what the file is for
- [ ] The Gate is answered by upload, not by form — no file embedded in JSON
- [ ] The uploaded file travels the same ingestion path as any Document and enters the Company Context
- [ ] The Node unblocks only once the Document is ready, and the branch resumes with the context already updated
- [ ] Every Node afterwards, in any branch, sees the new Document
- [ ] Dismiss behaves as it does for `ask_human`, with the conservative assumption recorded
- [ ] Behavioural test: a Run blocks asking for a document, an upload answers it, and the branch completes using the uploaded content
