#!/usr/bin/env python3
"""Scrape the historical VTD (Vector de Tasas de Descuento) from the CMF.

The VTD for policy valuation (NCG 374, reservas de rentas vitalicias) from
jun-2015 to ago-2020 was published ONLY as monthly "Oficios Circulares" in
scanned PDFs. There is no consolidated file for that range (the consolidated
xlsx at articles-51926 only covers 2020-09 onward).

This scraper:
  1. Queries the CMF normativa search page for "vector de tasas de descuento".
  2. Filters the Oficios Circulares for "VTD ... VALORIZACION DE PASIVOS
     (NCG N° 374" (excludes the TSA/suficiencia-de-activos series).
  3. Downloads each scanned PDF and runs OCR (tesseract) on the annex page.
  4. Extracts the month of application from the cover page ("correspondiente al
     mes de X de YYYY") and the 25-period curve from the annex.
  5. Writes a CSV (year, month, period, rate) with rate as a decimal fraction,
     matching the vtd_vector table convention.

Dependencies: requests, beautifulsoup4, pdftoppm, tesseract.
Usage:  python3 scripts/scrape_vtd_cmf.py [--out data/vtd_historico.csv] [--cache /tmp/vtd_oficios]
"""

import argparse
import csv
import os
import re
import subprocess
import sys
import tempfile

import requests
from bs4 import BeautifulSoup

SEARCH_URL = (
    "https://www.cmfchile.cl/institucional/legislacion_normativa/normativa2.php"
    "?buscar=vector+de+tasas+de+descuento&tiponorma=OFC&enviado=1"
)
HEADERS = {"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"}

MESES = {
    "enero": 1, "febrero": 2, "marzo": 3, "abril": 4, "mayo": 5, "junio": 6,
    "julio": 7, "agosto": 8, "septiembre": 9, "octubre": 10, "noviembre": 11,
    "diciembre": 12,
}


def fetch_search_results():
    r = requests.get(SEARCH_URL, headers=HEADERS, timeout=60)
    r.raise_for_status()
    soup = BeautifulSoup(r.text, "html.parser")
    oficios = []
    for tr in soup.find_all("tr"):
        tds = tr.find_all("td")
        if len(tds) < 5:
            continue
        if not tds[0].get_text(strip=True) == "OFC":
            continue
        a = tds[1].find("a")
        if not a or "ofc_" not in a.get("href", ""):
            continue
        ref = tds[3].get_text(" ", strip=True)
        if "vector de tasas de descuento" not in ref.lower():
            continue
        if "TSA" in ref.upper() or "SUFICIENCIA DE ACTIVOS" in ref.upper():
            continue
        if "NCG N° 374" not in ref.upper() and "NCG N°374" not in ref.upper():
            continue
        href = a["href"]
        if href.startswith("../../"):
            href = "https://www.cmfchile.cl" + href[5:]
        elif href.startswith("../"):
            href = "https://www.cmfchile.cl/institucional/legislacion_normativa/" + href[2:]
        m = re.search(r"ofc_(\d+)_(\d+)\.pdf", href)
        if not m:
            continue
        oficios.append({
            "num": int(m.group(1)),
            "year": int(m.group(2)),
            "url": href,
            "ref": ref,
        })
    oficios.sort(key=lambda o: (o["year"], o["num"]))
    return oficios


def download_pdf(url, cache_dir):
    os.makedirs(cache_dir, exist_ok=True)
    m = re.search(r"ofc_(\d+)_(\d+)\.pdf", url)
    local = os.path.join(cache_dir, f"ofc_{m.group(1)}_{m.group(2)}.pdf")
    if os.path.exists(local) and os.path.getsize(local) > 0:
        return local
    r = requests.get(url, headers=HEADERS, timeout=60)
    r.raise_for_status()
    with open(local, "wb") as f:
        f.write(r.content)
    return local


def ocr_pdf(pdf_path):
    """Return OCR text per page of a scanned PDF."""
    texts = []
    with tempfile.TemporaryDirectory() as tmp:
        subprocess.run(
            ["pdftoppm", "-r", "200", "-png", pdf_path, os.path.join(tmp, "p")],
            check=True, capture_output=True,
        )
        pages = sorted(f for f in os.listdir(tmp) if f.endswith(".png"))
        for page in pages:
            base = os.path.join(tmp, page[:-4])  # sin extensión .png
            # psm 6: block de texto uniforme — funciona mejor con las tablas del
            # anexo (default psm 3 pierde filas).
            subprocess.run(["tesseract", base + ".png", base, "--psm", "6"],
                           check=True, capture_output=True)
            with open(base + ".txt") as f:
                texts.append(f.read())
    return texts


def extract_month(text):
    """Return (year, month) of the VTD application month, e.g. (2015, 6)."""
    m = re.search(
        r"correspondiente(?:s)?\s+al?\s+mes de\s*(enero|febrero|marzo|abril|mayo|"
        r"junio|julio|agosto|septiembre|octubre|noviembre|diciembre)\s*de\s*(\d{4})",
        text, re.I,
    )
    if m:
        return int(m.group(2)), MESES[m.group(1).lower()]
    m2 = re.search(r"correspondiente(?:s)?\s+al?\s+mes de\s*(\d{1,2})\s*de\s*(\d{4})", text, re.I)
    if m2:
        return int(m2.group(2)), int(m2.group(1))
    return None


def parse_annex(text):
    """Parse the 'Curva Cero 80% AAA (VTD)' table: lines with 'period value'."""
    curve = {}
    for line in text.splitlines():
        line = line.strip().replace("\u00a0", " ")
        # Anexo format: "1 1,40", "1 -0,76" or "1 1.40"
        m = re.match(r"^(\d{1,3})\s+(-?\d{1,2}[.,]\d{2,4})$", line)
        if m:
            period = int(m.group(1))
            val = float(m.group(2).replace(",", "."))
            curve[period] = val
    return curve


def interp_curve(curve, n_periods=25):
    """Fill gaps in a (period -> value) curve with linear interpolation. The
    curve is smooth and monotonic, so linear interpolation between neighbours
    is a faithful reconstruction of the odd OCR-missed value."""
    full = {}
    periods = sorted(p for p in curve if 1 <= p <= n_periods)
    if not periods:
        return full
    for p in range(1, n_periods + 1):
        if p in curve:
            full[p] = curve[p]
            continue
        # lower neighbour (p-1 if present, else first below)
        lo = [q for q in periods if q < p]
        hi = [q for q in periods if q > p]
        if not lo and hi:
            full[p] = curve[hi[0]]  # extrapolación al inicio
        elif not hi and lo:
            full[p] = curve[lo[-1]]  # extrapolación al final
        elif lo and hi:
            a, b = lo[-1], hi[0]
            # interpolar linealmente en el índice del período
            t = (p - a) / (b - a)
            full[p] = round(curve[a] + t * (curve[b] - curve[a]), 6)
    return full


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", default="data/vtd_historico.csv")
    ap.add_argument("--cache", default="/tmp/vtd_oficios")
    ap.add_argument("--limit", type=int, default=0, help="0 = all")
    args = ap.parse_args()

    print("Consultando buscador normativa CMF...")
    oficios = fetch_search_results()
    print(f"Oficios VTD NCG 374 encontrados: {len(oficios)}")
    if args.limit:
        oficios = oficios[: args.limit]

    rows = []
    for i, of in enumerate(oficios, 1):
        pdf = download_pdf(of["url"], args.cache)
        try:
            texts = ocr_pdf(pdf)
        except Exception as e:
            print(f"  [{i}/{len(oficios)}] ofc_{of['num']}_{of['year']}: OCR fail: {e}")
            continue
        full = "\n".join(texts)
        mes_apl = extract_month(full)
        curve = parse_annex(texts[1] if len(texts) > 1 else full)
        if not mes_apl or not curve:
            print(f"  [{i}/{len(oficios)}] ofc_{of['num']}_{of['year']}: "
                  f"mes={mes_apl} valores={len(curve)} — OMITIDO")
            continue
        y, ym = mes_apl
        rows.append((y, ym, of["num"], of["year"], len(curve)))
        print(f"  [{i}/{len(oficios)}] ofc_{of['num']}_{of['year']}: "
              f"mes {ym}/{y}, {len(curve)} valores")

    with open(args.out, "w", newline="") as f:
        w = csv.writer(f)
        w.writerow(["year", "month", "period", "rate"])
        n_total = 0
        for (y, ym, num, yr, nvals) in sorted(rows):
            pdf = download_pdf(
                f"https://www.cmfchile.cl/normativa/ofc_{num}_{yr}.pdf", args.cache)
            try:
                texts = ocr_pdf(pdf)
            except Exception:
                continue
            full = "\n".join(texts)
            curve = parse_annex(texts[1] if len(texts) > 1 else full)
            curve = interp_curve(curve)
            for period in sorted(curve):
                w.writerow([y, ym, period, round(curve[period] / 100.0, 6)])
                n_total += 1
        print(f"CSV escrito en {args.out} ({n_total} puntos, {len(rows)} meses)")


if __name__ == "__main__":
    main()
