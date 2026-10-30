# 01 — Legacy SSH algorithms rejected before auth

**Patch:** [`patches/01-legacy-ssh-algorithms.patch`](./patches/01-legacy-ssh-algorithms.patch) · commit `7df0a09` · `pkg/streamer/ssh/ssh.go`

## Symptom
Old gear returned a bare `connect_error` on every `/execute`, `/state`,
`/diag`. Verbose OpenSSH against the same device negotiated fine, e.g.:

```
kex: algorithm: diffie-hellman-group-exchange-sha1
kex: host key algorithm: ssh-rsa
```

## Root cause
Go's `x/crypto/ssh` disables SHA-1 host keys (`ssh-rsa`, `ssh-dss`) and legacy
KEX/ciphers by default (since Go 1.20). Old devices (FS/FSOS switches, aging
RouterOS, Eltex MES) offer **only** those, so the transport handshake failed
**before authentication** — and gnetcli wraps any dial/handshake failure into
the same generic `connect_error`, indistinguishable from a bad password.

## Affected devices
FS/FSOS S-series, Eltex MES33xx (e.g. `10.254.255.211:21022`, offers only
`diffie-hellman-group-exchange-sha1`/`group1-sha1` + `ssh-rsa`/`ssh-dss`), and a
long tail of old RouterOS.

## Patch
`GetConfig` in `pkg/streamer/ssh/ssh.go`:
- `sshConf.SetDefaults()` then **append** legacy KEX
  (`diffie-hellman-group-exchange-sha256/-sha1`, `diffie-hellman-group1-sha1`)
  and CBC/3DES ciphers (`aes128/192/256-cbc`, `3des-cbc`) — modern kept first so
  strong devices still pick strong crypto.
- Set `HostKeyAlgorithms` explicitly: modern first, then `ssh-rsa`, `ssh-dss`.

## Verification
FS S5810 (FSOS 11.4, offers only `ssh-rsa`): handshake completes, `show version`
returns exit 0 via the fsos profile. Eltex MES3324F: handshake + `show version`
OK.

## Companion note (annet-oil side)
annet-oil's own native-login probe (`internal/check/check.go` `sshLogin`) had the
**same gap** — it built a bare `ssh.ClientConfig` with default algos, so the
sweep's `native_login` column falsely reported `login=failed` on healthy legacy
devices. Fixed there identically (mirror `SetDefaults()`+legacy+HostKeyAlgorithms).
