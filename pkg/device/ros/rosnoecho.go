/*
rosnoecho.go is a RouterOS profile variant that tolerates the interactive
line-editor echo redraw/wrap on LONG commands (e.g.
`/ip service set ssh address=<50+ chars>`).

The stock `ros` profile requests an 80-col pty (wider widths make RouterOS 6.49.x
flood redraw padding and drop the session), and matches the FULL echoed command
to detect echo. A command longer than ~72 chars wraps at col 80, so the full
command never appears contiguously and echo detection fails with "echo read
error". This profile instead matches only a short, un-wrappable PREFIX of the
echoed command, so wrapped long commands still pass echo detection; the wrapped
remainder is consumed as (ignored) pre-prompt output. Everything else — prompt,
error, pager, question exprs, write delay, credential modifier — is identical to
the stock ros profile.
*/
package ros

import (
	"regexp"
	"time"

	"github.com/annetutil/gnetcli/pkg/cmd"
	"github.com/annetutil/gnetcli/pkg/device/genericcli"
	"github.com/annetutil/gnetcli/pkg/expr"
	"github.com/annetutil/gnetcli/pkg/streamer"
)

// echoPrefixLen is how many leading command bytes to match. It must be short
// enough to never wrap at the 80-col pty (prompt + this < 80).
const echoPrefixLen = 20

// NewNoEchoDevice builds a RouterOS device that tolerates wrapped echo on long
// commands.
func NewNoEchoDevice(connector streamer.Connector, opts ...genericcli.GenericDeviceOption) genericcli.GenericDevice {
	cli := genericcli.MakeGenericCLI(
		expr.NewSimpleExprLast(65536).FromPattern(promptExpression),
		expr.NewSimpleExprLast(2500).FromPattern(errorExpression),
		genericcli.WithQuestion(
			expr.NewSimpleExprLast200().FromPattern(questionExpression),
		),
		genericcli.WithPager(expr.NewSimpleExprLast200().FromPattern(pagerExpression)),
		genericcli.WithResultCB(dataCallback),
		genericcli.WithCredentialInterceptor(credentialLoginModifier),
		genericcli.WithTerminalParams(80, 4098),
		genericcli.WithWriteDelay(12*time.Millisecond),
		genericcli.WithWriteNewLine([]byte("\r\n")),
		genericcli.WithEchoExprFn(func(command cmd.Cmd) expr.Expr {
			v := string(command.Value())
			n := len(v)
			if n > echoPrefixLen {
				n = echoPrefixLen
			}
			// prefix, then any chars (incl. redraw padding / wrap newlines), then
			// a newline — matches the echoed command whether or not it wrapped.
			return expr.NewSimpleExpr().FromPattern(regexp.QuoteMeta(v[:n]) + `[\s\S]*?` + genericcli.AnyNLPattern)
		}),
	)
	return genericcli.MakeGenericDevice(cli, connector, opts...)
}
