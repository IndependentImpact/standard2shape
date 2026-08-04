import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/e2e",
  fullyParallel: false,
  retries: 0,
  reporter: "line",
  use: {
    baseURL: "http://127.0.0.1:8092",
    trace: "retain-on-failure",
  },
  projects: [
    {
      name: "desktop",
      use: { ...devices["Desktop Chrome"], viewport: { width: 1440, height: 1000 } },
    },
    {
      name: "mobile",
      use: { ...devices["Pixel 5"] },
    },
  ],
  webServer: {
    command: "npm run build && go run ./cmd/standard2shape --addr 127.0.0.1:8092 --fixture fixtures/tracer --web-dir web/dist",
    url: "http://127.0.0.1:8092/api/health",
    reuseExistingServer: false,
    timeout: 120_000,
  },
});

