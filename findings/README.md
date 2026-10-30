# gnetcli fork — findings & patches

This directory documents every change this fork (`github.com/vadvolo/gnetcli`,
branch `main`) carries on top of upstream `annetutil/gnetcli`, one issue per
file. Each issue file has: **Symptom → Root cause → Affected devices → Patch →
Verification**. Exact, apply-ready diffs live in [`patches/`](./patches).

All of these were driven by the **annet-oil** health sweep over Orion Telekom's
~12.5k-device inventory, where each class of failure collapsed into gnetcli's
generic `connect_error` / `generic_error` gRPC codes. Splitting those codes into
real causes is the recurring theme below.

## Build & deploy

```sh
cd /root/gnetcli/gnetcli
go build -o /root/go/bin/gnetcli_server ./cmd/gnetcli_server
systemctl restart gnetcli          # unit runs with -debug; raw I/O in `journalctl -u gnetcli`
```

Device routing lives in `pkg/devconf/devconf.go` `InitDefaultDeviceMapping`
(deviceType string → profile constructor). annet-oil selects the deviceType via
`inventory.Device.DeviceType()` (platform-aware) and passes it to
`client.ExecWithDevice(...)`.

## Issue index

| # | Issue | Area | Status |
|---|-------|------|--------|
| 01 | [Legacy SSH algorithms rejected](./01-legacy-ssh-algorithms.md) | `pkg/streamer/ssh` | ✅ shipped (commit `7df0a09`) |
| 02 | [Output lost when device closes session on prompt](./02-eof-final-match.md) | `pkg/streamer` | ✅ shipped |
| 03 | [Cursor-movement escapes corrupt parsed output](./03-terminal-cursor-strip.md) | `pkg/terminal` | ✅ shipped |
| 04 | [RouterOS `generic_error` (prompt/redraw/echo-storm)](./04-routeros-generic-error.md) | `pkg/device/ros` + `genericcli` + `ssh` | ✅ shipped |
| 05 | [Telnet-only devices dialed over SSH](./05-telnet-streamer-type.md) | `pkg/streamer` + `genericcli` | ✅ shipped |
| 06 | [Device-level answers ignored outside login](./06-genericcli-default-answers.md) | `pkg/device/genericcli` | ✅ shipped |
| 07 | [Huawei SmartAX/MA5800 needs its own profile](./07-huawei-smartax-profile.md) | `pkg/device/huawei` | ✅ shipped |
| 08 | [FS/FSOS switches need their own profile](./08-fsos-profile.md) | `pkg/device/fsos` | ✅ shipped |
| 09 | [Cisco IOS-XR needs its own profile](./09-cisco-iosxr-profile.md) | `pkg/device/cisco` | ✅ shipped |
| 10 | [Weak RouterOS CPE `ros_slow` — attempted, not adopted](./10-routeros-slow-profile.md) | `pkg/device/ros` | ⚠️ dormant (negative result) |
| 11 | [Make failures self-explanatory (error reason taxonomy)](./11-error-reason-taxonomy.md) | `cmd/gnetcli_server` + annet-oil client | ✅ shipped |

## Housekeeping before committing

The working tree has stray backups that should NOT be committed:
`pkg/**/**.bak*`, `pkg/devconf/devconf.go.bak-rosslow`, `cmd/test-fsos/`.
Add them to `.gitignore` or `git clean` them first. The real change set is:

```
pkg/streamer/streamer.go              (02)
pkg/terminal/parser.go                (03)
pkg/device/ros/device.go              (04)
pkg/streamer/ssh/ssh.go               (04 write-throttle; 01 already committed)
pkg/device/genericcli/genericcli.go   (04 write-delay wiring, 06 answers, 05 streamer)
pkg/devconf/devconf.go                (registrations: smartax, cisco_xr, ros_slow, fsos)
pkg/device/huawei/smartax.go          (07, new)
pkg/device/fsos/device.go             (08, new)
pkg/device/cisco/ciscoxr.go           (09, new)
pkg/device/ros/rosslow.go             (10, new, dormant)
```
