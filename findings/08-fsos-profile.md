# 08 — FS / FSOS switches need their own profile

**Patch:** [`patches/newfile-device.go.patch`](./patches/newfile-device.go.patch) (`pkg/device/fsos/device.go`, new) · registered `"fsos"` in `pkg/devconf/devconf.go` · depends on issue 01 (legacy algos).

## Symptom
FS switches (FSOS, e.g. S5810 running FSOS 11.4) failed `/execute`. Two layers:
the SSH handshake failed (issue 01), and even after that the generic profile
mis-drove the CLI.

## Root cause
1. Legacy SSH crypto only (fixed in issue 01).
2. **No SSH exec channel:** `ssh host "cmd"` returns `SCP Server Service is
   disable!` — FSOS needs an interactive **PTY** session, not a one-shot exec.
3. FSOS prompt/pager differ enough from the generic Cisco-like profile to need a
   dedicated one.

## Patch — new `fsos.NewDevice`
Dedicated genericcli profile for FSOS: its own prompt/error/pager expressions and
PTY-based interaction, registered as deviceType `"fsos"`. Routed by inventory
`vendor: fsos`.

## Verification
FS S5810 (FSOS 11.4, offers only `ssh-rsa`): with issue 01 + this profile,
`show version` returns exit 0. Only **port 22** works on these (10022/telnet
closed).
