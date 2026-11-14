// Oberflächentests: echtes Programm, echter Browser, echte Seiten.
const { defineConfig } = require("@playwright/test");

const port = 47899;

module.exports = defineConfig({
  testDir: "tests/ui",
  timeout: 5 * 60 * 1000,
  expect: { timeout: 20 * 1000 },
  workers: 1,
  fullyParallel: false,
  reporter: [["list"]],
  use: {
    baseURL: `http://127.0.0.1:${port}`,
    browserName: "chromium",
    headless: true,
    acceptDownloads: true,
    locale: "de-DE",
  },
  webServer: {
    command: `rm -rf tests/ui/daten && go run ./cmd/marktscout -daten tests/ui/daten -port ${port} -kein-browser -kein-tray`,
    url: `http://127.0.0.1:${port}/api/version`,
    reuseExistingServer: false,
    timeout: 180 * 1000,
  },
});
