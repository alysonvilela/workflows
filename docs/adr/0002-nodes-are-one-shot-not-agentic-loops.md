# A Node is one LLM call, not an agent loop

Node Strategies drive the LLM through tool calling, which normally implies a loop: the model reads, calls a tool, reads the result, calls another. We deliberately do not loop. A Node builds one prompt containing the whole Company Context, and the model must answer with exactly one terminal tool call. The orchestrator turns that call into child Nodes.

The reason is that the loop exists to let a model go and fetch context it wasn't given — and per ADR-0001 the entire Company Context already fits in the prompt. Without read tools there is nothing to iterate over, so the loop would only multiply LLM calls per Node. One call per Node also keeps the Node a pure function of its inputs, which is what makes the orchestrator testable against a scripted fake provider.

The looping version is a good design and is expected to be revisited — specifically when vector retrieval lands and the corpus stops fitting in the prompt. At that point the Node gains read tools (`read_document`, `search_context`), a message history, and a `max_steps` budget, and this decision is superseded.
