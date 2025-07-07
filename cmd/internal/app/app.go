package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/maczg/kube-event-generator/cmd/internal/analyze"
	"github.com/maczg/kube-event-generator/cmd/internal/cluster"
	"github.com/maczg/kube-event-generator/cmd/internal/scenario"
	"github.com/maczg/kube-event-generator/cmd/internal/simulation"
	"github.com/maczg/kube-event-generator/pkg/logger"
)

// App represents the main application.
type App struct {
	rootCmd *cobra.Command
	logger  *logger.Logger
	config  *Config
}

// Config holds application-wide configuration.
type Config struct {
	LogFormat  string
	LogFile    string
	Kubeconfig string
	Verbose    bool
}

// New creates a new application instance.
func New() *App {
	app := &App{
		logger: logger.Default(),
		config: &Config{},
	}

	app.setupCommands()

	return app
}

// setupCommands initializes the command tree.
func (app *App) setupCommands() {
	app.rootCmd = &cobra.Command{
		Use:     "keg",
		Aliases: []string{"kube-event-generator"},
		Short:   "Kubernetes Event Generator",
		Long: `Kubernetes Event Generator (KEG) is a tool to simulate events in a Kubernetes cluster.
It allows you to define scenarios, generate workloads, and analyze scheduler behavior.`,
		PersistentPreRunE: app.persistentPreRun,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	// Global flags.
	app.rootCmd.PersistentFlags().BoolVarP(&app.config.Verbose, "verbose", "v", false, "Enable verbose logging")
	app.rootCmd.PersistentFlags().StringVar(&app.config.LogFormat, "log-format", "text", "Log format (text, json)")
	app.rootCmd.PersistentFlags().StringVar(&app.config.LogFile, "log-file", "", "Log file path (default: stdout)")
	app.rootCmd.PersistentFlags().StringVar(&app.config.Kubeconfig, "kubeconfig", "", "Path to kubeconfig file")

	// Add sub-commands.
	app.rootCmd.AddCommand(
		cluster.NewCommand(app.logger),
		simulation.NewCommand(app.logger),
		scenario.NewCommand(app.logger),
		analyze.NewCommand(app.logger),
		app.versionCommand(),
		app.completionCommand(),
	)
}

// Execute runs the application.
func (app *App) Execute() error {
	return app.rootCmd.Execute()
}

// persistentPreRun sets up logging and validates global configuration.
func (app *App) persistentPreRun(cmd *cobra.Command, args []string) error {
	// Configure logger.
	app.logger.SetVerbose(app.config.Verbose)

	if app.config.LogFormat == "json" {
		app.logger.SetJSONFormat(true)
	}

	if app.config.LogFile != "" {
		if err := app.logger.SetOutput(app.config.LogFile); err != nil {
			return fmt.Errorf("failed to set log output: %w", err)
		}
	}

	// Set kubeconfig environment variable if provided.
	if app.config.Kubeconfig != "" {
		os.Setenv("KUBECONFIG", app.config.Kubeconfig)
	}

	return nil
}

// versionCommand returns the version command.
func (app *App) versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("kube-event-generator version %s\n", getVersion())
		},
	}
}

// completionCommand returns the completion command.
func (app *App) completionCommand() *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for keg.

To load completions:

Bash:
  $ source <(keg completion bash)
  
  # To load completions for each session, execute once:
  # Linux:
  $ keg completion bash > /etc/bash_completion.d/keg
  
  # macOS:
  $ keg completion bash > /usr/local/etc/bash_completion.d/keg

Zsh:
  $ source <(keg completion zsh)
  
  # To load completions for each session, execute once:
  $ keg completion zsh > "${fpath[1]}/_keg"

Fish:
  $ keg completion fish | source
  
  # To load completions for each session, execute once:
  $ keg completion fish > ~/.config/fish/completions/keg.fish

PowerShell:
  PS> keg completion powershell | Out-String | Invoke-Expression
  
  # To load completions for every new session, run:
  PS> keg completion powershell > keg.ps1
  # and source this file from your PowerShell profile.
`,
	}

	completionCmd.AddCommand(
		&cobra.Command{
			Use:   "bash",
			Short: "Generate bash completion script",
			RunE: func(cmd *cobra.Command, args []string) error {
				return app.rootCmd.GenBashCompletion(os.Stdout)
			},
		},
		&cobra.Command{
			Use:   "zsh",
			Short: "Generate zsh completion script",
			RunE: func(cmd *cobra.Command, args []string) error {
				return app.rootCmd.GenZshCompletion(os.Stdout)
			},
		},
		&cobra.Command{
			Use:   "fish",
			Short: "Generate fish completion script",
			RunE: func(cmd *cobra.Command, args []string) error {
				return app.rootCmd.GenFishCompletion(os.Stdout, true)
			},
		},
		&cobra.Command{
			Use:   "powershell",
			Short: "Generate powershell completion script",
			RunE: func(cmd *cobra.Command, args []string) error {
				return app.rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			},
		},
	)

	return completionCmd
}

// getVersion returns the application version.
func getVersion() string {
	// This could be set via ldflags during build.
	version := os.Getenv("KEG_VERSION")
	if version == "" {
		version = "dev"
	}

	return version
}
