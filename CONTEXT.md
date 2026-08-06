# Improvement Discovery

A system that reads a company's own documents and produces concrete software-improvement plans for its processes. It does not execute anything — the deliverable is a plan.

## Language

**Company**:
The organisation whose processes are being analysed, and the owner of everything else in the system.
_Avoid_: Client, customer, tenant, account

**Document**:
A file the company supplied, stored exactly as uploaded.
_Avoid_: File, resource, asset, source

**Company Context**:
Everything the system knows about a Company, as clean text the LLM can read — one entry per Document. It is what gets ingested into every prompt.
_Avoid_: Corpus, knowledge base, RAG, index

**Parse Strategy**:
How one file format (CSV, Markdown) is turned into clean text. Belongs to ingestion.

**Node Strategy**:
How one kind of Node reasons — what it asks the LLM and what children it may spawn. Belongs to a Discovery Run.

**Discovery Run**:
One analysis of a Company, from the moment it is started to the moment a final plan exists.
_Avoid_: Job, session, analysis, execution

**Node**:
One unit of reasoning inside a Discovery Run. It receives what its parent produced, produces its own result, and may spawn children.
_Avoid_: Step, task, stage

**Opportunity**:
A single improvement worth investigating — one process pain that software might solve. Each initial Node of a Discovery Run explores exactly one.
_Avoid_: Idea, finding, insight, improvement

**Single Workflow**:
A Discovery Run started with one initial Node — one Opportunity investigated in depth.

**Discovery Workflow**:
A Discovery Run started with more than one initial Node — several Opportunities explored in parallel.

**Gate**:
A Node that stops its own branch and waits for a human, because the Company Context did not contain something the reasoning needs. Sibling branches keep running. A Gate asks either for an **answer** (a dynamic form) or for a **Document** (a file upload).
_Avoid_: Question, block, pause, human task
