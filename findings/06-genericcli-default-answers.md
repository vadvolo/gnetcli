# 06 — Device-level default answers ignored outside the login path

**Patch:** [`patches/05-genericcli-writedelay-answers.patch`](./patches/05-genericcli-writedelay-answers.patch) · `pkg/device/genericcli/genericcli.go` (`GenericExecute` question handler)

## Symptom
Devices whose CLI blocks on an interactive prompt **mid-command** (not at login)
hung to timeout → `generic_error`, even when the device profile declared a
matching `WithAnswers` default answer. Concretely the Huawei SmartAX
`{ <cr>|... }:` parameter menu (see issue 07).

## Root cause
In the exec loop, when the command itself had no answer for a question
(`cmd.ErrNotFoundAnswer`), `GenericExecute` immediately threw a
`QuestionException`. Device-level default answers (`WithAnswers` /
`cli.defaultAnswers`) were only consulted in the **login** path, never during
command execution.

## Patch
On `ErrNotFoundAnswer`, fall back to `cli.defaultAnswers`: iterate them, and if
one `Match`es the question, use its answer; only throw if none match. Additive —
devices that set no default answers keep the previous throw behavior.

```go
if !errors.Is(err, cmd.ErrNotFoundAnswer) {
    return nil, fmt.Errorf("QuestionHandler error %w", err)
}
// fall back to device-level default answers (WithAnswers)
found := false
for _, cmdAnswer := range cli.defaultAnswers {
    if ans, ok, aerr := cmdAnswer.Match(question); aerr != nil {
        return nil, aerr
    } else if ok { answer = ans; found = true; break }
}
if !found { return nil, device.ThrowQuestionException(question) }
```

## Verification
SmartAX MA5800 menu-driven commands auto-answer the `{ <cr>|... }:` prompt with
Enter and complete (issue 07). Also fixes the same class for any profile that
relies on `WithAnswers` during execution.
