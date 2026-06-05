#!/usr/bin/env swift
//
// make-icon.swift — regenerates the app icon (single 1024×1024 PNG).
//
// Usage:  swift ios/Tools/make-icon.swift
// Output: ios/StudyHelp/Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png
//
// Draws an open-book mark in cream on a deep-indigo vertical gradient.
// Full-bleed and OPAQUE (no alpha channel) — iOS masks the corners itself and
// rejects marketing icons that carry alpha.

import AppKit
import CoreGraphics
import ImageIO
import UniformTypeIdentifiers

let S = 1024
let outPath = CommandLine.arguments.count > 1
    ? CommandLine.arguments[1]
    : "StudyHelp/Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png"

// Opaque bitmap context (noneSkipLast = RGB, alpha byte ignored → no alpha).
let cs = CGColorSpaceCreateDeviceRGB()
guard let cg = CGContext(
    data: nil, width: S, height: S, bitsPerComponent: 8, bytesPerRow: 0,
    space: cs, bitmapInfo: CGImageAlphaInfo.noneSkipLast.rawValue
) else { fatalError("could not allocate CGContext") }

// Wrap so AppKit drawing (NSBezierPath/NSColor) targets this context.
let nsctx = NSGraphicsContext(cgContext: cg, flipped: false)
NSGraphicsContext.saveGraphicsState()
NSGraphicsContext.current = nsctx

let Sf = CGFloat(S)

// ---- background gradient (top → bottom) ----
let top = NSColor(srgbRed: 0.23, green: 0.27, blue: 0.49, alpha: 1).cgColor
let bottom = NSColor(srgbRed: 0.11, green: 0.13, blue: 0.27, alpha: 1).cgColor
let grad = CGGradient(colorsSpace: cs, colors: [top, bottom] as CFArray, locations: [0, 1])!
cg.drawLinearGradient(grad, start: CGPoint(x: 0, y: Sf), end: CGPoint(x: 0, y: 0), options: [])

let cream = NSColor(srgbRed: 0.96, green: 0.93, blue: 0.85, alpha: 1)
let line = NSColor(srgbRed: 0.55, green: 0.50, blue: 0.42, alpha: 1)
let spineX: CGFloat = 512

// page corners (origin bottom-left)
func page(left: Bool) -> NSBezierPath {
    let outerX: CGFloat = left ? 232 : 792
    let oTop = NSPoint(x: outerX, y: 628)
    let oBot = NSPoint(x: outerX, y: 388)
    let sBot = NSPoint(x: spineX, y: 452)
    let sTop = NSPoint(x: spineX, y: 676)
    let dir: CGFloat = left ? 1 : -1
    let p = NSBezierPath()
    p.move(to: oTop)
    p.line(to: oBot)
    p.curve(to: sBot,
            controlPoint1: NSPoint(x: (outerX + spineX) / 2, y: oBot.y + 6),
            controlPoint2: NSPoint(x: spineX - 90 * dir, y: sBot.y - 18))
    p.line(to: sTop)
    p.curve(to: oTop,
            controlPoint1: NSPoint(x: spineX - 90 * dir, y: sTop.y - 18),
            controlPoint2: NSPoint(x: (outerX + spineX) / 2, y: oTop.y + 6))
    p.close()
    return p
}

// soft shadow under the book
cg.setShadow(offset: CGSize(width: 0, height: -18), blur: 40,
             color: NSColor(white: 0, alpha: 0.30).cgColor)
cream.setFill()
page(left: true).fill()
page(left: false).fill()
cg.setShadow(offset: .zero, blur: 0, color: nil) // details: no shadow

// spine cleft
let spine = NSBezierPath()
spine.move(to: NSPoint(x: spineX, y: 452))
spine.line(to: NSPoint(x: spineX, y: 676))
line.withAlphaComponent(0.45).setStroke()
spine.lineWidth = 8
spine.stroke()

// text lines on each page
line.withAlphaComponent(0.8).setStroke()
let rows: [CGFloat] = [600, 556, 512, 468]
for (i, y) in rows.enumerated() {
    let inset = 30 + CGFloat(i) * 6
    let drop = CGFloat(i)
    let l = NSBezierPath()
    l.move(to: NSPoint(x: 286, y: y - drop * 8))
    l.line(to: NSPoint(x: spineX - inset, y: y - drop * 4))
    l.lineWidth = 14; l.lineCapStyle = .round; l.stroke()
    let r = NSBezierPath()
    r.move(to: NSPoint(x: spineX + inset, y: y - drop * 4))
    r.line(to: NSPoint(x: 738, y: y - drop * 8))
    r.lineWidth = 14; r.lineCapStyle = .round; r.stroke()
}

NSGraphicsContext.restoreGraphicsState()

guard let image = cg.makeImage() else { fatalError("makeImage failed") }
let url = URL(fileURLWithPath: outPath) as CFURL
guard let dest = CGImageDestinationCreateWithURL(url, UTType.png.identifier as CFString, 1, nil) else {
    fatalError("could not create PNG destination")
}
CGImageDestinationAddImage(dest, image, nil)
guard CGImageDestinationFinalize(dest) else { fatalError("PNG finalize failed") }
print("wrote \(outPath)")
