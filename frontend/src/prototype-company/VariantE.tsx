// PROTOTYPE — Variant E: "Cockpit". CHOSEN DESIGN.
// Three columns: Company Context on the left, the Discovery Run canvas in the
// middle with the run bar on top, and an inspector on the right whose shape
// changes with the selected node kind.
// Palette: violet as primary, amber for what is waiting, emerald for what was
// delivered, slate for what was discarded.
import "@xyflow/react/dist/style.css";
import {
  Background,
  BackgroundVariant,
  type Edge,
  Handle,
  type Node as FlowNode,
  Position,
  ReactFlow,
} from "@xyflow/react";
import {
  CheckCircle2Icon,
  ClockIcon,
  FileTextIcon,
  LayersIcon,
  SparklesIcon,
  TrashIcon,
  UploadIcon,
  XCircleIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import {
  Accordion,
  AccordionItem,
  AccordionPanel,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectItem,
  SelectPopup,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import { Textarea } from "@/components/ui/textarea";
import {
  childrenOf,
  fmtBytes,
  isDiscarded,
  type NodeKind,
  type RunNode,
  runStatusLabel,
  score,
  type Store,
} from "./data";

export const name = "Cockpit — run bar + inspector per node kind";

/* palette per node kind ----------------------------------------------- */

const kindStyle: Record<NodeKind, { ring: string; chip: string; dot: string }> = {
  scan_opportunities: {
    ring: "border-violet-300 bg-violet-50 dark:border-violet-500/40 dark:bg-violet-500/10",
    chip: "text-violet-700 dark:text-violet-300",
    dot: "bg-violet-500",
  },
  assess_opportunity: {
    ring: "border-indigo-200 bg-white dark:border-indigo-500/30 dark:bg-indigo-500/5",
    chip: "text-indigo-700 dark:text-indigo-300",
    dot: "bg-indigo-500",
  },
  ask_human: {
    ring: "border-amber-300 bg-amber-50 dark:border-amber-500/40 dark:bg-amber-500/10",
    chip: "text-amber-700 dark:text-amber-300",
    dot: "bg-amber-500",
  },
  ask_document: {
    ring: "border-orange-300 bg-orange-50 dark:border-orange-500/40 dark:bg-orange-500/10",
    chip: "text-orange-700 dark:text-orange-300",
    dot: "bg-orange-500",
  },
  spec_solution: {
    ring: "border-emerald-300 bg-emerald-50 dark:border-emerald-500/40 dark:bg-emerald-500/10",
    chip: "text-emerald-700 dark:text-emerald-300",
    dot: "bg-emerald-500",
  },
  synthesize: {
    ring: "border-fuchsia-300 bg-fuchsia-50 dark:border-fuchsia-500/40 dark:bg-fuchsia-500/10",
    chip: "text-fuchsia-700 dark:text-fuchsia-300",
    dot: "bg-fuchsia-500",
  },
};

/* node ----------------------------------------------------------------- */

interface NodeData extends Record<string, unknown> {
  node: RunNode;
  selected: boolean;
}

function CockpitNode({ data }: { data: NodeData }): React.ReactElement {
  const n = data.node;
  const s = kindStyle[n.kind];
  const dim = isDiscarded(n);

  return (
    <div
      className={`w-56 rounded-2xl border-2 px-3.5 py-2.5 shadow-sm transition-all ${
        dim ? "border-dashed border-slate-300 bg-slate-50/70 opacity-64" : s.ring
      } ${data.selected ? "ring-2 ring-violet-400 ring-offset-2" : ""}`}
    >
      <Handle position={Position.Top} type="target" className="!bg-transparent !border-0" />
      <div className="flex items-center gap-1.5">
        <span className={`size-1.5 rounded-full ${dim ? "bg-slate-400" : s.dot}`} />
        <span className={`font-mono text-[10px] ${dim ? "text-slate-500" : s.chip}`}>
          {n.kind}
        </span>
        <span className="flex-1" />
        {n.status === "running" && <Spinner className="size-3" />}
        {n.status === "blocked" && <ClockIcon className="size-3 text-amber-600" />}
        {n.status === "done" && !dim && (
          <CheckCircle2Icon className="size-3 text-emerald-600" />
        )}
        {dim && <XCircleIcon className="size-3 text-slate-400" />}
      </div>
      <p className="mt-1 line-clamp-2 font-medium text-[13px] leading-snug">{n.title}</p>
      {n.impact && n.effort && (
        <div className="mt-2 flex items-center gap-2 text-[10px] text-muted-foreground">
          <span className="rounded bg-black/4 px-1.5 py-0.5 dark:bg-white/8">
            impact {n.impact}
          </span>
          <span className="rounded bg-black/4 px-1.5 py-0.5 dark:bg-white/8">
            effort {n.effort}
          </span>
          <span className="ms-auto font-semibold text-foreground">{score(n)}×</span>
        </div>
      )}
      {n.waitingSince && (
        <p className="mt-2 text-[10px] text-amber-700 dark:text-amber-300">
          waiting {n.waitingSince}
        </p>
      )}
      <Handle
        position={Position.Bottom}
        type="source"
        className="!bg-transparent !border-0"
      />
    </div>
  );
}

const nodeTypes = { cockpit: CockpitNode };

/* layered layout -------------------------------------------------------- */

function layout(nodes: RunNode[]): { flow: FlowNode<NodeData>[]; edges: Edge[] } {
  const flow: FlowNode<NodeData>[] = [];
  const edges: Edge[] = [];
  const place = (parentId: string | null, depth: number, offset: number): number => {
    const kids = childrenOf(nodes, parentId);
    let x = offset;
    for (const k of kids) {
      const width = place(k.id, depth + 1, x);
      flow.push({
        id: k.id,
        type: "cockpit",
        position: { x: x + (width - 240) / 2, y: depth * 150 },
        data: { node: k, selected: false },
      });
      if (parentId) {
        edges.push({
          id: `${parentId}-${k.id}`,
          source: parentId,
          target: k.id,
          type: "smoothstep",
          style: { stroke: "#c4b5fd", strokeWidth: 1.5 },
        });
      }
      x += width;
    }
    return Math.max(x - offset, 260);
  };
  place(null, 0, 0);
  return { flow, edges };
}

/* screen ---------------------------------------------------------------- */

export function VariantE({ store }: { store: Store }): React.ReactElement {
  const [sel, setSel] = useState<string | null>("n2");
  const [count, setCount] = useState(4);
  const [theme, setTheme] = useState("");
  const c = store.selected;
  const run = store.run;
  const nodes = run?.nodes ?? [];
  const selected = nodes.find((n) => n.id === sel) ?? null;

  const { flow, edges } = useMemo(() => layout(nodes), [nodes]);
  const flowWithSel = flow.map((f) => ({
    ...f,
    data: { ...f.data, selected: f.id === sel },
  }));

  return (
    <div className="flex h-dvh flex-col bg-gradient-to-b from-violet-50/60 to-background text-foreground dark:from-violet-950/20">
      {/* top bar */}
      <header className="flex items-center gap-3 border-b bg-card/80 px-5 py-2.5 backdrop-blur">
        <div className="flex size-8 items-center justify-center rounded-xl bg-gradient-to-br from-violet-500 to-indigo-600 text-white shadow-sm">
          <SparklesIcon className="size-4" />
        </div>
        <Select
          items={store.companies.map((co) => ({ label: co.name, value: co.id }))}
          value={store.selectedId}
          onValueChange={(v) => store.select(v as string)}
        >
          <SelectTrigger className="w-56 border-0 bg-transparent font-semibold shadow-none">
            <SelectValue />
          </SelectTrigger>
          <SelectPopup>
            {store.companies.map((co) => (
              <SelectItem key={co.id} value={co.id}>
                {co.name}
              </SelectItem>
            ))}
          </SelectPopup>
        </Select>
        <div className="flex-1" />
        {run && (
          <Badge
            variant={run.status === "awaiting_input" ? "warning" : "info"}
            className="gap-1.5"
          >
            <span className="size-1.5 animate-pulse rounded-full bg-current" />
            Run {runStatusLabel[run.status]}
          </Badge>
        )}
        <Button size="sm" variant="ghost" onClick={() => c && store.deleteCompany(c.id)}>
          <TrashIcon />
          Delete company
        </Button>
      </header>

      <div className="grid min-h-0 flex-1 grid-cols-[340px_1fr_400px]">
        {/* Company Context */}
        <aside className="flex min-h-0 flex-col overflow-auto border-r bg-card/40">
          <div className="p-4">
            <button
              type="button"
              onClick={() => c && store.addDocument(c.id, `document-${Date.now()}.md`)}
              className="flex w-full flex-col items-center gap-1 rounded-2xl border-2 border-violet-200 border-dashed bg-violet-50/50 py-6 transition-colors hover:border-violet-400 hover:bg-violet-100/50 dark:border-violet-500/30 dark:bg-violet-500/5"
            >
              <UploadIcon className="size-5 text-violet-500" />
              <span className="font-medium text-sm">Drop Markdown or CSV</span>
              <span className="text-[11px] text-muted-foreground">
                up to 10 MB · becomes context automatically
              </span>
            </button>
          </div>

          <div className="flex items-center gap-2 px-4 pb-1">
            <LayersIcon className="size-3.5 text-muted-foreground" />
            <h2 className="font-semibold text-sm">Company Context</h2>
            <Badge size="sm" variant="secondary">
              {c?.docs.filter((d) => d.status === "ready").length}/{c?.docs.length}
            </Badge>
          </div>

          <div className="px-4 pb-6">
            <Accordion className="w-full">
              {c?.docs.map((d) => (
                <AccordionItem key={d.id} value={d.id}>
                  <AccordionTrigger className="gap-2">
                    <span className="flex min-w-0 flex-1 items-center gap-2">
                      <FileTextIcon
                        className={`size-3.5 shrink-0 ${
                          d.status === "failed" ? "text-rose-500" : "text-violet-500"
                        }`}
                      />
                      <span className="truncate text-[13px]">{d.name}</span>
                      <Badge
                        size="sm"
                        variant={
                          d.status === "ready"
                            ? "success"
                            : d.status === "failed"
                              ? "error"
                              : "warning"
                        }
                      >
                        {d.status}
                      </Badge>
                      <span className="ms-auto shrink-0 text-[11px] text-muted-foreground">
                        {fmtBytes(d.bytes)}
                      </span>
                    </span>
                  </AccordionTrigger>
                  <AccordionPanel>
                    <div className="pb-4">
                      {d.error && (
                        <p className="rounded-lg bg-rose-50 px-2.5 py-1.5 text-rose-700 text-xs dark:bg-rose-500/10 dark:text-rose-300">
                          {d.error}
                        </p>
                      )}
                      {d.summary && (
                        <>
                          <p className="font-medium text-foreground">{d.summary.theme}</p>
                          <ul className="mt-1.5 space-y-1">
                            {d.summary.pains.map((p) => (
                              <li key={p} className="flex gap-1.5 text-xs">
                                <span className="mt-1.5 size-1 shrink-0 rounded-full bg-rose-400" />
                                {p}
                              </li>
                            ))}
                          </ul>
                        </>
                      )}
                      <div className="mt-3 flex gap-1.5">
                        <Button
                          size="xs"
                          variant="outline"
                          onClick={() => c && store.resummarise(c.id, d.id)}
                        >
                          Re-summarise
                        </Button>
                        <Button
                          size="xs"
                          variant="ghost"
                          onClick={() => c && store.removeDocument(c.id, d.id)}
                        >
                          Remove
                        </Button>
                      </div>
                    </div>
                  </AccordionPanel>
                </AccordionItem>
              ))}
            </Accordion>
          </div>
        </aside>

        {/* canvas */}
        <section className="relative min-h-0">
          {/* run bar */}
          <div className="absolute inset-x-4 top-4 z-10 flex items-center gap-2 rounded-2xl border bg-card/90 p-2 shadow-sm backdrop-blur">
            <Input
              value={theme}
              onChange={(e) => setTheme(e.target.value)}
              placeholder="Discovery theme (optional) — e.g. sales process"
              className="h-8 flex-1 border-0 shadow-none focus-visible:ring-0"
            />
            <Separator orientation="vertical" className="h-5" />
            <span className="text-[11px] text-muted-foreground">opportunities</span>
            <div className="flex items-center rounded-lg border">
              <Button
                size="icon-xs"
                variant="ghost"
                onClick={() => setCount(Math.max(1, count - 1))}
              >
                −
              </Button>
              <span className="w-6 text-center font-semibold text-sm tabular-nums">
                {count}
              </span>
              <Button
                size="icon-xs"
                variant="ghost"
                onClick={() => setCount(Math.min(6, count + 1))}
              >
                +
              </Button>
            </div>
            <Badge size="sm" variant={count === 1 ? "secondary" : "info"}>
              {count === 1 ? "Single Workflow" : "Discovery Workflow"}
            </Badge>
            <Button
              size="sm"
              className="bg-gradient-to-br from-violet-600 to-indigo-600"
              onClick={() => c && store.startRun(c.id, theme, count)}
            >
              Run discovery
            </Button>
          </div>

          <ReactFlow
            nodes={flowWithSel}
            edges={edges}
            nodeTypes={nodeTypes}
            fitView
            fitViewOptions={{ padding: 0.25 }}
            onNodeClick={(_, n) => setSel(n.id)}
            proOptions={{ hideAttribution: true }}
          >
            <Background variant={BackgroundVariant.Dots} gap={18} size={1} />
          </ReactFlow>

          {/* legend */}
          <div className="absolute bottom-4 left-4 flex flex-wrap gap-2 rounded-xl border bg-card/90 px-3 py-2 text-[10px] backdrop-blur">
            {(Object.keys(kindStyle) as NodeKind[]).map((k) => (
              <span key={k} className="flex items-center gap-1">
                <span className={`size-1.5 rounded-full ${kindStyle[k].dot}`} />
                {k}
              </span>
            ))}
          </div>
        </section>

        {/* inspector */}
        <aside className="flex min-h-0 flex-col overflow-auto border-l bg-card">
          {!selected && (
            <p className="p-6 text-muted-foreground text-sm">Select a node to inspect it.</p>
          )}
          {selected && (
            <>
              <div className="px-5 pt-5">
                <p className={`font-mono text-[11px] ${kindStyle[selected.kind].chip}`}>
                  {selected.kind}
                </p>
                <h2 className="mt-0.5 font-semibold text-lg leading-tight">
                  {selected.title}
                </h2>
                {selected.detail && (
                  <p className="mt-1 text-muted-foreground text-sm">{selected.detail}</p>
                )}
              </div>
              <Separator className="mt-4" />

              <div className="flex-1 space-y-5 overflow-auto px-5 py-4">
                {/* discarded */}
                {isDiscarded(selected) && (
                  <div className="rounded-xl border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-800/40">
                    <p className="font-medium text-slate-600 text-xs uppercase dark:text-slate-300">
                      Discarded
                    </p>
                    <p className="mt-1 text-sm">{selected.discardReason}</p>
                  </div>
                )}

                {/* scores */}
                {selected.impact && selected.effort && (
                  <div className="grid grid-cols-3 gap-2">
                    {[
                      { label: "impact", v: selected.impact, tone: "text-violet-600" },
                      { label: "effort", v: selected.effort, tone: "text-amber-600" },
                      {
                        label: "priority",
                        v: `${score(selected)}×`,
                        tone: "text-emerald-600",
                      },
                    ].map((m) => (
                      <div key={m.label} className="rounded-xl border p-3 text-center">
                        <p className={`font-semibold text-xl ${m.tone}`}>{m.v}</p>
                        <p className="text-[11px] text-muted-foreground">{m.label}</p>
                      </div>
                    ))}
                  </div>
                )}

                {/* form gate */}
                {selected.kind === "ask_human" && (
                  <>
                    <div className="rounded-xl border border-amber-200 bg-amber-50 p-3 dark:border-amber-500/30 dark:bg-amber-500/10">
                      <p className="font-medium text-sm">{selected.gateTitle}</p>
                      <p className="mt-1 text-muted-foreground text-xs">
                        {selected.gateWhatFor}
                      </p>
                    </div>
                    {selected.gateFields?.map((f) => (
                      <div key={f.name}>
                        <label className="mb-1 block font-medium text-sm">
                          {f.label}
                          {f.required && <span className="text-rose-500"> *</span>}
                        </label>
                        {f.type === "enum" ? (
                          <Select items={f.options?.map((o) => ({ label: o, value: o }))}>
                            <SelectTrigger className="w-full">
                              <SelectValue placeholder="Select…" />
                            </SelectTrigger>
                            <SelectPopup>
                              {f.options?.map((o) => (
                                <SelectItem key={o} value={o}>
                                  {o}
                                </SelectItem>
                              ))}
                            </SelectPopup>
                          </Select>
                        ) : f.type === "boolean" ? (
                          <div className="flex gap-2">
                            <Button size="sm" variant="outline">
                              Yes
                            </Button>
                            <Button size="sm" variant="outline">
                              No
                            </Button>
                          </div>
                        ) : f.name === "notes" ? (
                          <Textarea placeholder={f.placeholder} rows={3} />
                        ) : (
                          <Input placeholder={f.placeholder} />
                        )}
                      </div>
                    ))}
                  </>
                )}

                {/* document gate */}
                {selected.kind === "ask_document" && (
                  <>
                    <div className="rounded-xl border border-orange-200 bg-orange-50 p-3 dark:border-orange-500/30 dark:bg-orange-500/10">
                      <p className="font-medium text-sm">{selected.gateTitle}</p>
                      <p className="mt-1 text-muted-foreground text-xs">
                        {selected.gateWhatFor}
                      </p>
                    </div>
                    <button
                      type="button"
                      className="flex w-full flex-col items-center gap-1 rounded-2xl border-2 border-orange-300 border-dashed bg-orange-50/50 py-8 hover:bg-orange-100/50 dark:border-orange-500/30 dark:bg-orange-500/5"
                    >
                      <UploadIcon className="size-5 text-orange-500" />
                      <span className="font-medium text-sm">Upload the requested file</span>
                      <span className="text-[11px] text-muted-foreground">
                        joins the Company Context and unblocks this branch
                      </span>
                    </button>
                  </>
                )}

                {/* delivered spec */}
                {selected.spec && (
                  <div className="space-y-3">
                    {(
                      [
                        ["Stack", selected.spec.stack],
                        ["Deliverables", selected.spec.deliverables],
                        ["Risks", selected.spec.risks],
                      ] as const
                    ).map(([label, items]) => (
                      <div key={label}>
                        <p className="mb-1 font-medium text-muted-foreground text-xs uppercase">
                          {label}
                        </p>
                        <ul className="space-y-1">
                          {items.map((i) => (
                            <li
                              key={i}
                              className="rounded-lg bg-muted/60 px-2.5 py-1.5 text-sm"
                            >
                              {i}
                            </li>
                          ))}
                        </ul>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {(selected.kind === "ask_human" || selected.kind === "ask_document") && (
                <>
                  <Separator />
                  <div className="flex gap-2 p-4">
                    <Button
                      variant="outline"
                      className="flex-1"
                      onClick={() => run && store.dismissGate(run.id, selected.id)}
                    >
                      I don't have this
                    </Button>
                    <Button
                      className="flex-1 bg-gradient-to-br from-violet-600 to-indigo-600"
                      onClick={() => run && store.answerGate(run.id, selected.id)}
                    >
                      Answer and unblock
                    </Button>
                  </div>
                </>
              )}
            </>
          )}
        </aside>
      </div>
    </div>
  );
}
