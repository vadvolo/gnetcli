/*
Package huawei — SmartAX/MA5800 GPON OLT variant.

MA5800-series OLTs share Huawei's error/login/question grammar but differ from
VRP routers in two ways that break the generic huawei profile:

 1. Prompt: OLTs print a bare "HOSTNAME>" / "HOSTNAME#" / "HOSTNAME(config)#"
    (no angle brackets / square brackets like VRP "<HOST>" / "[HOST]").
 2. Session setup: they do NOT understand VRP autoCommands such as
    "screen-length 0 temporary" — sending them yields "% Unknown command" and
    the OLT tears the channel down. So this profile sends none.
 3. Input assistant: after a command the OLT often prints a parameter menu like
    "{ <cr>|backplane<K>|frameid/slotid<S> }:" and waits. When the menu offers
    "<cr>" the command is complete and just needs an Enter to run, so we detect
    the menu as a question and auto-answer it with a carriage return.
*/
package huawei

import (
	"fmt"
	"regexp"

	"github.com/annetutil/gnetcli/pkg/cmd"
	"github.com/annetutil/gnetcli/pkg/device/genericcli"
	"github.com/annetutil/gnetcli/pkg/expr"
	"github.com/annetutil/gnetcli/pkg/streamer"
)

const (
	// Bare OLT prompt: hostname, optional "(mode)" section, ending in > or #.
	smartaxPromptExpression = `(\r\n|^)(?P<prompt>[/\w\-.:]+(\([\w\-.:/]+\))?[>#])$`
	// MA5800 pager text differs from VRP's "  ---- More ----"; match the family.
	smartaxPagerExpression = `(?P<store>(\r\n|\n))?\s*---- More.*----\s*$`
	// Parameter menu shown by the input assistant. We only treat menus that offer
	// "<cr>" as questions — those mean the command is complete and Enter runs it.
	// Menus without "<cr>" are genuinely incomplete commands and are left to fail.
	smartaxCrMenuExpression = `\{[^\r\n]*<cr>[^\r\n]*\}:\s*$`
	// Input-line redraw state. On the first command of a session the OLT injects
	// an async "default password" warning right after the "<cr>" menu, aborts the
	// menu and redraws the prompt with the command still typed and a trailing
	// space, e.g. "HOST>display version ". We detect that (a prompt followed by
	// text ending in a space) and answer with Enter to re-run the command; the
	// second time the "<cr>" menu is clean and gets confirmed normally.
	smartaxRedrawExpression = `(\r\n|^)[/\w\-.:]+(\([\w\-.:/]+\))?[>#][^\r\n]+ $`
	// Combined question grammar: Huawei's Y/N confirmations, the <cr> menu, and
	// the redraw state — all end-anchored so they only fire as the buffer suffix.
	smartaxQuestionExpression = `(?P<question>.*(privilege level|Continue)\s?\?\s?\[Y/N\]:?$|` + smartaxCrMenuExpression + `|` + smartaxRedrawExpression + `)`
)

// NewSmartAXDevice builds a gnetcli device for Huawei SmartAX / MA5800 OLTs.
func NewSmartAXDevice(connector streamer.Connector, opts ...genericcli.GenericDeviceOption) genericcli.GenericDevice {
	cli := genericcli.MakeGenericCLI(
		expr.NewSimpleExprLast200().FromPattern(smartaxPromptExpression),
		expr.NewSimpleExprLast200().FromPattern(errorExpression),
		genericcli.WithLoginExprs(
			expr.NewSimpleExprLast200().FromPattern(loginExpression),
			expr.NewSimpleExprLast200().FromPattern(passwordExpression),
			expr.NewSimpleExprLast200().FromPattern(passwordErrorExpression),
		),
		genericcli.WithPager(
			expr.NewSimpleExprLast200().FromPattern(smartaxPagerExpression),
		),
		genericcli.WithQuestion(
			expr.NewSimpleExprLast200().FromPattern(smartaxQuestionExpression),
		),
		// Auto-confirm the "{ <cr>|... }:" parameter menu with a carriage return.
		// Answer.Match returns the answer verbatim (no newline is appended), so the
		// answer must itself be the CR that selects <cr>.
		genericcli.WithAnswers([]cmd.Answer{
			cmd.NewAnswer("/"+smartaxCrMenuExpression+"/", "\r", true),
			cmd.NewAnswer("/"+smartaxRedrawExpression+"/", "\r", true),
		}),
		// Send one empty line right after login to flush the async "default
		// password" warning (and any pending redraw) so real commands start from a
		// clean, quiet prompt.
		genericcli.WithAutoCommands([]cmd.Cmd{
			cmd.NewCmd("", cmd.WithErrorIgnore()),
		}),
		genericcli.WithSFTPEnabled(),
		genericcli.WithTerminalParams(400, 0),
		// MA5800's interactive shell treats a bare LF as "show completion" (it
		// pops the "{ <cr>|... }:" parameter menu and redraws the line) instead
		// of executing. It needs a CR to submit, like a real terminal Enter.
		genericcli.WithWriteNewLine([]byte("\r")),
		genericcli.WithEchoExprFn(func(command cmd.Cmd) expr.Expr {
			// MA5800's input assistant may append a trailing space to the echoed
			// command line (auto-completing the last token), so tolerate optional
			// trailing whitespace before the newline.
			return expr.NewSimpleExpr().FromPattern(fmt.Sprintf(`%s\s*(\r\n|\n)`, regexp.QuoteMeta(string(command.Value()))))
		}),
	)
	return genericcli.MakeGenericDevice(cli, connector, opts...)
}
