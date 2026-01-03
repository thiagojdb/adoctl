package cmd

import (
	"fmt"

	"adoctl/pkg/devops"

	"github.com/spf13/cobra"
)

type CommandBuilder struct {
	cmd *cobra.Command
}

func NewCommand(name, short, long string) *CommandBuilder {
	return &CommandBuilder{
		cmd: &cobra.Command{
			Use:     name,
			Short:   short,
			Long:    long,
			Example: "",
		},
	}
}

func (b *CommandBuilder) WithExample(example string) *CommandBuilder {
	b.cmd.Example = example
	return b
}

func (b *CommandBuilder) WithTimeout() *CommandBuilder {
	original := b.cmd.RunE
	if original == nil {
		original = func(cmd *cobra.Command, args []string) error { return nil }
	}
	b.cmd.RunE = func(cmd *cobra.Command, args []string) error {
		_, cancel := GetContext()
		defer cancel()
		return original(cmd, args)
	}
	return b
}

func (b *CommandBuilder) WithPRManager(fn func(*devops.DevOpsService) error) *CommandBuilder {
	original := b.cmd.RunE
	if original == nil {
		original = func(cmd *cobra.Command, args []string) error { return nil }
	}
	b.cmd.RunE = func(cmd *cobra.Command, args []string) error {
		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("failed to create devops service: %w", err)
		}
		defer svc.Close()
		return fn(svc)
	}
	return b
}

func (b *CommandBuilder) WithDeploymentManager(fn func(*devops.DevOpsService) error) *CommandBuilder {
	original := b.cmd.RunE
	if original == nil {
		original = func(cmd *cobra.Command, args []string) error { return nil }
	}
	b.cmd.RunE = func(cmd *cobra.Command, args []string) error {
		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("failed to create devops service: %w", err)
		}
		defer svc.Close()
		return fn(svc)
	}
	return b
}

func (b *CommandBuilder) WithArgsValidation(minArgs int) *CommandBuilder {
	b.cmd.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) < minArgs {
			return fmt.Errorf("requires at least %d argument(s)", minArgs)
		}
		return nil
	}
	return b
}

func (b *CommandBuilder) Build() *cobra.Command {
	return b.cmd
}

func AddCommand(parent, child *cobra.Command) {
	parent.AddCommand(child)
}

func AddCommands(parent *cobra.Command, children ...*cobra.Command) {
	for _, child := range children {
		parent.AddCommand(child)
	}
}
