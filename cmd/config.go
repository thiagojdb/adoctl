package cmd

import (
	"fmt"

	"adoctl/pkg/config"
	"adoctl/pkg/errors"

	"github.com/spf13/cobra"
)

var (
	configProfileName string
	configOrg         string
	configProject     string
	configToken       string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage adoctl configuration and profiles",
	Long:  `Manage adoctl configuration, including Azure DevOps profiles for multi-organization support.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration including active profile settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		fmt.Println("Current Configuration:")
		fmt.Println("======================")
		fmt.Printf("Active Profile: %s\n", func() string {
			if cfg.ActiveProfile == "" {
				return "(none)"
			}
			return cfg.ActiveProfile
		}())
		fmt.Println()
		fmt.Printf("Organization: %s\n", cfg.Azure.Organization)
		fmt.Printf("Project: %s\n", cfg.Azure.Project)
		fmt.Printf("API Version: %s\n", cfg.Azure.APIVersion)
		fmt.Printf("Token: %s\n", func() string {
			if cfg.Azure.PersonalAccessToken == "" {
				return "(not set)"
			}
			return "(set)"
		}())
		fmt.Println()
		fmt.Printf("Thread Pool Size: %d\n", cfg.ThreadPool.ParallelProcesses)

		if len(cfg.Profiles) > 0 {
			fmt.Println()
			fmt.Println("Available Profiles:")
			for _, p := range cfg.Profiles {
				active := ""
				if cfg.IsProfileActive(p.Name) {
					active = " (active)"
				}
				fmt.Printf("  - %s%s\n", p.Name, active)
				fmt.Printf("      Org: %s, Project: %s\n", p.Azure.Organization, p.Azure.Project)
			}
		}

		return nil
	},
}

var configProfilesCmd = &cobra.Command{
	Use:     "profiles",
	Aliases: []string{"profile"},
	Short:   "Manage configuration profiles",
	Long:    `List, add, remove, and switch between Azure DevOps configuration profiles.`,
}

var configProfilesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		profiles := cfg.ListProfiles()
		if len(profiles) == 0 {
			fmt.Println("No profiles configured.")
			fmt.Println("Use 'adoctl config profiles add --name <name>' to create one.")
			return nil
		}

		fmt.Println("Profiles:")
		for _, name := range profiles {
			profile, _ := cfg.GetProfile(name)
			active := ""
			if cfg.IsProfileActive(name) {
				active = " *active*"
			}
			fmt.Printf("  %s%s\n", name, active)
			fmt.Printf("    Organization: %s\n", profile.Azure.Organization)
			fmt.Printf("    Project: %s\n", profile.Azure.Project)
		}

		return nil
	},
}

var configProfilesAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new profile",
	Long:  `Add a new Azure DevOps configuration profile.`,
	Example: `  # Add a profile interactively
  adoctl config profiles add --name work --org MyOrg --project MyProject

  # Add a profile with token (not recommended - use environment variable instead)
  adoctl config profiles add --name work --org MyOrg --project MyProject --token $AZURE_PAT`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if configProfileName == "" {
			return errors.ConfigError("profile name is required (--name)")
		}
		if configOrg == "" {
			return errors.ConfigError("organization is required (--org)")
		}
		if configProject == "" {
			return errors.ConfigError("project is required (--project)")
		}

		cfg, err := config.Load()
		if err != nil {
			cfg = &config.Config{}
		}

		profile := config.Profile{
			Name: configProfileName,
			Azure: config.AzureConfig{
				Organization:        configOrg,
				Project:             configProject,
				PersonalAccessToken: configToken,
				APIVersion:          "7.1",
			},
		}

		if err := cfg.AddProfile(profile); err != nil {
			return err
		}

		if err := config.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("Profile '%s' added successfully.\n", configProfileName)
		fmt.Printf("Use 'adoctl config profiles use --name %s' to activate it.\n", configProfileName)

		return nil
	},
}

var configProfilesRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configProfileName == "" {
			return errors.ConfigError("profile name is required (--name)")
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if err := cfg.RemoveProfile(configProfileName); err != nil {
			return err
		}

		if err := config.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("Profile '%s' removed successfully.\n", configProfileName)
		return nil
	},
}

var configProfilesUseCmd = &cobra.Command{
	Use:   "use",
	Short: "Switch to a profile",
	Long:  `Set the active profile for subsequent commands.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if configProfileName == "" {
			return errors.ConfigError("profile name is required (--name)")
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if err := cfg.SetProfile(configProfileName); err != nil {
			return err
		}

		if err := config.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("Switched to profile '%s'.\n", configProfileName)
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.GetConfigPath()
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	},
}

func init() {
	// Profile management flags
	configProfilesAddCmd.Flags().StringVar(&configProfileName, "name", "", "Profile name (required)")
	configProfilesAddCmd.Flags().StringVar(&configOrg, "org", "", "Azure DevOps organization (required)")
	configProfilesAddCmd.Flags().StringVar(&configProject, "project", "", "Azure DevOps project (required)")
	configProfilesAddCmd.Flags().StringVar(&configToken, "token", "", "Personal access token (optional, prefer env var)")
	if err := configProfilesAddCmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
	if err := configProfilesAddCmd.MarkFlagRequired("org"); err != nil {
		panic(err)
	}
	if err := configProfilesAddCmd.MarkFlagRequired("project"); err != nil {
		panic(err)
	}

	configProfilesRemoveCmd.Flags().StringVar(&configProfileName, "name", "", "Profile name (required)")
	if err := configProfilesRemoveCmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}

	configProfilesUseCmd.Flags().StringVar(&configProfileName, "name", "", "Profile name (required)")
	if err := configProfilesUseCmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}

	// Add commands
	configProfilesCmd.AddCommand(configProfilesListCmd)
	configProfilesCmd.AddCommand(configProfilesAddCmd)
	configProfilesCmd.AddCommand(configProfilesRemoveCmd)
	configProfilesCmd.AddCommand(configProfilesUseCmd)

	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configProfilesCmd)
	configCmd.AddCommand(configPathCmd)
}
