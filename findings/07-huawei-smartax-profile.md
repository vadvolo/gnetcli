# 07 — Huawei SmartAX / MA5800 GPON OLTs need their own profile

**Patch:** [`patches/newfile-smartax.go.patch`](./patches/newfile-smartax.go.patch) (`pkg/device/huawei/smartax.go`, new) · registered `"smartax"` in `pkg/devconf/devconf.go` · depends on issue 06 (default answers).

## Symptom
Huawei SmartAX/MA5800 OLTs (`vendor: huawei, platform: smartax`, e.g. `OLT XPON
Beogradjanka` 172.24.150.170) returned `generic_error` on `annet_execute`. Auth
and connect succeeded — the failure was in the CLI interaction.

## Root cause
The MA5800 interactive CLI differs from Huawei VRP and broke gnetcli's generic
`huawei` profile: bare prompt, CR line terminator, a blocking parameter menu, and
an async warning that redraws the prompt.

## Patch — new `NewSmartAXDevice`
Kept the shared `huawei` profile untouched (zero blast radius). Quirks handled:
1. **Bare prompt** `HOST>` / `HOST#` / `HOST(config)#` (no VRP `<HOST>`/`[HOST]`).
2. **No VRP autocommands** (`screen-length 0 temporary` → "% Unknown command").
3. **CR terminator** `WithWriteNewLine("\r")` — LF alone doesn't submit.
4. **Trailing-space echo** — device autocompletes the last token and echoes
   `cmd `+space, so the echo expr tolerates `\s*` before newline.
5. **`{ <cr>|... }:` parameter menu** is a blocking prompt — detected as a
   question, auto-answered with `\r` via `WithAnswers` (answer must literally be
   `"\r"`; `Answer.Match` appends no newline). Relies on issue 06.
6. **Async "default password" warning** on the first menu command redraws
   `HOST>cmd ` — detected as a redraw-question and also answered `\r`.

## Verification
`annet_execute display version` → exit 0 with full version output (MA5800V100R018,
PATCH SPH613, MA5800-X17). Commands must be Huawei `display …` (not `show …`).

## Notes / caveats
- Two whitelists gate `/execute`: the Go one (`internal/api/handlers/execute.go`)
  **and** the MCP TS one (`mcp-annet-oil/src/command-whitelist.ts`) — both need a
  Huawei `display .+`/`tracert`/`ping -c` category.
- Fleet flakiness: some commands intermittently `generic_error`/`connect_error`
  because the OLTs still use the **default password** (root/admin123), so the
  async warning keeps firing. Real fix = change the OLT password on the device.
