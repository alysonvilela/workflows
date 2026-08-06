// PROTOTYPE — throwaway. The chosen design (variant E, "Cockpit") for the
// Company Context + Discovery Run screen, at /prototype-company.html.
// No API, no persistence. Tickets 03 and 14 rewrite this under production
// conditions; do not promote it as-is.
import "../global.css";
import { createRoot } from "react-dom/client";
import { useStore } from "./data";
import { VariantE } from "./VariantE";

function App(): React.ReactElement {
  const store = useStore();
  return <VariantE store={store} />;
}

createRoot(document.getElementById("app") as HTMLElement).render(<App />);
