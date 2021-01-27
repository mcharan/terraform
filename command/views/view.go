package views

import (
	"github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

type View struct {
	ui              cli.Ui
	colorize        *colorstring.Colorize
	compactWarnings bool
	outputColumns   int
	errorColumns    int
	configSources   func() map[string][]byte
}

func NewView(ui cli.Ui, color, compactWarnings bool, outputColumns, errorColumns int, configSources func() map[string][]byte) View {
	return View{
		ui: ui,
		colorize: &colorstring.Colorize{
			Colors:  colorstring.DefaultColors,
			Disable: !color,
			Reset:   true,
		},
		compactWarnings: compactWarnings,
		outputColumns:   outputColumns,
		errorColumns:    errorColumns,
		configSources:   configSources,
	}
}

func (v *View) output(s string) {
	v.ui.Output(s)
}

func (v *View) Diagnostics(diags tfdiags.Diagnostics) {
	diags.Sort()

	if len(diags) == 0 {
		return
	}

	diags = diags.ConsolidateWarnings(1)

	// Since warning messages are generally competing
	if v.compactWarnings {
		// If the user selected compact warnings and all of the diagnostics are
		// warnings then we'll use a more compact representation of the warnings
		// that only includes their summaries.
		// We show full warnings if there are also errors, because a warning
		// can sometimes serve as good context for a subsequent error.
		useCompact := true
		for _, diag := range diags {
			if diag.Severity() != tfdiags.Warning {
				useCompact = false
				break
			}
		}
		if useCompact {
			msg := format.DiagnosticWarningsCompact(diags, v.colorize)
			msg = "\n" + msg + "\nTo see the full warning notes, run Terraform without -compact-warnings.\n"
			v.ui.Warn(msg)
			return
		}
	}

	for _, diag := range diags {
		var msg string
		if v.colorize.Disable {
			msg = format.DiagnosticPlain(diag, v.configSources(), v.errorColumns)
		} else {
			msg = format.Diagnostic(diag, v.configSources(), v.colorize, v.errorColumns)
		}

		switch diag.Severity() {
		case tfdiags.Error:
			v.ui.Error(msg)
		case tfdiags.Warning:
			v.ui.Warn(msg)
		default:
			v.ui.Output(msg)
		}
	}
}
