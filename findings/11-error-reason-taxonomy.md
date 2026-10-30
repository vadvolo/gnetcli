# 11 — Make failures self-explanatory (fine-grained error reason)

**Patches:** [`patches/11a-gnetcli-error-reason.patch`](./patches/11a-gnetcli-error-reason.patch) (`cmd/gnetcli_server/server.go`) · [`patches/11b-annet-surface-errdetail.patch`](./patches/11b-annet-surface-errdetail.patch) (annet-oil `internal/gnetcli/client.go`)

## Symptom
Every `/execute` failure surfaced as one of two opaque codes —
`generic_error` or `connect_error` — with no hint of the actual cause. Triaging
12k devices meant reproducing each failure by hand with `journalctl -u gnetcli`
to find out whether it was bad creds, a crypto handshake, a dropped session, or a
prompt mismatch.

## Key finding
gnetcli **already transmits the real cause**: `connectionErrorInterceptor`
attaches the underlying error as a gRPC status detail
(`errdetails.ErrorInfo{Reason, Metadata["err"]}`). The top-level status *message*
was just the opaque code, and **annet-oil never read the detail** — it logged and
returned only the code. So the information existed end-to-end; it was thrown away
at the last step.

## Patch — two additive parts

**A. annet-oil surfaces the detail** (`internal/gnetcli/client.go`, `describeGnetcliErr`).
On a failed `Exec`, parse the gRPC status, pull `ErrorInfo.Reason` +
`Metadata["err"]`, and compose them into the returned error. The original code
word (`connect_error`/`generic_error`) is preserved as a substring, so existing
`Contains`-based classification and the retry gate (`isRetryableCmdErr`) keep
working unchanged.

**B. gnetcli emits a fine-grained `Reason`** (`cmd/gnetcli_server/server.go`,
`classifyExecReason`). The top-level message stays one of the two stable codes
(back-compat), but the `Reason` is now specific:

| phase | reason values |
|-------|---------------|
| connect (`failed to connect to host`) | `auth_error`, `handshake_error`, `refused`, `reset`, `no_route`, `connect_timeout`, `connect_error` |
| exec (CLI) | `echo_error`, `session_eof`, `unanswered_question`, `prompt_mismatch`, `exec_timeout`, `exec_error` |

## Result
Errors are now self-explanatory on both the sweep and the API:

```
connect_error [auth_error]: ... ssh: unable to authenticate, attempted methods [none password], no supported methods remain
connect_error [handshake_error]: ... ssh: handshake failed: no common algorithm for key exchange
generic_error [echo_error]: ... echo read error        # RouterOS echo-storm / weak-CPE teardown
generic_error [session_eof]: ... EOF
```

`[auth_error]` (wrong password → netpasswd) is now instantly distinguishable from
`[handshake_error]` (legacy crypto → issue 01) and from `[echo_error]` (weak
RouterOS → issue 04/10), which previously all hid behind two codes.

## Compatibility
- gRPC message codes unchanged → no break for anything matching
  `connect_error`/`generic_error`.
- `ErrorInfo.Reason` value changed from the old `connection_error`/`generic_error`
  to the taxonomy above; nothing consumed the old value (annet-oil ignored
  details entirely).
- Requires rebuilding **both** `gnetcli_server` (restart `gnetcli`) and
  `annet-oil-server` (restart API for `/execute`; the `check` binary picks it up
  on next run).

## Follow-up ideas
- Optionally split the top-level code too (e.g. add `auth_error` as its own gRPC
  code) once all consumers read `Reason` — currently kept as `connect_error` for
  back-compat.
- Feed `Reason` into the sweep's `result.json` as a first-class column so the
  dashboard can group by cause without string parsing.
