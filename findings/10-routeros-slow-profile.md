# 10 — Weak RouterOS CPE `ros_slow` — attempted, NOT adopted (negative result)

**Patch:** [`patches/newfile-rosslow.go.patch`](./patches/newfile-rosslow.go.patch) (`pkg/device/ros/rosslow.go`, new, **dormant**) · registered `"ros_slow"` in `pkg/devconf/devconf.go`.

## Status
⚠️ **Kept in the tree but dormant.** No inventory device is routed to it, so it
has zero effect unless something is explicitly tagged `platform: routeros_slow`.
Documented here so the experiment is not repeated.

## Hypothesis
After all issue-04 fixes, a full sweep left an **827-device RouterOS
command-fail tail**. Breakdown (fresh sweep, conc 3):

| count | reachable | login | reason | nature |
|------:|-----------|-------|--------|--------|
| 291 | yes | timeout | i/o timeout | slow/offline (device-side) |
| 257 | yes | auth_err | bad password | creds (netpasswd, not code) |
| **248** | yes | **ok** | **generic_error** | weak RB tear-down (only code-fixable class) |
| ~31 | yes | ok/failed | connect_error | transient |

Hypothesis: the 248 (login OK, session dropped on submit, EOF surviving retries)
are old RB2011 that can't keep up with the interactive-editor redraw even at the
standard 12ms write delay. A **separate** profile with a larger `WithWriteDelay`
(so the ~9000 healthy RouterOS on the standard profile are untouched) might
recover them.

## What was built
`pkg/device/ros/rosslow.go` `NewSlowDevice` — a clone of `ros.NewDevice` with
`WithWriteDelay(50ms)` instead of 12ms; everything else identical (reuses the
package's prompt/error/pager exprs, `dataCallback`, `credentialLoginModifier`).
Routed via annet-oil `DeviceType()` case `routeros_slow|ros_slow → ros_slow`.

## Result — the delay is NOT the lever
A/B on 20 teardown devices was pure noise:

| run | Commands OK / 20 |
|-----|-----------------|
| baseline (standard 12ms) | 6 |
| ros_slow 50ms, conc 3 | 11 |
| ros_slow 50ms, conc 3 (repeat) | 8 |
| ros_slow 50ms, **conc 1 (gentlest)** | **5** |

The gentlest condition (no concurrency) gave the *worst* result — so timing is
not the cause. These CPE drop the session for device-side reasons (weak CPU /
firmware), and more write delay won't fix that.

**Telnet was also checked:** only 11 of the 248 have port 23 open at all, and of
those telnet recovered just 1/11. Not a path either.

## Conclusion
No clean code fix remains for the RouterOS tail; it is ~66% device-side
(creds + offline). Left `ros_slow` in place as a **manual escape hatch** for
individually tagging a known-problem device, nothing more. Consider deleting
before upstreaming if a dormant profile is unwanted.
