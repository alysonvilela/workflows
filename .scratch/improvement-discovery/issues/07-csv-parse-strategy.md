# 07 — CSV Parse Strategy

**What to build:** the consultant uploads a CSV — a conversation export, an operational spreadsheet, a ticket export, anything — and it becomes `raw_text` just like Markdown, entering the Company Context through the same path.

**Blocked by:** 04

**Status:** ready-for-agent

Can run in parallel with 05 and 06: it depends only on the upload path, not on the summariser.

- [ ] The CSV Parse Strategy reads the file as a stream, without loading it all into memory
- [ ] The CSV shape is not assumed — arbitrary header, arbitrary column count
- [ ] Parse Strategy selection comes from the file type, and an unknown type fails the Document with a clear message
- [ ] An oversized file fails through the same size guard as Markdown, with no sampling and no map-reduce (out of scope by decision)
- [ ] Unit tests for both Parse Strategies with sample files, including a malformed CSV and an empty CSV
