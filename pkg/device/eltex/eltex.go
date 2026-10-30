/*
Package eltex implements Eltex CLI using genericcli.
*/
package eltex

import (
	"fmt"
	"regexp"

	"github.com/annetutil/gnetcli/pkg/cmd"
	"github.com/annetutil/gnetcli/pkg/device/genericcli"
	"github.com/annetutil/gnetcli/pkg/expr"
	"github.com/annetutil/gnetcli/pkg/streamer"
)

const (
	loginExpression    = `.*Username:\s?$`
	questionExpression = `\n(?P<question>.*Continue\? \[Y/N\]:)$`
	promptExpression   = `(?P<prompt>\r?[\w\-.:/]+(\(conf(ig)?(-[^)]+)*\))?)(>|#)$`
	errorExpression    = `(` +
		`\r\n% Invalid input detected at '\^' marker\r\n` +
		`|\r% Ambiguous Command\r\n` +
		`|\r% Invalid Command\r\n` +
		`)`
	passwordExpression      = `.*password:\s?$`
	passwordErrorExpression = `\nPermission denied, please try again.(\r\n|\n)`
	// Two pager styles: the classic MES 23xx/24xx "--More--" and the newer
	// MES 3300/5300 (4.x firmware) "More: <space>, Quit: ... One line: <return>".
	pagerExpression         = `(\r\x1b\[K\r--More--\x1b\[K|More: <space>, +Quit: q or CTRL\+Z, One line: <return>)`
)

var autoCommands = []cmd.Cmd{
	cmd.NewCmd("set cli pagination off", cmd.WithErrorIgnore()), //Eltex MES 23xx/24xx
	cmd.NewCmd("terminal datadump", cmd.WithErrorIgnore()),      //Eltex MES 3300/5300 (4.x): disable pager so output streams
}

func NewDevice(connector streamer.Connector, opts ...genericcli.GenericDeviceOption) genericcli.GenericDevice {
	cli := genericcli.MakeGenericCLI(expr.NewSimpleExprLast200().FromPattern(promptExpression), expr.NewSimpleExprLast200().FromPattern(errorExpression),
		genericcli.WithLoginExprs(
			expr.NewSimpleExprLast200().FromPattern(loginExpression),
			expr.NewSimpleExprLast200().FromPattern(passwordExpression),
			expr.NewSimpleExprLast200().FromPattern(passwordErrorExpression)),
		genericcli.WithPager(
			expr.NewSimpleExprLast200().FromPattern(pagerExpression)),
		genericcli.WithQuestion(
			expr.NewSimpleExprLast200().FromPattern(questionExpression)),
		genericcli.WithAutoCommands(autoCommands),
		genericcli.WithEchoExprFn(func(c cmd.Cmd) expr.Expr {
			return expr.NewSimpleExpr().FromPattern(fmt.Sprintf(`%s\r*\n`, regexp.QuoteMeta(string(c.Value()))))
		}),
		genericcli.WithTerminalParams(400, 0),
	)
	return genericcli.MakeGenericDevice(cli, connector, opts...)
}
