# 04 — RouterOS `generic_error` (prompt truncation / redraw flood / echo-storm)

**Patches:** [`patches/04-routeros-prompt-terminal.patch`](./patches/04-routeros-prompt-terminal.patch) (`pkg/device/ros/device.go`) · [`patches/05-genericcli-writedelay-answers.patch`](./patches/05-genericcli-writedelay-answers.patch) (`genericcli` write-delay wiring) · write-throttle in `pkg/streamer/ssh/ssh.go` (`SetWriteDelay`, char-by-char `Write`)

## Symptom
A large fraction of RouterOS devices returned `rpc error: Internal desc =
generic_error` on `/execute` / `/state`, with no useful detail. gnetcli collapses
**several independent RouterOS quirks** into this one code.

## Root causes (all real, all seen)
1. **Prompt left-truncation.** A long/custom identity — especially with spaces
   (`[admin@Access:BD_W002-Spas CRS2] >`) — makes RouterOS left-truncate the
   prompt at width 80, sometimes dropping the whole `[login@` prefix down to a
   bare `<`. The strict prompt regex required `[` + `login` + `@`, so it missed
   the prompt → 15s wait → `generic_error`.
2. **Redraw flood.** RouterOS 6.49.x pads each line to terminal width and redraws
   it via `\r` on every keystroke; at the upstream-default wide PTY
   (`+cte999w255h`, 200 cols) this floods ~1KB/keystroke and the device tears
   down the session ~10KB in.
3. **Prompt beyond the match window.** With redraw padding, the real prompt can
   sit >1500 bytes from the tail (e.g. after `ping`/`traceroute`), past the
   1500-byte match window → prompt missed → EOF → `generic_error`.
4. **Echo-storm on paste.** RouterOS drops the interactive editor session a few
   ms after a full-line paste.

## Patch
`pkg/device/ros/device.go`:
- **Prompt regex** made tolerant: accept `[` **or** `<` at the start and make the
  `login@` group **optional** — `[\[<]((?P<login>[^@\r\n]*)@)?(?P<hostname>[^\]\r\n]+)\]…`
  (both `visiblePrompt` and `hiddenPrompt`).
- **Match window** widened `1500 → 65536` so the prompt is found even after a big
  redraw/ping/traceroute.
- **Narrow PTY** `WithTerminalParams(80, 4098)` + login modifier `+cte80w4098h`
  (was `+cte999w255h`): width 80 bounds the redraw flood; tall height disables
  the pager that stalled output.
- **Write throttle** `WithWriteDelay(12ms)` + `WithWriteNewLine("\r\n")`: the
  command is typed byte-by-byte with a delay so the editor keeps up and doesn't
  drop the session (implemented in the ssh streamer `SetWriteDelay` + per-byte
  `Write`, wired through `genericcli` — see issue 06 patch).

## Verification
On a fixed 120-device RouterOS sample the OK rate rose from 0 to a large majority
after these fixes; a long-identity device (`[admin@Access:…Spas CRS2]`) that
500'd now returns output. **Do not widen the PTY past 80** — it reintroduces
cause 2.

## Residual (see issue 10)
A ~12-20% tail of weak CPE (old RB2011) still tears the session down on submit
regardless of write delay — that is device-side instability, not a code bug. A
larger-delay `ros_slow` profile was tried and **did not** reliably help.

## Companion note (annet-oil side)
The remaining transient failures at sweep concurrency are best absorbed by a
retry in annet-oil's command stage (`internal/check/check.go` `collectCommands`,
retry on generic_error/connect_error/EOF) — first-pass OK on the sample went
57.5% → 87.5%.
