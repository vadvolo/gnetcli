# 05 — Telnet-only devices were always dialed over SSH

**Patch:** streamer-type wiring in `pkg/streamer` + `pkg/device/genericcli` (see [`patches/05-genericcli-writedelay-answers.patch`](./patches/05-genericcli-writedelay-answers.patch) and the streamer changes). Companion routing in annet-oil.

## Symptom
91 telnet-only devices (port 23 Cisco 4948/2960 and similar) timed out ~20s with
`connect_error` on every `/execute`, `/state`, `/topology`.

## Root cause
annet-oil's `ExecWithDevice` never set gnetcli's `StreamerType`, so gnetcli
defaulted to the **SSH** streamer for every device — including port-23
telnet-only boxes, which have no SSH listener.

## Patch
- gnetcli: honor a per-request `StreamerType` (ssh|telnet) so the caller can pick
  the transport.
- annet-oil: `inventory.Device.Protocol` (`telnet`|`ssh`) + `UsesTelnet()`
  (explicit protocol wins, else `port==23` ⇒ telnet); `ExecWithDevice` takes a
  telnet bool and sets `HostParams.StreamerType = telnet`; inventory got
  `protocol: telnet` on the 91 port-23 devices.

### Build-sync caveat (important)
`go.mod` must `replace annetutil/gnetcli => /root/gnetcli/gnetcli`. The released
v1.2.20 proto lacks `StreamerType`; v1.3.10 uses a **different enum value**
(telnet=2 vs the fork/server's telnet=1). Building the client against the fork
keeps client and running server in sync.

## Verification
Telnet-only Cisco (e.g. `LAN SW Prodaja 4`, Catalyst 3550 `192.168.29.54`) with
`protocol: telnet` + `port: 23`: gnetcli logs `streamer=telnet`, `show version`
exit 0. NB: the NOC "routing.json/container mismatch" RCA was wrong — `/execute`
does not use routing.json/containers at all.
