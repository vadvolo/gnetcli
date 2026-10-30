# 02 — Output discarded when the device closes the session on the prompt

**Patch:** [`patches/02-eof-final-match.patch`](./patches/02-eof-final-match.patch) · `pkg/streamer/streamer.go` (`GenericReadX`)

## Symptom
Devices that print the command output + prompt and **immediately close** the SSH
session returned `generic_error` (EOF) even though the full, matchable output had
already arrived in the last chunk.

## Root cause
`GenericReadX` matched the read buffer against the expected expr only while the
stream was open. On channel close (`!ok`) it returned `EOF` unconditionally,
throwing away a buffer that may already contain a complete prompt/pager match.
Some RouterOS and weak CPE close the session within milliseconds of emitting the
prompt, so the final chunk both completes the command and closes the stream.

## Patch
Before returning `EOF`, try the regexp **once more** against the final buffer; if
it matches, return a normal `Expr` result (with the same `Underlying` group
handling as the in-loop path) instead of `EOF`.

## Verification
RouterOS devices that previously raced to `EOF` after `/system resource print`
now return the parsed output. Reduces the "connected, output present, still
generic_error" class. (Complementary to issue 04 — narrower terminal reduces how
often the race happens; this makes the race non-fatal when it does.)
