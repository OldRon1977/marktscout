// Oberflächentests gegen das laufende Programm. Die Suche läuft live gegen
// kleinanzeigen.de und marktplaats.nl.
const { test, expect } = require("@playwright/test");
const fs = require("fs");

test.describe.configure({ mode: "serial" });

let seite;

test.beforeAll(async ({ browser }) => {
  seite = await browser.newPage();
  await seite.goto("/");
});

test.afterAll(async () => {
  await seite.close();
});

test("Startseite", async () => {
  await expect(seite).toHaveTitle("Marktscout");
  await expect(seite.getByRole("heading", { name: "Willkommen" })).toBeVisible();
  await expect(seite.getByRole("button", { name: "+ Neuer Suchauftrag" })).toBeVisible();
});

test("Suchauftrag anlegen mit Ortsvorschlägen (Maus und Tastatur)", async () => {
  await seite.getByRole("button", { name: "+ Neuer Suchauftrag" }).click();
  await expect(seite.getByRole("heading", { name: "Neuer Suchauftrag" })).toBeVisible();
  await seite.getByLabel("Name", { exact: true }).fill("RTX Test");
  await seite.getByLabel("Suchbegriff").fill("rtx");

  // Maus: Vorschlag anklicken
  const ort = seite.getByLabel("Ort oder Postleitzahl");
  await ort.fill("Köln");
  const koeln = seite.locator("#f-ort-vorschlaege li", { hasText: "Köln - Nordrhein-Westfalen" });
  await expect(koeln).toBeVisible();
  await koeln.click();
  await expect(seite.locator("#f-orte-de li")).toHaveCount(1);
  await expect(seite.locator("#f-orte-de li").first()).toContainText("Köln");

  // Tastatur: Pfeil runter, Enter
  await ort.fill("41372");
  await expect(seite.locator("#f-ort-vorschlaege li").first()).toContainText("Niederkrüchten");
  await ort.press("ArrowDown");
  await ort.press("Enter");
  await expect(seite.locator("#f-orte-de li")).toHaveCount(2);
  await expect(seite.locator("#f-orte-de li").nth(1)).toContainText("Niederkrüchten");

  // NL-Postleitzahl
  await seite.getByLabel("Postleitzahl", { exact: true }).fill("6041");
  await seite.getByRole("button", { name: "Hinzufügen" }).click();
  await expect(seite.locator("#f-orte-nl li")).toContainText("NL 6041");

  // Preis: enges Fenster, damit wenige Detailseiten geladen werden
  await seite.getByLabel("Suchpreis €").fill("150");
  await seite.getByLabel("Toleranz", { exact: true }).fill("10");
  await seite.getByLabel("Mindestpreis €").fill("120");
  await expect(seite.locator("#f-preisinfo")).toContainText("Obergrenze 165 €");
  await seite.getByLabel("Seiten je Ort").fill("1");

  await seite.getByRole("button", { name: "Speichern" }).click();
  await expect(seite.locator("#a-name")).toHaveText("RTX Test");
  await expect(seite.locator("#a-zusammenfassung")).toContainText("NL 6041");
  await expect(seite.locator("#auftragsliste")).toContainText("RTX Test");
});

test("Ungültige niederländische Postleitzahl wird abgelehnt", async () => {
  await seite.getByRole("button", { name: "Bearbeiten" }).click();
  await seite.getByLabel("Postleitzahl", { exact: true }).fill("41372");
  await seite.getByRole("button", { name: "Hinzufügen" }).click();
  await expect(seite.locator("#f-fehler")).toContainText("keine niederländische Postleitzahl");
  await expect(seite.locator("#f-orte-nl li")).toHaveCount(1);
  await seite.getByRole("button", { name: "Abbrechen" }).click();
  await expect(seite.locator("#a-name")).toHaveText("RTX Test");
});

test("Suche ausführen (live)", async () => {
  await seite.getByRole("button", { name: "Jetzt suchen" }).click();
  await expect(seite.locator("#jobstatus")).toContainText("Suche „RTX Test“");
  await expect(seite.locator("#jobstatus")).toHaveText("", { timeout: 4 * 60 * 1000 });
  await expect(seite.locator("#a-lauf")).toContainText("Letzte Suche");
  await expect(seite.locator("#a-fehler")).toBeHidden();
  const karten = seite.locator("#ergebnisse article.karte");
  await expect(karten.first()).toBeVisible();
  expect(await karten.count()).toBeGreaterThan(0);
  await expect(seite.locator("#ergebnisse .gruppe h3").first()).toBeVisible();
  // Preis innerhalb des Fensters oder ohne Betrag
  for (const p of await seite.locator("#ergebnisse .preis").allTextContents()) {
    const m = p.match(/^([\d.]+)(?:,\d+)? €/);
    if (m) {
      const euro = parseInt(m[1].replace(/\./g, ""), 10);
      expect(euro).toBeGreaterThanOrEqual(120);
      expect(euro).toBeLessThanOrEqual(184); // 165 € Obergrenze, VB-Spielraum aus; Seite rundet nicht
    }
  }
  const link = karten.first().locator(".k-titel a");
  expect(await link.getAttribute("href")).toMatch(/^https:\/\/www\.(kleinanzeigen\.de|marktplaats\.nl)\//);
  await expect(seite.locator("#auftragsliste .zahl-neu")).toBeVisible();
});

test("Ausgeblendete zeigen", async () => {
  const vorher = await seite.locator("#ergebnisse article.karte").count();
  await seite.getByLabel("Ausgeblendete zeigen").check();
  await expect.poll(async () => seite.locator("#ergebnisse article.karte").count()).toBeGreaterThanOrEqual(vorher);
  const ausgeblendet = seite.locator("#ergebnisse article.karte.ausgeblendet");
  if (await ausgeblendet.count()) {
    await expect(ausgeblendet.first().locator(".gruende")).toContainText("Ausgeblendet:");
  }
  await seite.getByLabel("Ausgeblendete zeigen").uncheck();
  await expect(seite.locator("#ergebnisse article.karte")).toHaveCount(vorher);
});

test("Status einer Anzeige prüfen", async () => {
  const karte = seite.locator("#ergebnisse article.karte").first();
  const id = await karte.getAttribute("data-id");
  const quelle = await karte.getAttribute("data-quelle");
  const [antwort] = await Promise.all([
    seite.waitForResponse((r) => r.url().endsWith(`/treffer/${quelle}/${id}/pruefen`), { timeout: 90 * 1000 }),
    karte.getByRole("button", { name: "Status prüfen" }).click(),
  ]);
  expect(antwort.status()).toBe(200);
  const { status } = await antwort.json();
  expect(["aktiv", "reserviert", "geloescht"]).toContain(status);
  // Neu gezeichnete Karte trägt den Status
  const neu = seite.locator(`#ergebnisse article.karte[data-id="${id}"]`);
  if (status === "geloescht") await expect(neu.locator(".marke.gefahr", { hasText: "gelöscht" })).toBeVisible();
  if (status === "reserviert") await expect(neu.locator(".marke", { hasText: "reserviert" })).toBeVisible();
  if (status === "aktiv") await expect(neu.locator(".marke", { hasText: /gelöscht|reserviert/ })).toHaveCount(0);
});

test("Anzeige löschen", async () => {
  const karten = seite.locator("#ergebnisse article.karte");
  const vorher = await karten.count();
  const erste = karten.first();
  const id = await erste.getAttribute("data-id");
  await erste.getByRole("button", { name: "Löschen" }).click();
  await expect(seite.locator("#dlg-frage")).toBeVisible();
  await seite.locator("#frage-ja").click();
  await expect(karten).toHaveCount(vorher - 1);
  await expect(seite.locator(`#ergebnisse article.karte[data-id="${id}"]`)).toHaveCount(0);
});

test("Suchauftrag bearbeiten", async () => {
  await seite.getByRole("button", { name: "Bearbeiten" }).click();
  await expect(seite.getByLabel("Suchbegriff")).toHaveValue("rtx");
  await expect(seite.locator("#f-orte-de li")).toHaveCount(2);
  await seite.getByLabel("Name", { exact: true }).fill("RTX Test geändert");
  await seite.getByLabel("Pflichtwörter (durch Komma getrennt)").fill("rtx");
  await seite.getByRole("button", { name: "Speichern" }).click();
  await expect(seite.locator("#a-name")).toHaveText("RTX Test geändert");
  await expect(seite.locator("#auftragsliste")).toContainText("RTX Test geändert");
});

test("Einstellungen bleiben gespeichert, Logdatei herunterladen", async () => {
  await seite.getByRole("button", { name: "Einstellungen" }).click();
  const ton = seite.getByLabel("Ton bei neuen Anzeigen");
  await expect(ton).toBeChecked();
  await ton.uncheck();
  await seite.waitForTimeout(300);

  const [download] = await Promise.all([
    seite.waitForEvent("download"),
    seite.getByRole("link", { name: "Logdatei speichern" }).click(),
  ]);
  expect(download.suggestedFilename()).toMatch(/^marktscout-log-\d{4}-\d{2}-\d{2}-\d{4}\.txt$/);
  const inhalt = fs.readFileSync(await download.path(), "utf8");
  expect(inhalt).toContain("suchlauf ende");

  await seite.getByRole("button", { name: "Schließen" }).click();
  await seite.reload();
  await seite.getByRole("button", { name: "Einstellungen" }).click();
  await expect(seite.getByLabel("Ton bei neuen Anzeigen")).not.toBeChecked();
  await expect(seite.getByLabel("Zeitsteuerung")).toBeChecked();
  await seite.getByRole("button", { name: "Schließen" }).click();
});

test("Suchauftrag löschen", async () => {
  await seite.locator("#auftragsliste button", { hasText: "RTX Test geändert" }).click();
  await seite.locator("#btn-loeschen").click();
  await seite.locator("#frage-ja").click();
  await expect(seite.getByRole("heading", { name: "Willkommen" })).toBeVisible();
  await expect(seite.locator("#auftragsliste li")).toHaveCount(0);
});
