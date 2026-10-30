# 09 — Cisco IOS-XR needs its own profile

**Patch:** [`patches/newfile-ciscoxr.go.patch`](./patches/newfile-ciscoxr.go.patch) (`pkg/device/cisco/ciscoxr.go`, new) · registered `"cisco_xr"` in `pkg/devconf/devconf.go` · companion routing in annet-oil.

## Symptom
Cisco **IOS-XR** core routers (`vendor: cisco, platform: iosxr`, e.g. the
172.22.2.x ASR9K: `RP/0/RSP1/CPU0:CORE-RIGEL#`, White Walker, Yoda, Viserion,
R2D2, Night King) returned a mix of `connect_error` and `generic_error`.

## Root cause
gnetcli routed them to the generic `cisco` (IOS/Catalyst) profile. Two problems:
1. **No `enable` mode on IOS-XR.** The IOS profile sends an `enable` autocommand
   with a `Password:` answer. On XR `enable` is `% Invalid input`, and the exec
   loop is left waiting on a `Password:` answer that never comes → desync →
   `generic_error`.
2. Timestamp/pager/prompt specifics differ from Catalyst.

(The transport `connect_error` on these boxes is separate — see "Not fixed by
this" below.)

## Patch — new `NewXRDevice`
Based on the IOS profile, in the same package, reusing its error/login/pager
consts, but:
- **Drop `enable`** (XR is already privileged).
- Autocommands: `terminal length 0`, `terminal width 0`, `terminal exec prompt
  no-timestamp` (kills the per-prompt `Thu Aug 20 …` timestamp line), `terminal
  monitor disable` — all `WithErrorIgnore`.
- Prompt `(?P<prompt>[\w\-.:/]+(\([^)]+\))?)(>|#)$` — the base class already
  allows the `/` and `:` of the `RP/0/RSP0/CPU0:host` path; one optional
  parenthetical covers `(config)`, `(config-if)`, `(admin)`, …

`NewDevice` (Catalyst/IOS) is left untouched — zero blast radius.

## Routing (annet-oil companion)
`inventory.Device.DeviceType()` maps `platform: iosxr|ios-xr|ios_xr → cisco_xr`.
**Routing bug found & fixed:** the sweep's command runner
(`internal/cli/check.go` `makeCommandRunner`) passed `dev.Vendor`, not
`dev.DeviceType()`, so `check` always used the generic profile even for
smartax/iosxr (the `/execute` and `/state` handlers already used `DeviceType()`).
Any new dedicated profile needs **both** the `DeviceType()` case and this call
site.

## Verification
All 6 IOS-XR → `Commands OK 6/6` over SSH; `/execute show version` on the live
API returns the XR banner (verified profile is live: gnetcli writes `terminal
exec prompt no-timestamp`, XR-only).

## Not fixed by this
These boxes **reset SSH handshakes under concurrency** (`read: connection reset
by peer`) — a device-side SSH-session/rate limit, not a profile issue. It shows
as first-attempt `connect_error`; absorbed by annet-oil's command-stage retry +
low concurrency, not by this profile.
