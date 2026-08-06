// PROTOTYPE — throwaway. In-memory stub of the whole domain.
// Covers what the screen has to show: Company Context, a Discovery Run with
// opportunity_count, a Node tree with impact/effort, both Gate kinds, discard
// reasons, the cross-run inbox and the final plan.
import { useMemo, useState } from "react";

/* ── Documents ───────────────────────────────────────────────────────── */

export type DocStatus = "parsing" | "summarising" | "ready" | "failed";

export interface Summary {
  theme: string;
  pains: string[];
  opportunities: string[];
}

export interface Doc {
  id: string;
  name: string;
  kind: "markdown" | "csv";
  status: DocStatus;
  bytes: number;
  addedAt: string;
  summary: Summary | null;
  error?: string;
}

/* ── Discovery Run ───────────────────────────────────────────────────── */

export type NodeKind =
  | "scan_opportunities"
  | "assess_opportunity"
  | "ask_human"
  | "ask_document"
  | "spec_solution"
  | "synthesize";

export type NodeStatus = "pending" | "running" | "blocked" | "done" | "failed";

export interface GateField {
  name: string;
  label: string;
  type: "string" | "number" | "boolean" | "enum";
  options?: string[];
  placeholder?: string;
  required?: boolean;
}

export interface RunNode {
  id: string;
  parentId: string | null;
  kind: NodeKind;
  status: NodeStatus;
  title: string;
  detail?: string;
  /** assess_opportunity only — ranked by the LLM */
  impact?: number;
  effort?: number;
  /** set when the strategy discarded the Opportunity */
  discardReason?: string;
  /** ask_human / ask_document only */
  gateTitle?: string;
  gateWhatFor?: string;
  gateFields?: GateField[];
  waitingSince?: string;
  /** spec_solution only */
  spec?: { stack: string[]; deliverables: string[]; risks: string[] };
}

export type RunStatus = "running" | "awaiting_input" | "synthesizing" | "done";

export interface Run {
  id: string;
  companyId: string;
  companyName: string;
  theme: string | null;
  opportunityCount: number;
  maxDepth: number;
  status: RunStatus;
  startedAt: string;
  nodes: RunNode[];
  plan: { headline: string; narrative: string; items: string[] } | null;
}

export interface Company {
  id: string;
  name: string;
  slug: string;
  docs: Doc[];
}

/* ── seed ────────────────────────────────────────────────────────────── */

const uid = (): string => Math.random().toString(36).slice(2, 9);

const doc = (
  name: string,
  kind: Doc["kind"],
  status: DocStatus,
  bytes: number,
  addedAt: string,
  summary: Summary | null,
  error?: string,
): Doc => ({ id: uid(), name, kind, status, bytes, addedAt, summary, error });

const seedCompanies = (): Company[] => [
  {
    id: "co-aurora",
    name: "Aurora Logistics",
    slug: "aurora-logistics",
    docs: [
      doc("sales-process.md", "markdown", "ready", 18_400, "1 Aug", {
        theme: "Sales process",
        pains: [
          "Discount approval passes through three people on three different channels",
          "Every quote is rebuilt by hand in a spreadsheet",
        ],
        opportunities: [
          "Rule-based approval up to a ceiling",
          "Quote generator driven by the catalogue",
        ],
      }),
      doc("support-conversations.csv", "csv", "ready", 2_310_000, "2 Aug", {
        theme: "Post-sale support",
        pains: ["Delivery status is looked up by hand on every customer question"],
        opportunities: ["Self-service tracking link per order"],
      }),
      doc("pricing-policy.md", "markdown", "ready", 7_800, "3 Aug", {
        theme: "Pricing policy",
        pains: ["The discount table lives in a spreadsheet with no history"],
        opportunities: ["Versioned table with an audit trail"],
      }),
      doc("costs-2026.csv", "csv", "summarising", 88_200, "6 Aug", null),
      doc(
        "supplier-contract.md",
        "markdown",
        "failed",
        41_000_000,
        "6 Aug",
        null,
        "Above MAX_UPLOAD_BYTES (10 MB)",
      ),
    ],
  },
  {
    id: "co-clinic",
    name: "Bem-Estar Clinic",
    slug: "bem-estar-clinic",
    docs: [
      doc("scheduling.md", "markdown", "ready", 9_100, "28 Jul", {
        theme: "Scheduling",
        pains: ["Rescheduling happens by phone and leaves no record"],
        opportunities: ["Self-service rescheduling with a window policy"],
      }),
    ],
  },
  { id: "co-vera", name: "Vera Accounting", slug: "vera-accounting", docs: [] },
];

const auroraNodes: RunNode[] = [
  {
    id: "n1",
    parentId: null,
    kind: "scan_opportunities",
    status: "done",
    title: "4 opportunities ranked",
    detail: "Scanned 3 ready sources from the Company Context",
  },
  {
    id: "n2",
    parentId: "n1",
    kind: "assess_opportunity",
    status: "blocked",
    title: "Discount approval",
    detail: "Three approvers, no automatic ceiling",
    impact: 5,
    effort: 2,
  },
  {
    id: "n3",
    parentId: "n1",
    kind: "assess_opportunity",
    status: "running",
    title: "Quotes built by hand",
    detail: "Spreadsheet rebuilt for every order",
    impact: 4,
    effort: 2,
  },
  {
    id: "n4",
    parentId: "n1",
    kind: "assess_opportunity",
    status: "done",
    title: "Delivery tracking",
    detail: "Manual lookup on every customer question",
    impact: 3,
    effort: 3,
  },
  {
    id: "n5",
    parentId: "n1",
    kind: "assess_opportunity",
    status: "done",
    title: "Version the pricing table",
    discardReason:
      "Solved by process rather than software — the spreadsheet already has a single owner and enough manual history.",
    impact: 2,
    effort: 4,
  },
  {
    id: "n6",
    parentId: "n2",
    kind: "ask_document",
    status: "blocked",
    title: "I need the cost spreadsheet",
    gateTitle: "Upload the per-order cost spreadsheet",
    gateWhatFor:
      "Without cost per order there is no way to compute a safe discount ceiling or measure the gain.",
    waitingSince: "2 days",
  },
  {
    id: "n7",
    parentId: "n3",
    kind: "spec_solution",
    status: "done",
    title: "Quote generator",
    spec: {
      stack: ["Template service", "Catalogue as single source", "PDF export"],
      deliverables: [
        "Four-field form that generates the quote",
        "Versioned catalogue with the current price",
      ],
      risks: ["The catalogue lives in a spreadsheet today — it has to move first"],
    },
  },
  {
    id: "n8",
    parentId: "n4",
    kind: "spec_solution",
    status: "done",
    title: "Self-service tracking",
    spec: {
      stack: ["Carrier webhook", "Public page per order"],
      deliverables: ["Tracking link sent with the order confirmation"],
      risks: ["Carrier B exposes no webhook — polling as the fallback"],
    },
  },
];

const clinicNodes: RunNode[] = [
  {
    id: "m1",
    parentId: null,
    kind: "scan_opportunities",
    status: "done",
    title: "1 opportunity ranked",
  },
  {
    id: "m2",
    parentId: "m1",
    kind: "assess_opportunity",
    status: "blocked",
    title: "Self-service rescheduling",
    impact: 4,
    effort: 2,
  },
  {
    id: "m3",
    parentId: "m2",
    kind: "ask_human",
    status: "blocked",
    title: "What is the rescheduling policy?",
    gateTitle: "Rescheduling policy",
    gateWhatFor:
      "The documents do not say how much notice a reschedule needs, or whether a fee applies.",
    waitingSince: "5 hours",
    gateFields: [
      {
        name: "notice_hours",
        label: "Minimum notice (hours)",
        type: "number",
        placeholder: "e.g. 24",
        required: true,
      },
      {
        name: "charges_fee",
        label: "Is a fee charged outside the window?",
        type: "boolean",
      },
      {
        name: "who_approves",
        label: "Who approves an exception",
        type: "enum",
        options: ["Front desk", "Coordination", "Nobody — it is automatic"],
        required: true,
      },
      {
        name: "notes",
        label: "Notes",
        type: "string",
        placeholder: "Anything else that helps",
      },
    ],
  },
];

const seedRuns = (): Run[] => [
  {
    id: "run-1",
    companyId: "co-aurora",
    companyName: "Aurora Logistics",
    theme: "Sales process",
    opportunityCount: 4,
    maxDepth: 3,
    status: "awaiting_input",
    startedAt: "6 Aug, 09:12",
    nodes: auroraNodes,
    plan: null,
  },
  {
    id: "run-2",
    companyId: "co-clinic",
    companyName: "Bem-Estar Clinic",
    theme: null,
    opportunityCount: 1,
    maxDepth: 3,
    status: "awaiting_input",
    startedAt: "5 Aug, 16:40",
    nodes: clinicNodes,
    plan: null,
  },
];

/* ── store ───────────────────────────────────────────────────────────── */

export interface Store {
  companies: Company[];
  selected: Company | undefined;
  selectedId: string;
  select: (id: string) => void;
  createCompany: (name: string) => void;
  renameCompany: (id: string, name: string) => void;
  deleteCompany: (id: string) => void;
  addDocument: (companyId: string, name: string) => void;
  removeDocument: (companyId: string, docId: string) => void;
  resummarise: (companyId: string, docId: string) => void;

  runs: Run[];
  run: Run | undefined;
  /** every open Gate, across every Run */
  inbox: (RunNode & { runId: string; companyName: string })[];
  answerGate: (runId: string, nodeId: string) => void;
  dismissGate: (runId: string, nodeId: string) => void;
  startRun: (companyId: string, theme: string, opportunityCount: number) => void;
}

export function useStore(): Store {
  const [companies, setCompanies] = useState<Company[]>(seedCompanies);
  const [runs, setRuns] = useState<Run[]>(seedRuns);
  const [selectedId, setSelectedId] = useState("co-aurora");

  const selected = companies.find((c) => c.id === selectedId) ?? companies[0];
  const run = runs.find((r) => r.companyId === selected?.id);

  const inbox = useMemo(
    () =>
      runs.flatMap((r) =>
        r.nodes
          .filter(
            (n) =>
              (n.kind === "ask_human" || n.kind === "ask_document") &&
              n.status === "blocked",
          )
          .map((n) => ({ ...n, runId: r.id, companyName: r.companyName })),
      ),
    [runs],
  );

  const patchCompany = (id: string, fn: (c: Company) => Company): void =>
    setCompanies((prev) => prev.map((c) => (c.id === id ? fn(c) : c)));

  const closeGate = (runId: string, nodeId: string, dismissed: boolean): void =>
    setRuns((prev) =>
      prev.map((r) => {
        if (r.id !== runId) return r;
        const nodes = r.nodes.map((n) =>
          n.id === nodeId
            ? {
                ...n,
                status: "done" as NodeStatus,
                detail: dismissed
                  ? "Dismissed — proceeding on the most conservative assumption"
                  : "Answered",
              }
            : n.id === r.nodes.find((x) => x.id === nodeId)?.parentId
              ? { ...n, status: "running" as NodeStatus }
              : n,
        );
        const stillBlocked = nodes.some((n) => n.status === "blocked");
        return { ...r, nodes, status: stillBlocked ? r.status : "running" };
      }),
    );

  return {
    companies,
    selected,
    selectedId: selected?.id ?? "",
    select: setSelectedId,

    createCompany: (name) => {
      const created: Company = {
        id: uid(),
        name,
        slug: name.toLowerCase().replace(/[^a-z0-9]+/g, "-"),
        docs: [],
      };
      setCompanies((prev) => [...prev, created]);
      setSelectedId(created.id);
    },
    renameCompany: (id, name) => patchCompany(id, (c) => ({ ...c, name })),
    deleteCompany: (id) => {
      setCompanies((prev) => prev.filter((c) => c.id !== id));
      setSelectedId("co-aurora");
    },

    addDocument: (companyId, name) => {
      const created = doc(
        name,
        name.endsWith(".csv") ? "csv" : "markdown",
        "parsing",
        12_000,
        "today",
        null,
      );
      patchCompany(companyId, (c) => ({ ...c, docs: [...c.docs, created] }));
      const advance = (status: DocStatus, summary: Summary | null): void =>
        patchCompany(companyId, (c) => ({
          ...c,
          docs: c.docs.map((d) => (d.id === created.id ? { ...d, status, summary } : d)),
        }));
      window.setTimeout(() => advance("summarising", null), 900);
      window.setTimeout(
        () =>
          advance("ready", {
            theme: "Theme extracted by the summariser",
            pains: ["Pain identified in the document"],
            opportunities: ["Opportunity suggested from the pain"],
          }),
        2200,
      );
    },

    removeDocument: (companyId, docId) =>
      patchCompany(companyId, (c) => ({
        ...c,
        docs: c.docs.filter((d) => d.id !== docId),
      })),

    resummarise: (companyId, docId) => {
      patchCompany(companyId, (c) => ({
        ...c,
        docs: c.docs.map((d) =>
          d.id === docId ? { ...d, status: "summarising", error: undefined } : d,
        ),
      }));
      window.setTimeout(
        () =>
          patchCompany(companyId, (c) => ({
            ...c,
            docs: c.docs.map((d) =>
              d.id === docId
                ? {
                    ...d,
                    status: "ready",
                    summary: {
                      theme: "Re-summarised with the current prompt",
                      pains: ["Pain re-extracted"],
                      opportunities: ["Opportunity re-extracted"],
                    },
                  }
                : d,
            ),
          })),
        1500,
      );
    },

    runs,
    run,
    inbox,
    answerGate: (runId, nodeId) => closeGate(runId, nodeId, false),
    dismissGate: (runId, nodeId) => closeGate(runId, nodeId, true),
    startRun: (companyId, theme, opportunityCount) => {
      const co = companies.find((c) => c.id === companyId);
      setRuns((prev) => [
        ...prev.filter((r) => r.companyId !== companyId),
        {
          id: uid(),
          companyId,
          companyName: co?.name ?? "",
          theme: theme || null,
          opportunityCount,
          maxDepth: 3,
          status: "running",
          startedAt: "just now",
          nodes: [
            {
              id: uid(),
              parentId: null,
              kind: "scan_opportunities",
              status: "running",
              title: `Scanning the context — ${opportunityCount} opportunit${
                opportunityCount === 1 ? "y" : "ies"
              }`,
            },
          ],
          plan: null,
        },
      ]);
    },
  };
}

/* ── presentation helpers ────────────────────────────────────────────── */

export const fmtBytes = (n: number): string =>
  n > 1_000_000 ? `${(n / 1_000_000).toFixed(1)} MB` : `${Math.round(n / 1000)} KB`;

export const isGate = (n: RunNode): boolean =>
  n.kind === "ask_human" || n.kind === "ask_document";

export const isDiscarded = (n: RunNode): boolean => Boolean(n.discardReason);

/** impact ÷ effort — the order the LLM ranks by */
export const score = (n: RunNode): number =>
  n.impact && n.effort ? Number((n.impact / n.effort).toFixed(1)) : 0;

export const childrenOf = (nodes: RunNode[], id: string | null): RunNode[] =>
  nodes.filter((n) => n.parentId === id);

export const runStatusLabel: Record<RunStatus, string> = {
  running: "running",
  awaiting_input: "awaiting you",
  synthesizing: "consolidating",
  done: "done",
};
