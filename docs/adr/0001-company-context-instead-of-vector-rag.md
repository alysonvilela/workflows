# Company Context is concatenated text, not a vector index

The obvious build for "answer questions over a client's documents" is chunking plus embeddings plus similarity search. We deliberately skipped all of it: each Document is parsed to clean text, and the Company Context is the concatenation of those texts, ingested whole into every prompt.

The reason is that a Company supplies tens of documents, not thousands — the entire corpus fits in a modern context window, so retrieval would be solving a problem we do not have. Skipping it removes pgvector, an embedding provider, a chunking strategy, a chunk table, and the re-indexing that follows every edit.

This has a ceiling, and it is expected to be hit. Vector retrieval is planned as the next phase, at which point the Company Context stops being one blob and becomes a real index. The seam is `BuildCompanyContext(companyID)` — everything upstream of it (S3, parsing, Documents) survives that change untouched.
