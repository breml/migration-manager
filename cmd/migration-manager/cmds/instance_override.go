package cmds

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lxc/incus/v6/shared/units"
	"github.com/spf13/cobra"

	"github.com/FuturFusion/migration-manager/internal/util"
	"github.com/FuturFusion/migration-manager/shared/api"
)

type CmdInstanceOverride struct {
	Global *CmdGlobal
}

func (c *CmdInstanceOverride) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = "override"
	cmd.Short = "Override instance config"
	cmd.Long = `Description:
  Override specific instance configuration values
`

	// Add
	instanceOverrideAddCmd := cmdInstanceOverrideAdd{global: c.Global}
	cmd.AddCommand(instanceOverrideAddCmd.Command())

	// Remove
	instanceOverrideRemoveCmd := cmdInstanceOverrideRemove{global: c.Global}
	cmd.AddCommand(instanceOverrideRemoveCmd.Command())

	// Show
	instanceOverrideShowCmd := cmdInstanceOverrideShow{global: c.Global}
	cmd.AddCommand(instanceOverrideShowCmd.Command())

	// Update
	instanceOverrideUpdateCmd := cmdInstanceOverrideUpdate{global: c.Global}
	cmd.AddCommand(instanceOverrideUpdateCmd.Command())

	// Workaround for subcommand usage errors. See: https://github.com/spf13/cobra/issues/706
	cmd.Args = cobra.NoArgs
	cmd.Run = func(cmd *cobra.Command, args []string) { _ = cmd.Usage() }

	return cmd
}

// Add an instance override.
type cmdInstanceOverrideAdd struct {
	global *CmdGlobal
}

func (c *cmdInstanceOverrideAdd) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = "add <uuid>"
	cmd.Short = "Add an instance override"
	cmd.Long = `Description:
  Add an instance override

  Only a few fields can be set, such as the number of vCPUs or memory. Updating
  other values must be done on through the UI/API of the instance's Source.
`

	cmd.RunE = c.Run

	return cmd
}

func (c *cmdInstanceOverrideAdd) Run(cmd *cobra.Command, args []string) error {
	// Quick checks.
	exit, err := c.global.CheckArgs(cmd, args, 1, 1)
	if exit {
		return err
	}

	UUIDString := args[0]
	UUID, err := uuid.Parse(UUIDString)
	if err != nil {
		return err
	}

	// Add the override.
	override := api.InstanceOverride{
		UUID: UUID,
	}

	override.Comment, err = c.global.Asker.AskString("Comment: ", "", nil)
	if err != nil {
		return err
	}

	val, err := c.global.Asker.AskInt("Number of vCPUs [1] : ", 1, 1024, "1", nil)
	if err != nil {
		return err
	}

	override.NumberCPUs = int(val)

	override.MemoryInBytes, err = c.global.Asker.AskInt("Memory in bytes: [1073741824] ", 1, 1024*1024*1024*1024*1024, "1073741824", nil)
	if err != nil {
		return err
	}

	override.LastUpdate = time.Now().UTC()

	// Insert into database.
	content, err := json.Marshal(override)
	if err != nil {
		return err
	}

	_, err = c.global.doHTTPRequestV1("/instances/"+UUIDString+"/override", http.MethodPost, "", content)
	if err != nil {
		return err
	}

	cmd.Printf("Successfully added new override for instance '%s'.\n", UUIDString)
	return nil
}

// Remove an instance overrirde.
type cmdInstanceOverrideRemove struct {
	global *CmdGlobal
}

func (c *cmdInstanceOverrideRemove) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = "remove <uuid>"
	cmd.Short = "Remove an instance override"
	cmd.Long = `Description:
  Remove an instance override
`

	cmd.RunE = c.Run

	return cmd
}

func (c *cmdInstanceOverrideRemove) Run(cmd *cobra.Command, args []string) error {
	// Quick checks.
	exit, err := c.global.CheckArgs(cmd, args, 1, 1)
	if exit {
		return err
	}

	UUIDString := args[0]

	// Remove the instance override.
	_, err = c.global.doHTTPRequestV1("/instances/"+UUIDString+"/override", http.MethodDelete, "", nil)
	if err != nil {
		return err
	}

	cmd.Printf("Successfully removed override for instance '%s'.\n", UUIDString)
	return nil
}

// Show an instance override.
type cmdInstanceOverrideShow struct {
	global *CmdGlobal

	flagFormat string
}

func (c *cmdInstanceOverrideShow) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = "show <uuid>"
	cmd.Short = "Show an instance override"
	cmd.Long = `Description:
  Show an instance override
`

	cmd.RunE = c.Run
	cmd.Flags().StringVarP(&c.flagFormat, "format", "f", "table", "Format (csv|json|table|yaml|compact)")

	return cmd
}

func (c *cmdInstanceOverrideShow) Run(cmd *cobra.Command, args []string) error {
	// Quick checks.
	exit, err := c.global.CheckArgs(cmd, args, 1, 1)
	if exit {
		return err
	}

	UUIDString := args[0]

	// Get the instance override.
	resp, err := c.global.doHTTPRequestV1("/instances/"+UUIDString+"/override", http.MethodGet, "", nil)
	if err != nil {
		return err
	}

	o, err := parseReturnedInstanceOverride(resp.Metadata)
	if err != nil {
		return err
	}

	override, ok := o.(api.InstanceOverride)
	if !ok {
		return fmt.Errorf("Invalid type for InstanceOverride")
	}

	// Render the table.
	header := []string{"UUID", "Last Update", "Comment", "Num vCPUs", "Memory"}
	data := [][]string{{override.UUID.String(), override.LastUpdate.String(), override.Comment, strconv.Itoa(override.NumberCPUs), units.GetByteSizeStringIEC(override.MemoryInBytes, 2)}}

	return util.RenderTable(cmd.OutOrStdout(), c.flagFormat, header, data, override)
}

// Update an instance override.
type cmdInstanceOverrideUpdate struct {
	global *CmdGlobal
}

func (c *cmdInstanceOverrideUpdate) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = "update <uuid>"
	cmd.Short = "Update instance override"
	cmd.Long = `Description:
  Update instance override

  Only a few fields can be updated, such as the number of vCPUs or memory. Updating
  other values must be done on through the UI/API of the instance's Source.
`

	cmd.RunE = c.Run

	return cmd
}

func (c *cmdInstanceOverrideUpdate) Run(cmd *cobra.Command, args []string) error {
	// Quick checks.
	exit, err := c.global.CheckArgs(cmd, args, 1, 1)
	if exit {
		return err
	}

	UUIDString := args[0]

	// Get the existing instance override.
	resp, err := c.global.doHTTPRequestV1("/instances/"+UUIDString+"/override", http.MethodGet, "", nil)
	if err != nil {
		return err
	}

	o, err := parseReturnedInstanceOverride(resp.Metadata)
	if err != nil {
		return err
	}

	// Prompt for updates.
	switch override := o.(type) {
	case api.InstanceOverride:
		override.Comment, err = c.global.Asker.AskString("Comment: ["+override.Comment+"] ", override.Comment, func(s string) error { return nil })
		if err != nil {
			return err
		}

		val, err := c.global.Asker.AskInt("Number of vCPUs: ["+strconv.Itoa(override.NumberCPUs)+"] ", 1, 1024, strconv.Itoa(override.NumberCPUs), nil)
		if err != nil {
			return err
		}

		if override.NumberCPUs != int(val) {
			override.NumberCPUs = int(val)
		}

		val, err = c.global.Asker.AskInt("Memory in bytes: ["+strconv.FormatInt(override.MemoryInBytes, 10)+"] ", 1, 1024*1024*1024*1024*1024, strconv.FormatInt(override.MemoryInBytes, 10), nil)
		if err != nil {
			return err
		}

		if override.MemoryInBytes != val {
			override.MemoryInBytes = val
		}

		override.LastUpdate = time.Now().UTC()

		o = override
	}

	content, err := json.Marshal(o)
	if err != nil {
		return err
	}

	_, err = c.global.doHTTPRequestV1("/instances/"+UUIDString+"/override", http.MethodPut, "", content)
	if err != nil {
		return err
	}

	cmd.Printf("Successfully updated instance override '%s'.\n", UUIDString)
	return nil
}

func parseReturnedInstanceOverride(i any) (any, error) {
	reJsonified, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	ret := api.InstanceOverride{}
	err = json.Unmarshal(reJsonified, &ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}
