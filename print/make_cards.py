#!/usr/bin/env python3
"""Builds 4x6 laminate-card PDF of the Study Plan (see specs/study-plan.md).

Usage: python3 print/make_cards.py   (needs: pip install reportlab)
OT card reads the generated internal/dailyreader/study-plan.md (365 days,
02/29 skipped). NT card is the quarterly pass on a 90-day pace, computed with
the same `total*j/n` split the generator uses.
"""
import re, pathlib
from reportlab.pdfgen import canvas
from reportlab.pdfbase.pdfmetrics import stringWidth

ROOT = pathlib.Path(__file__).resolve().parent.parent
W, H = 6 * 72, 4 * 72  # 6x4in landscape (a 4x6 card turned sideways)

ABBR = {"Genesis":"Gen","Exodus":"Exod","Leviticus":"Lev","Numbers":"Num","Deuteronomy":"Deut",
 "Joshua":"Josh","Judges":"Judg","Ruth":"Ruth","1 Samuel":"1 Sam","2 Samuel":"2 Sam",
 "1 Kings":"1 Kgs","2 Kings":"2 Kgs","1 Chronicles":"1 Chr","2 Chronicles":"2 Chr","Ezra":"Ezra",
 "Nehemiah":"Neh","Esther":"Esth","Job":"Job","Psalms":"Ps","Psalm":"Ps","Proverbs":"Prov",
 "Ecclesiastes":"Eccl","Song of Solomon":"Song","Song of Songs":"Song","Isaiah":"Isa","Jeremiah":"Jer",
 "Lamentations":"Lam","Ezekiel":"Ezek","Daniel":"Dan","Hosea":"Hos","Joel":"Joel","Amos":"Amos",
 "Obadiah":"Obad","Jonah":"Jonah","Micah":"Mic","Nahum":"Nah","Habakkuk":"Hab","Zephaniah":"Zeph",
 "Haggai":"Hag","Zechariah":"Zech","Malachi":"Mal","Matthew":"Matt","Mark":"Mark","Luke":"Luke",
 "John":"John","Acts":"Acts","Romans":"Rom","1 Corinthians":"1 Cor","2 Corinthians":"2 Cor",
 "Galatians":"Gal","Ephesians":"Eph","Philippians":"Phil","Colossians":"Col",
 "1 Thessalonians":"1 Thes","2 Thessalonians":"2 Thes","1 Timothy":"1 Tim","2 Timothy":"2 Tim",
 "Titus":"Titus","Philemon":"Phlm","Hebrews":"Heb","James":"Jas","1 Peter":"1 Pet","2 Peter":"2 Pet",
 "1 John":"1 Jn","2 John":"2 Jn","3 John":"3 Jn","Jude":"Jude","Revelation":"Rev"}

def abbr(cell):
    out = []
    for part in cell.split(";"):
        m = re.match(r"\s*(.+?)\s+([\d\-,: ]+)$", part)
        out.append(f"{ABBR[m.group(1)]} {m.group(2).strip()}")
    return "; ".join(out)

# ---- OT
ot = []
for line in (ROOT/"internal/dailyreader/study-plan.md").read_text().splitlines():
    m = re.match(r"\| (\d\d)/(\d\d) \| (.*?) \|", line)
    if m and (m.group(1), m.group(2)) != ("02", "29"):
        ot.append(abbr(m.group(3)))
assert len(ot) == 365, len(ot)

# ---- NT (90-day pace)
NT = [("Matthew",28),("Mark",16),("Luke",24),("John",21),("Acts",28),("Romans",16),
 ("1 Corinthians",16),("2 Corinthians",13),("Galatians",6),("Ephesians",6),("Philippians",4),
 ("Colossians",4),("1 Thessalonians",5),("2 Thessalonians",3),("1 Timothy",6),("2 Timothy",4),
 ("Titus",3),("Philemon",1),("Hebrews",13),("James",5),("1 Peter",5),("2 Peter",3),("1 John",5),
 ("2 John",1),("3 John",1),("Jude",1),("Revelation",22)]
chs = [(b, c) for b, n in NT for c in range(1, n + 1)]
assert len(chs) == 260
N = 90
nt = []
for j in range(N):
    seg = chs[260*j//N : 260*(j+1)//N]
    parts = []
    for b, c in seg:
        if parts and parts[-1][0] == b and parts[-1][2] == c - 1: parts[-1][2] = c
        else: parts.append([b, c, c])
    nt.append("; ".join(f"{ABBR[b]} {a}" + (f"-{z}" if z != a else "") for b, a, z in parts))

INK, MUTE, RULE = (0.08,0.08,0.1), (0.4,0.4,0.45), (0.8,0.8,0.84)

def page(c, title, sub, rows, start, ncols, per_col, fs, note=None):
    m = 11
    c.setFont("Helvetica-Bold", 9); c.setFillColorRGB(*INK)
    c.drawString(m, H - m - 7, title)
    c.setFont("Helvetica", 6); c.setFillColorRGB(*MUTE)
    c.drawRightString(W - m, H - m - 7, sub)
    top = H - m - 13
    c.setStrokeColorRGB(*RULE); c.setLineWidth(.5); c.line(m, top, W - m, top)
    colw = (W - 2*m) / ncols
    lead = (top - m - (10 if note else 0) - 3) / per_col
    for i, r in enumerate(rows):
        col, row = divmod(i, per_col)
        x = m + col*colw + 2
        y = top - 3 - (row + 1)*lead + (lead - fs)/2 + 1
        if row % 2 == 0:
            c.setFillColorRGB(0.95,0.95,0.97)
            c.rect(x - 2, top - 3 - (row + 1)*lead, colw - 2, lead, stroke=0, fill=1)
        d = start + i
        c.setFillColorRGB(*MUTE); c.setFont("Helvetica-Bold", fs)
        c.drawRightString(x + fs*1.9, y, str(d))
        c.setFillColorRGB(*INK)
        hi = r.startswith("Ps 119")
        c.setFont("Helvetica-Bold" if hi else "Helvetica", fs)
        c.drawString(x + fs*2.4, y, r + ("  ★" if hi else ""))
    if note:
        c.setFont("Helvetica", 5.6); c.setFillColorRGB(*MUTE)
        c.drawString(m, m - 3, note)
    c.showPage()

c = canvas.Canvas(str(pathlib.Path(__file__).with_name("study-plan-cards.pdf")), pagesize=(W, H))
c.setTitle("Study Plan – 4x6 cards"); c.setAuthor("study-help")
PS = "Psalm every day: Day n → Ps ((n−1) mod 150)+1 (Day 1 = Ps 1, Day 150 = Ps 150, Day 151 = Ps 1 again). ★ Day 119 & 269: Psalm 119 is the whole OT reading."
page(c, "OLD TESTAMENT · Day 1–183", "Card 1 · front  |  Genesis → Malachi (no Psalms)  |  Jan 1 = Day 1", ot[:183], 1, 5, 37, 6.4, PS)
page(c, "OLD TESTAMENT · Day 184–365", "Card 1 · back  |  Day 365 = Dec 31  |  Feb 29 = catch-up day", ot[183:], 184, 5, 37, 6.4, PS)
NTNOTE = "Restart at Day 1 on Jan 1, Apr 1, Jul 1 and Oct 1 — four full passes a year. Days 91–92 (Q3/Q4) are catch-up. Add the day's Psalm (see OT card)."
page(c, "NEW TESTAMENT · Day 1–45", "Card 2 · front  |  Matthew → Revelation, once a quarter", nt[:45], 1, 3, 15, 10.5, NTNOTE)
page(c, "NEW TESTAMENT · Day 46–90", "Card 2 · back  |  then start over", nt[45:], 46, 3, 15, 10.5, NTNOTE)
c.save()
print("ok", ot[0], ot[118], ot[268], ot[-1], "|", nt[0], nt[-1])
