// Plan catalog. The IDs and order here mirror the server registry in
// `internal/dailyreader/dailyreader.go`. Adding a plan is a two-line
// change: register it server-side and add an entry here.

export type PlanID = "bible-year" | "hope";

export type PlanCatalogEntry = {
  id: PlanID;
  name: string;
};

export const PLANS: PlanCatalogEntry[] = [
  { id: "bible-year", name: "Bible in One Year" },
  { id: "hope", name: "Hope (2026)" },
];

export const DEFAULT_PLAN_IDS: PlanID[] = ["bible-year"];

export function isPlanID(s: string): s is PlanID {
  return PLANS.some((p) => p.id === s);
}

export function planName(id: PlanID): string {
  return PLANS.find((p) => p.id === id)?.name ?? id;
}
