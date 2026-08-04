import type { Snapshot } from "./types";

async function decode(response: Response): Promise<Snapshot> {
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({ error: response.statusText }))) as { error?: string };
    throw new Error(payload.error ?? `Request failed with ${response.status}`);
  }
  return (await response.json()) as Snapshot;
}

export async function loadWorkspace(): Promise<Snapshot> {
  return decode(await fetch("/api/workspace", { headers: { Accept: "application/json" } }));
}

export async function saveGuidance(guidance: string): Promise<Snapshot> {
  return decode(
    await fetch("/api/guidance", {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ guidance }),
    }),
  );
}
