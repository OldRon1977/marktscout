"use strict";

// Marktscout – Oberfläche. Alle Inhalte aus Anzeigen werden als Text gesetzt, nie als HTML.

const $ = (id) => document.getElementById(id);

const zustand = {
  auftraege: [],
  aktiv: null,          // ID des gewählten Auftrags
  bearbeite: null,      // Auftrag im Formular (Kopie)
  status: null,
  letzterJob: null,
  standardliste: [],
};

// --- Schnittstelle ---

async function api(methode, pfad, daten) {
  const opt = { method: methode, headers: {} };
  if (methode !== "GET") opt.headers["X-Marktscout"] = "1";
  if (daten !== undefined) {
    opt.headers["Content-Type"] = "application/json";
    opt.body = JSON.stringify(daten);
  }
  const r = await fetch(pfad, opt);
  const text = await r.text();
  let json = null;
  try { json = text ? JSON.parse(text) : null; } catch { /* kein JSON */ }
  if (!r.ok) throw new Error((json && json.fehler) || text || ("HTTP " + r.status));
  return json;
}

// --- Hilfen ---

function el(tag, attrs, ...kinder) {
  const e = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (v === undefined || v === null || v === false) continue;
    if (k === "class") e.className = v;
    else if (k.startsWith("on")) e.addEventListener(k.slice(2), v);
    else if (k === "text") e.textContent = v;
    else e.setAttribute(k, v === true ? "" : v);
  }
  for (const k of kinder.flat()) {
    if (k === null || k === undefined || k === false) continue;
    e.append(k instanceof Node ? k : document.createTextNode(String(k)));
  }
  return e;
}

const euro = (cent) => (cent / 100).toLocaleString("de-DE", { minimumFractionDigits: cent % 100 ? 2 : 0, maximumFractionDigits: 2 }) + " €";

function zeit(iso) {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d) || d.getFullYear() < 2000) return "";
  return d.toLocaleString("de-DE", { dateStyle: "short", timeStyle: "short" });
}

function preisText(e) {
  switch (e.preisTyp) {
    case "fest": return euro(e.preisCent);
    case "vb": return euro(e.preisCent) + " VB";
    case "vb_ohne": return "VB";
    case "verschenken": return "zu verschenken";
    default: return "ohne Preis";
  }
}

const preisMarke = {
  budget: ["im Budget", "ok"],
  toleranz: ["in Toleranz", "warn"],
  vb_spielraum: ["über VB-Spielraum", "warn"],
  ohne_grenze: null,
  ohne_preis: ["ohne Preis", ""],
  verschenken: ["zu verschenken", ""],
  darueber: ["über Obergrenze", "gefahr"],
  darunter: ["unter Mindestpreis", "gefahr"],
};

function teile(liste) {
  return (liste || []).map((t) => t.art ? el("mark", { class: t.art, title: markTitel[t.art] || "" }, t.t) : document.createTextNode(t.t));
}

const markTitel = {
  ausschluss: "Ausschlusswort",
  verneint: "Ausschlusswort, verneint",
  suchbegriff: "Suchbegriff",
  vergleich: "Suchbegriff als Vergleich",
  pflicht: "Pflichtwort",
};

function frage(text) {
  return new Promise((ok) => {
    const d = $("dlg-frage");
    $("frage-text").textContent = text;
    d.returnValue = "";
    d.addEventListener("close", () => ok(d.returnValue === "ja"), { once: true });
    d.showModal();
    $("frage-nein").focus();
  });
}

function zeige(bereich) {
  for (const id of ["leer", "ansicht", "formular"]) $(id).hidden = id !== bereich;
}

// --- Seitenleiste ---

async function ladeAuftraege() {
  zustand.auftraege = await api("GET", "/api/auftraege");
  zeichneListe();
}

function zeichneListe() {
  const ul = $("auftragsliste");
  ul.replaceChildren();
  const sortiert = [...zustand.auftraege].sort((a, b) => a.name.localeCompare(b.name, "de"));
  for (const a of sortiert) {
    const meta = [];
    meta.push(a.sichtbar + " Anzeigen");
    if (a.letzterLauf) meta.push(zeit(a.letzterLauf));
    if (a.intervallMin > 0) meta.push("alle " + a.intervallMin + " min");
    ul.append(el("li", {},
      el("button", { type: "button", "aria-current": a.id === zustand.aktiv ? "true" : "false", "data-id": a.id, onclick: () => waehle(a.id) },
        a.name, a.neu > 0 ? el("span", { class: "zahl-neu", title: "neu seit letzter Suche" }, a.neu) : null,
        el("span", { class: "meta" }, meta.join(" · ")))));
  }
}

async function waehle(id) {
  zustand.aktiv = id;
  zeichneListe();
  await zeichneAnsicht();
}

// --- Ansicht eines Auftrags ---

function zusammenfassung(a) {
  const teile = [];
  teile.push("„" + a.suchbegriff + "“");
  const orte = [...(a.orteDE || []).map((o) => o.name.split(" - ")[0] + " +" + o.radiusKm + " km"),
    ...(a.orteNL || []).map((o) => "NL " + o.postleitzahl + " +" + o.radiusKm + " km")];
  if (orte.length) teile.push(orte.join(", "));
  const p = a.preis;
  if (p.suchpreisCent > 0) {
    let s = "bis " + euro(p.suchpreisCent);
    if (p.toleranzWert > 0) s += " + " + (p.toleranzArt === "prozent" ? p.toleranzWert + " %" : euro(Math.round(p.toleranzWert * 100)));
    if (p.vbSpielraum) s += ", VB-Spielraum " + p.vbProzent + " %";
    teile.push(s);
  }
  if (p.mindestpreisCent > 0) teile.push("ab " + euro(p.mindestpreisCent));
  return teile.join(" · ");
}

async function zeichneAnsicht() {
  const id = zustand.aktiv;
  if (!id) { zeige("leer"); return; }
  let a;
  try {
    a = await api("GET", "/api/auftraege/" + id);
  } catch (e) {
    zustand.aktiv = null;
    zeige("leer");
    return;
  }
  zeige("ansicht");
  $("a-name").textContent = a.name;
  $("a-zusammenfassung").textContent = zusammenfassung(a);
  $("a-lauf").textContent = a.letzterLauf ? "Letzte Suche: " + zeit(a.letzterLauf) : "Noch nicht gesucht.";
  const fehlerText = (a.bericht && a.bericht.fehler) || a.letzterFehler || "";
  $("a-fehler").hidden = !fehlerText;
  $("a-fehler").textContent = fehlerText;
  await zeichneErgebnisse();
}

async function zeichneErgebnisse() {
  const id = zustand.aktiv;
  const mit = $("chk-ausgeblendete").checked;
  const l = await api("GET", "/api/auftraege/" + id + "/ergebnisse" + (mit ? "?ausgeblendete=1" : ""));
  if (zustand.aktiv !== id) return;
  $("zaehler").textContent = l.sichtbar + " Anzeigen" + (l.neu ? ", davon " + l.neu + " neu" : "") +
    (l.ausgeblendet ? " · " + l.ausgeblendet + " ausgeblendet" : "");
  const ziel = $("ergebnisse");
  ziel.replaceChildren();
  if (!l.gruppen || l.gruppen.length === 0) {
    ziel.append(el("p", { class: "leise-text" }, "Keine Anzeigen. „Jetzt suchen“ startet die Suche."));
    return;
  }
  for (const g of l.gruppen) {
    const sek = el("section", { class: "gruppe" },
      el("h3", {}, g.ort, el("span", { class: "leise-text" }, g.land + " · " + g.eintraege.length)));
    for (const e of g.eintraege) sek.append(karte(e));
    ziel.append(sek);
  }
}

function karte(e) {
  const marken = [];
  marken.push(el("span", { class: "marke", title: e.quelle }, e.quelle === "kleinanzeigen" ? "kleinanzeigen.de" : "marktplaats.nl"));
  if (e.neu) marken.push(el("span", { class: "marke akzent" }, "neu"));
  if (e.status === "reserviert") marken.push(el("span", { class: "marke warn" }, "reserviert"));
  if (e.status === "geloescht") marken.push(el("span", { class: "marke gefahr" }, "gelöscht"));
  const pm = preisMarke[e.preis.kennung];
  if (pm) marken.push(el("span", { class: "marke " + pm[1] }, pm[0] + (e.preis.abweichung > 0 && e.preis.kennung !== "budget" ? " +" + e.preis.abweichung.toLocaleString("de-DE") + " %" : "")));
  if (e.einstufung === "vergleich") marken.push(el("span", { class: "marke gefahr", title: "Suchbegriff nur als Vergleich genannt" }, "nur Vergleich"));
  if (e.einstufung === "zweifelhaft") marken.push(el("span", { class: "marke warn", title: "Suchbegriff nicht eindeutig angeboten" }, "zweifelhaft"));
  if (e.markiert) marken.push(el("span", { class: "marke gefahr" }, "Ausschlusswort"));
  if (e.hervorgehoben) marken.push(el("span", { class: "marke" }, "hervorgehoben"));

  const meta = [];
  meta.push([e.plz, e.ort].filter(Boolean).join(" "));
  if (e.entfernungKm >= 0) meta.push(e.entfernungKm.toLocaleString("de-DE") + " km von " + e.suchort);
  if (e.eingestellt) meta.push("eingestellt " + (e.eingestellt.includes("T") ? zeit(e.eingestellt) : e.eingestellt));
  meta.push("gefunden " + zeit(e.erstmals));
  if (e.statusGeprueft) meta.push("Status " + zeit(e.statusGeprueft));

  const text = el("div", { class: "k-text" }, teile(e.textTeile));
  const textKnopf = e.textTeile && e.textTeile.length
    ? el("button", { type: "button", class: "mini leise", onclick: (ev) => { text.classList.toggle("offen"); ev.target.textContent = text.classList.contains("offen") ? "Text einklappen" : "ganzer Text"; } }, "ganzer Text")
    : null;
  const statusKnopf = el("button", { type: "button", class: "mini", onclick: () => pruefeEine(e, statusKnopf) }, "Status prüfen");

  const klassen = ["karte"];
  if (!e.sichtbar) klassen.push("ausgeblendet");
  if (e.status === "geloescht") klassen.push("geloescht");
  return el("article", { class: klassen.join(" "), "data-quelle": e.quelle, "data-id": e.id },
    e.bildUrl ? el("img", { src: e.bildUrl, alt: "", loading: "lazy", referrerpolicy: "no-referrer" }) : el("div"),
    el("div", {},
      el("div", { class: "k-titel" }, el("a", { href: e.url, target: "_blank", rel: "noopener noreferrer" }, teile(e.titelTeile))),
      el("div", { class: "marken" }, marken),
      el("div", { class: "k-meta" }, meta.join(" · ")),
      e.gruende && e.gruende.length ? el("div", { class: "gruende" }, "Ausgeblendet: " + e.gruende.join("; ")) : null,
      text, textKnopf),
    el("div", { class: "k-rechts" },
      el("span", { class: "preis" }, preisText(e)),
      statusKnopf,
      el("button", { type: "button", class: "mini gefahr", onclick: () => verwerfe(e) }, "Löschen")));
}

async function pruefeEine(e, knopf) {
  knopf.disabled = true;
  knopf.textContent = "prüfe …";
  try {
    const r = await api("POST", `/api/auftraege/${zustand.aktiv}/treffer/${e.quelle}/${encodeURIComponent(e.id)}/pruefen`);
    knopf.textContent = { aktiv: "aktiv", reserviert: "reserviert", geloescht: "gelöscht" }[r.status] || r.status;
    await zeichneErgebnisse();
  } catch (err) {
    knopf.textContent = "Fehler";
    knopf.title = err.message;
    knopf.disabled = false;
  }
}

async function verwerfe(e) {
  if (!(await frage("Anzeige „" + e.titel + "“ löschen? Sie erscheint bei späteren Suchen nicht wieder."))) return;
  await api("DELETE", `/api/auftraege/${zustand.aktiv}/treffer/${e.quelle}/${encodeURIComponent(e.id)}`);
  await Promise.all([zeichneErgebnisse(), ladeAuftraege()]);
}

// --- Formular ---

function neuerAuftrag() {
  return {
    id: 0, name: "", suchbegriff: "", orteDE: [], orteNL: [], pflicht: [], ausschluss: [],
    ausschlussModus: "titel", standardAktiv: true, standardModus: "titel",
    preis: { suchpreisCent: 0, toleranzArt: "prozent", toleranzWert: 0, mindestpreisCent: 0, vbSpielraum: false, vbProzent: 10 },
    intervallMin: 0, maxSeiten: 4,
  };
}

function oeffneFormular(a) {
  zustand.bearbeite = JSON.parse(JSON.stringify(a));
  const b = zustand.bearbeite;
  $("f-titel").textContent = b.id ? "Suchauftrag bearbeiten" : "Neuer Suchauftrag";
  $("f-fehler").hidden = true;
  $("f-name").value = b.name;
  $("f-suchbegriff").value = b.suchbegriff;
  $("f-suchpreis").value = b.preis.suchpreisCent ? b.preis.suchpreisCent / 100 : "";
  $("f-toleranz").value = b.preis.toleranzWert || 0;
  $("f-toleranzart").value = b.preis.toleranzArt || "prozent";
  $("f-mindestpreis").value = b.preis.mindestpreisCent ? b.preis.mindestpreisCent / 100 : "";
  $("f-vb").checked = !!b.preis.vbSpielraum;
  $("f-vbprozent").value = b.preis.vbProzent || 10;
  $("f-pflicht").value = (b.pflicht || []).join(", ");
  $("f-ausschluss").value = (b.ausschluss || []).join(", ");
  $("f-ausschlussmodus").value = b.ausschlussModus || "titel";
  $("f-standard").checked = !!b.standardAktiv;
  $("f-standardmodus").value = b.standardModus || "titel";
  $("f-intervall").value = b.intervallMin || 0;
  $("f-seiten").value = b.maxSeiten || 4;
  $("f-ort-eingabe").value = "";
  $("f-plz").value = "";
  zeichneOrte();
  preisInfo();
  zeige("formular");
  $("f-suchbegriff").focus();
}

function zeichneOrte() {
  const b = zustand.bearbeite;
  const de = $("f-orte-de");
  de.replaceChildren(...b.orteDE.map((o, i) => el("li", {},
    o.name,
    el("input", { type: "number", min: 1, max: 200, value: o.radiusKm, "aria-label": "Umkreis km " + o.name, onchange: (ev) => { o.radiusKm = parseInt(ev.target.value, 10) || 0; } }),
    "km",
    el("button", { type: "button", "aria-label": "Ort entfernen " + o.name, onclick: () => { b.orteDE.splice(i, 1); zeichneOrte(); } }, "×"))));
  const nl = $("f-orte-nl");
  nl.replaceChildren(...b.orteNL.map((o, i) => el("li", {},
    "NL " + o.postleitzahl,
    el("input", { type: "number", min: 1, max: 200, value: o.radiusKm, "aria-label": "Umkreis km " + o.postleitzahl, onchange: (ev) => { o.radiusKm = parseInt(ev.target.value, 10) || 0; } }),
    "km",
    el("button", { type: "button", "aria-label": "Postleitzahl entfernen " + o.postleitzahl, onclick: () => { b.orteNL.splice(i, 1); zeichneOrte(); } }, "×"))));
}

function preisInfo() {
  const such = parseFloat($("f-suchpreis").value) || 0;
  const tol = parseFloat($("f-toleranz").value) || 0;
  const art = $("f-toleranzart").value;
  if (!such) { $("f-preisinfo").textContent = "Ohne Suchpreis gibt es keine Obergrenze."; return; }
  const ober = art === "prozent" ? such * (1 + tol / 100) : such + tol;
  let t = "Obergrenze " + euro(Math.round(ober * 100));
  if ($("f-vb").checked) {
    const vb = parseFloat($("f-vbprozent").value) || 0;
    if (vb > 0 && vb < 100) t += "; VB-Anzeigen bis " + euro(Math.round(ober / (1 - vb / 100) * 100));
  }
  $("f-preisinfo").textContent = t + ".";
}

function liste(text) {
  return text.split(/[,;\n]/).map((s) => s.trim()).filter(Boolean);
}

function leseFormular() {
  const b = zustand.bearbeite;
  b.name = $("f-name").value.trim();
  b.suchbegriff = $("f-suchbegriff").value.trim();
  const cent = (v) => Math.round((parseFloat(v) || 0) * 100);
  b.preis = {
    suchpreisCent: cent($("f-suchpreis").value),
    toleranzArt: $("f-toleranzart").value,
    toleranzWert: parseFloat($("f-toleranz").value) || 0,
    mindestpreisCent: cent($("f-mindestpreis").value),
    vbSpielraum: $("f-vb").checked,
    vbProzent: parseFloat($("f-vbprozent").value) || 0,
  };
  b.pflicht = liste($("f-pflicht").value);
  b.ausschluss = liste($("f-ausschluss").value);
  b.ausschlussModus = $("f-ausschlussmodus").value;
  b.standardAktiv = $("f-standard").checked;
  b.standardModus = $("f-standardmodus").value;
  b.intervallMin = parseInt($("f-intervall").value, 10) || 0;
  b.maxSeiten = parseInt($("f-seiten").value, 10) || 1;
  return b;
}

async function speichere(ev) {
  ev.preventDefault();
  const b = leseFormular();
  try {
    const g = b.id ? await api("PUT", "/api/auftraege/" + b.id, b) : await api("POST", "/api/auftraege", b);
    zustand.aktiv = g.id;
    await ladeAuftraege();
    await zeichneAnsicht();
  } catch (e) {
    $("f-fehler").textContent = e.message;
    $("f-fehler").hidden = false;
    $("f-fehler").scrollIntoView({ block: "nearest" });
  }
}

// Ortsvorschläge (kleinanzeigen.de)
let vorschlagZeit = null;
let vorschlagNr = 0;
let vorschlagAuswahl = -1;

function schliesseVorschlaege() {
  $("f-ort-vorschlaege").hidden = true;
  $("f-ort-vorschlaege").replaceChildren();
  vorschlagAuswahl = -1;
}

function uebernehmeOrt(o) {
  const b = zustand.bearbeite;
  const radius = parseInt($("f-ort-radius").value, 10) || 10;
  if (!b.orteDE.some((x) => x.id === o.id)) b.orteDE.push({ id: o.id, name: o.name, radiusKm: radius });
  $("f-ort-eingabe").value = "";
  schliesseVorschlaege();
  zeichneOrte();
  $("f-ort-eingabe").focus();
}

async function frageOrte() {
  const q = $("f-ort-eingabe").value.trim();
  const nr = ++vorschlagNr;
  if (q.length < 2) { schliesseVorschlaege(); return; }
  let orte = [];
  try { orte = await api("GET", "/api/orte?q=" + encodeURIComponent(q)); } catch { orte = []; }
  if (nr !== vorschlagNr) return;
  const ul = $("f-ort-vorschlaege");
  ul.replaceChildren(...orte.slice(0, 12).map((o, i) => el("li", {
    role: "option", id: "vorschlag-" + i, "data-id": o.id, "aria-selected": "false",
    onmousedown: (ev) => { ev.preventDefault(); uebernehmeOrt(o); },
  }, o.name)));
  ul.hidden = orte.length === 0;
  if (orte.length === 0) ul.replaceChildren();
  ul._orte = orte.slice(0, 12);
  vorschlagAuswahl = -1;
}

function vorschlagTaste(ev) {
  const ul = $("f-ort-vorschlaege");
  const orte = ul._orte || [];
  if (ul.hidden || orte.length === 0) {
    if (ev.key === "Enter") ev.preventDefault();
    return;
  }
  if (ev.key === "ArrowDown" || ev.key === "ArrowUp") {
    ev.preventDefault();
    vorschlagAuswahl = (vorschlagAuswahl + (ev.key === "ArrowDown" ? 1 : -1) + orte.length) % orte.length;
    [...ul.children].forEach((li, i) => li.setAttribute("aria-selected", i === vorschlagAuswahl ? "true" : "false"));
  } else if (ev.key === "Enter") {
    ev.preventDefault();
    uebernehmeOrt(orte[Math.max(0, vorschlagAuswahl)]);
  } else if (ev.key === "Escape") {
    schliesseVorschlaege();
  }
}

function fuegePlzHinzu() {
  const b = zustand.bearbeite;
  const plz = $("f-plz").value.replace(/\s+/g, "").toUpperCase();
  const radius = parseInt($("f-plz-radius").value, 10) || 10;
  if (!/^[1-9][0-9]{3}([A-Z]{2})?$/.test(plz)) {
    $("f-fehler").textContent = "„" + $("f-plz").value + "“ ist keine niederländische Postleitzahl (vier Ziffern, z. B. 5911).";
    $("f-fehler").hidden = false;
    return;
  }
  $("f-fehler").hidden = true;
  if (!b.orteNL.some((o) => o.postleitzahl === plz)) b.orteNL.push({ postleitzahl: plz, radiusKm: radius });
  $("f-plz").value = "";
  zeichneOrte();
}

// --- Status und Aktionen ---

async function aktualisiereStatus() {
  let st;
  try { st = await api("GET", "/api/status"); } catch { return; }
  zustand.status = st;
  const a = st.aktuell;
  const text = a ? `${a.art === "suche" ? "Suche" : "Statusprüfung"} „${a.name}“: ${a.text}` + (st.warte.length ? ` · ${st.warte.length} wartend` : "") : "";
  $("jobstatus").textContent = text;
  const laeuft = a && a.auftragId === zustand.aktiv;
  $("btn-suchen").disabled = !!(laeuft && a.art === "suche") || st.warte.some((j) => j.auftragId === zustand.aktiv && j.art === "suche");
  $("btn-status").disabled = !!(laeuft && a.art === "status") || st.warte.some((j) => j.auftragId === zustand.aktiv && j.art === "status");
  const schluessel = a ? a.art + a.auftragId : null;
  if (zustand.letzterJob && zustand.letzterJob !== schluessel) {
    // Ein Job ist fertig: Listen neu laden
    await ladeAuftraege();
    if (!$("ansicht").hidden) await zeichneAnsicht();
  }
  zustand.letzterJob = schluessel;
}

async function starte(art) {
  if (!zustand.aktiv) return;
  await api("POST", "/api/auftraege/" + zustand.aktiv + (art === "suche" ? "/suchen" : "/pruefen"));
  await aktualisiereStatus();
}

async function oeffneEinstellungen() {
  const e = await api("GET", "/api/einstellungen");
  $("e-zeit").checked = e.zeitsteuerung;
  $("e-desktop").checked = e.desktop;
  $("e-ton").checked = e.ton;
  const v = await api("GET", "/api/version");
  $("e-info").textContent = "Version " + v.version;
  $("dlg-einstellungen").showModal();
}

async function speichereEinstellungen() {
  await api("PUT", "/api/einstellungen", {
    zeitsteuerung: $("e-zeit").checked, desktop: $("e-desktop").checked, ton: $("e-ton").checked,
  });
}

// --- Start ---

function verbinde() {
  $("btn-neu").addEventListener("click", () => oeffneFormular(neuerAuftrag()));
  $("btn-abbrechen").addEventListener("click", () => { schliesseVorschlaege(); zustand.aktiv ? zeichneAnsicht() : zeige("leer"); });
  $("auftragsform").addEventListener("submit", speichere);
  $("btn-bearbeiten").addEventListener("click", async () => oeffneFormular(await api("GET", "/api/auftraege/" + zustand.aktiv)));
  $("btn-loeschen").addEventListener("click", async () => {
    const a = zustand.auftraege.find((x) => x.id === zustand.aktiv);
    if (!a || !(await frage("Suchauftrag „" + a.name + "“ mit allen gefundenen Anzeigen löschen?"))) return;
    await api("DELETE", "/api/auftraege/" + a.id);
    zustand.aktiv = null;
    await ladeAuftraege();
    zeige("leer");
  });
  $("btn-suchen").addEventListener("click", () => starte("suche"));
  $("btn-status").addEventListener("click", () => starte("status"));
  $("chk-ausgeblendete").addEventListener("change", zeichneErgebnisse);
  $("f-ort-eingabe").addEventListener("input", () => { clearTimeout(vorschlagZeit); vorschlagZeit = setTimeout(frageOrte, 350); });
  $("f-ort-eingabe").addEventListener("keydown", vorschlagTaste);
  $("f-ort-eingabe").addEventListener("blur", () => setTimeout(schliesseVorschlaege, 150));
  $("btn-plz-add").addEventListener("click", fuegePlzHinzu);
  $("f-plz").addEventListener("keydown", (ev) => { if (ev.key === "Enter") { ev.preventDefault(); fuegePlzHinzu(); } });
  for (const id of ["f-suchpreis", "f-toleranz", "f-toleranzart", "f-vb", "f-vbprozent"]) $(id).addEventListener("input", preisInfo);
  $("btn-einstellungen").addEventListener("click", oeffneEinstellungen);
  for (const id of ["e-zeit", "e-desktop", "e-ton"]) $(id).addEventListener("change", speichereEinstellungen);
  $("btn-meldungtest").addEventListener("click", async () => {
    const r = await api("POST", "/api/meldung-test");
    $("e-info").textContent = r.ok ? "Meldung und Ton ausgegeben." : r.fehler.join("; ");
  });
  $("btn-beenden").addEventListener("click", async () => {
    if (!(await frage("Marktscout beenden? Die Zeitsteuerung läuft dann nicht mehr."))) return;
    try { await api("POST", "/api/beenden"); } catch { /* Server weg */ }
    $("beendet").hidden = false;
  });
}

async function start() {
  verbinde();
  try { zustand.standardliste = await api("GET", "/api/standardliste"); } catch { zustand.standardliste = []; }
  $("f-standardliste").textContent = "Standardliste: " + zustand.standardliste.join(", ");
  await ladeAuftraege();
  if (zustand.auftraege.length) await waehle([...zustand.auftraege].sort((a, b) => a.name.localeCompare(b.name, "de"))[0].id);
  await aktualisiereStatus();
  setInterval(aktualisiereStatus, 2000);
}

start();
