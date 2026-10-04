/*
ciscoxr.go implements a Cisco IOS-XR CLI profile using genericcli.

IOS-XR differs from classic IOS/Catalyst in ways that matter to gnetcli:
  - The exec prompt is `RP/0/RSP0/CPU0:hostname#` (route-processor path + `:` +
    hostname), and config/admin submodes render as `...hostname(config)#`,
    `...hostname(config-if)#`, `...hostname(admin)#`, etc.
  - There is NO privileged (`enable`) mode — the session is already privileged,
    so sending `enable` (as the IOS profile does) yields `% Invalid input` and,
    worse, leaves the exec loop waiting on a `Password:` answer that never
    comes, desyncing the session (surfaces to annet-oil as generic_error).
  - The pager and `terminal length/width 0` disabling are the same as IOS.

This profile keeps the IOS error/login/pager handling but drops `enable` and
uses an XR-aware prompt. Kept separate from NewDevice so the Catalyst path is
untouched (zero blast radius).
*/
package cisco

import (
	"github.com/annetutil/gnetcli/pkg/cmd"
	"github.com/annetutil/gnetcli/pkg/device/genericcli"
	"github.com/annetutil/gnetcli/pkg/expr"
	"github.com/annetutil/gnetcli/pkg/streamer"
)

const (
	// Matches `RP/0/RSP0/CPU0:host#`, `...host(config)#`, `...host(config-if)#`,
	// `...host(admin)#`, and user mode `...host>`. The base class allows the
	// `/` and `:` of the RP path; a single optional parenthetical covers any
	// config/admin submode.
	xrPromptExpression = `(?P<prompt>[\w\-.:/]+(\([^)]+\))?)(>|#)$`
)

var xrAutoCommands = []cmd.Cmd{
	cmd.NewCmd("terminal length 0", cmd.WithErrorIgnore()),
	cmd.NewCmd("terminal width 0", cmd.WithErrorIgnore()),
	cmd.NewCmd("terminal exec prompt no-timestamp", cmd.WithErrorIgnore()),
	cmd.NewCmd("terminal monitor disable", cmd.WithErrorIgnore()),
}

// NewXRDevice builds a genericcli device for Cisco IOS-XR over ssh/telnet.
func NewXRDevice(connector streamer.Connector, opts ...genericcli.GenericDeviceOption) genericcli.GenericDevice {
	cli := genericcli.MakeGenericCLI(
		expr.NewSimpleExprLast200().FromPattern(xrPromptExpression),
		expr.NewSimpleExprLast200().FromPattern(errorExpression),
		genericcli.WithLoginExprs(
			expr.NewSimpleExprLast200().FromPattern(loginExpression),
			expr.NewSimpleExprLast200().FromPattern(passwordExpression),
			expr.NewSimpleExprLast200().FromPattern(passwordErrorExpression)),
		genericcli.WithPager(
			expr.NewSimpleExprLast200().FromPattern(pagerExpression)),
		genericcli.WithQuestion(
			expr.NewSimpleExprLast200().FromPattern(questionExpression)),
		genericcli.WithAutoCommands(xrAutoCommands),
		genericcli.WithTerminalParams(400, 0),
	)
	return genericcli.MakeGenericDevice(cli, connector, opts...)
}
