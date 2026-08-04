import { expect, test } from "@playwright/test";

test("opens, validates, previews, changes, patches, and reloads the local bundle", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") errors.push(message.text());
  });
  page.on("pageerror", (error) => errors.push(error.message));

  await page.goto("/");

  await expect(page.getByRole("heading", { name: "Project design document", level: 1 })).toBeVisible();
  const navigation = page.getByLabel("Bundle navigation");
  await expect(navigation.getByText("1. Project overview", { exact: true })).toBeVisible();
  await expect(navigation.getByText("Project details", { exact: true })).toBeVisible();
  const sources = navigation.locator(".source-list");
  await expect(sources.getByText("document.ttl", { exact: true })).toBeVisible();
  await expect(sources.getByText("shapes.ttl", { exact: true })).toBeVisible();
  await expect(sources.getByText("references.ttl", { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "shape2form preview" })).toBeVisible();
  await expect(page.getByText("2/2 expected", { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "SHACL-SPARQL" })).toBeVisible();

  const nextGuidance = "Explain the project purpose, location, and intended outcomes for an independent reviewer.";
  const guidance = page.getByLabel("Canonical guidance");
  await guidance.fill(nextGuidance);
  await expect(page.getByText("Unsaved semantic change")).toBeVisible();
  await page.getByRole("button", { name: "Save canonical guidance" }).click();

  await expect(page.getByText("Saved as a source patch and reloaded successfully.")).toBeVisible();
  await expect(guidance).toHaveValue(nextGuidance);
  await expect(page.getByText("Unsupported graph preserved")).toBeVisible();
  await expect(page.getByTestId("semantic-diff")).toContainText("Canonical guidance changed");
  await expect(page.getByTestId("semantic-diff")).toContainText(nextGuidance);
  await expect(page.getByTestId("source-patch")).toContainText("--- a/document.ttl");
  await expect(page.getByTestId("source-patch")).toContainText("+  s2s:canonicalGuidance");

  const apiState = await page.request.get("/api/workspace");
  expect(apiState.ok()).toBeTruthy();
  const payload = (await apiState.json()) as { document: { sections: Array<{ guidance: string }> } };
  expect(payload.document.sections[0]?.guidance).toBe(nextGuidance);
  expect(errors).toEqual([]);
});
