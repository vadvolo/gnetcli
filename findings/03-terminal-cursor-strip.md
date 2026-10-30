# 03 — Cursor-movement escapes corrupt parsed output

**Patch:** [`patches/03-terminal-cursor-strip.patch`](./patches/03-terminal-cursor-strip.patch) · `pkg/terminal/parser.go`

## Symptom
Output from devices that redraw with cursor-up/down escapes (`ESC[A` / `ESC[B`)
came back garbled or with stray control bytes, breaking prompt/error matching.

## Root cause
The terminal parser only stripped `CUP` (`H`) and `ED` (`J`) escape sequences.
`CUU` (cursor up, `A`) and `CUD` (cursor down, `B`) fell through and were left in
the byte stream, corrupting the rendered text the matchers see.

## Patch
Add `CUU='A'` and `CUD='B'` constants and include them in the strip branch
alongside `CUP, ED`, so cursor-positioning sequences we don't render are removed:

```go
case CUP, ED, CUU, CUD: // cursor positioning we don't render - strip it
    m.data = sliceEdit(m.data, escStart, m.pos+1)
    m.pos = escStart - 1
    continue
```

## Verification
Devices using cursor-up/down redraw now parse cleanly; combined with issue 04
this removes a chunk of RouterOS/CPE `generic_error`.
