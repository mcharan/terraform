package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/command/views"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// OutputCommand is a Command implementation that reads an output
// from a Terraform state and prints it.
type OutputCommand struct {
	Meta

	// Flags
	name       string
	jsonOutput bool
	rawOutput  bool
	statePath  string
}

func (c *OutputCommand) Run(args []string) int {
	// Parse and validate flags
	err := c.ParseFlags(args)
	if err != nil {
		c.Ui.Error(err.Error())
		c.Ui.Error(c.Help())
		return 1
	}

	view := c.View()

	// Fetch data from state
	outputs, diags := c.Outputs()
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Render the view
	viewDiags := view.Output(c.name, outputs)
	diags = diags.Append(viewDiags)

	view.Diagnostics(diags)

	if diags.HasErrors() {
		return 1
	}

	return 0
}

func (c *OutputCommand) ParseFlags(args []string) error {
	args = c.Meta.process(args)
	cmdFlags := c.Meta.defaultFlagSet("output")
	cmdFlags.BoolVar(&c.jsonOutput, "json", false, "json")
	cmdFlags.BoolVar(&c.rawOutput, "raw", false, "raw")
	cmdFlags.StringVar(&c.statePath, "state", "", "path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return fmt.Errorf("Error parsing command-line flags: %s\n", err.Error())
	}

	args = cmdFlags.Args()
	if len(args) > 1 {
		return fmt.Errorf("The output command expects exactly one argument with the name\n" +
			"of an output variable or no arguments to show all outputs.\n")
	}

	if c.jsonOutput && c.rawOutput {
		return fmt.Errorf("The -raw and -json options are mutually-exclusive.\n")
	}

	if c.rawOutput && len(args) == 0 {
		return fmt.Errorf("You must give the name of a single output value when using the -raw option.\n")
	}

	if len(args) > 0 {
		c.name = args[0]
	}

	return nil
}

func (c *OutputCommand) View() views.Output {
	view := c.Meta.View()
	switch {
	case c.jsonOutput:
		return &views.OutputJSON{View: view}
	case c.rawOutput:
		return &views.OutputRaw{View: view}
	default:
		return &views.OutputText{View: view}
	}
}

func (c *OutputCommand) Outputs() (map[string]*states.OutputValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Allow state path override
	if c.statePath != "" {
		c.Meta.statePath = c.statePath
	}

	// Load the backend
	b, backendDiags := c.Backend(nil)
	diags = diags.Append(backendDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	// This is a read-only command
	c.ignoreRemoteBackendVersionConflict(b)

	env, err := c.Workspace()
	if err != nil {
		diags = diags.Append(fmt.Errorf("Error selecting workspace: %s", err))
		return nil, diags
	}

	// Get the state
	stateStore, err := b.StateMgr(env)
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to load state: %s", err))
		return nil, diags
	}

	if err := stateStore.RefreshState(); err != nil {
		diags = diags.Append(fmt.Errorf("Failed to load state: %s", err))
		return nil, diags
	}

	state := stateStore.State()
	if state == nil {
		state = states.NewState()
	}

	return state.RootModule().OutputValues, nil
}

func (c *OutputCommand) Help() string {
	helpText := `
Usage: terraform output [options] [NAME]

  Reads an output variable from a Terraform state file and prints
  the value. With no additional arguments, output will display all
  the outputs for the root module.  If NAME is not specified, all
  outputs are printed.

Options:

  -state=path      Path to the state file to read. Defaults to
                   "terraform.tfstate".

  -no-color        If specified, output won't contain any color.

  -json            If specified, machine readable output will be
                   printed in JSON format.

  -raw             For value types that can be automatically
                   converted to a string, will print the raw
                   string directly, rather than a human-oriented
                   representation of the value.
`
	return strings.TrimSpace(helpText)
}

func (c *OutputCommand) Synopsis() string {
	return "Show output values from your root module"
}
