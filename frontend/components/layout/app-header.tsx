import { Link } from "@tanstack/react-router";
import type React from "react";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { useApiHealth } from "@/hooks/use-api-health";

const NAV_ITEMS = [
  { label: "Companies", to: "/companies" },
  { label: "Gates", to: "/gates" },
] as const;

export function AppHeader(): React.ReactElement {
  return (
    <header className="flex h-14 items-center gap-4 border-b px-4 sm:px-6">
      <Link to="/" className="font-heading text-sm font-semibold tracking-tight">
        Improvement Discovery
      </Link>
      <Separator orientation="vertical" className="h-5" />
      <nav className="flex items-center gap-1">
        {NAV_ITEMS.map((item) => (
          <Link
            key={item.to}
            to={item.to}
            className="rounded-md px-2.5 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            activeProps={{ className: "bg-accent text-foreground" }}
          >
            {item.label}
          </Link>
        ))}
      </nav>
      <span className="flex-1" />
      <ApiHealthBadge />
    </header>
  );
}

function ApiHealthBadge(): React.ReactElement {
  const health = useApiHealth();

  if (health.state === "loading") {
    return <Badge variant="outline">Checking API…</Badge>;
  }

  if (health.state === "offline") {
    return (
      <Badge variant="error" title={health.error}>
        API offline
      </Badge>
    );
  }

  if (health.data.status !== "ok") {
    return (
      <Badge
        variant="warning"
        title={`database: ${health.data.database} — storage: ${health.data.storage}`}
      >
        API degraded
      </Badge>
    );
  }

  return <Badge variant="success">API online</Badge>;
}
