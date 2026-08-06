# 04 — Markdown upload through to `raw_text`

**What to build:** the consultant uploads a Markdown file to a Company and watches the Document appear, get processed, and become available with its extracted text. No LLM involved yet.

**Blocked by:** 03

**Status:** ready-for-agent

- [ ] Upload from the frontend, one or several files at a time
- [ ] The file lands in object storage before any parsing happens
- [ ] The Markdown Parse Strategy turns the file into `raw_text`
- [ ] A Document has a visible status (`parsing` → `ready`) and the consultant sees the transition without reloading the page
- [ ] A file above `MAX_UPLOAD_BYTES` fails as a Document error, with a readable message, before any processing
- [ ] A parsing failure records the reason on the Document rather than failing the request
- [ ] `checksum` prevents reprocessing the same file uploaded twice
- [ ] Document list per Company, with the extracted text reachable
- [ ] Behavioural tests at Seam 1 covering a successful upload, an oversized file and an unreadable file
