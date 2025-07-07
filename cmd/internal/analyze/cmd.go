package analyze

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/spf13/cobra"
)

// NewCommand creates the analyze command
func NewCommand(log *logger.Logger) *cobra.Command {
	var (
		simulations  []string
		baseDir      string
		outputDir    string
		targetMetric string
		exportFormat string
	)

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze simulation results",
		Long: `Analyze and compare multiple simulation results to understand scheduler weight impacts.

This command runs the Python analyzer to compare metrics across different simulations
and identify optimal scheduler plugin weight configurations.`,
	}

	// Add subcommands
	cmd.AddCommand(
		newCompareCommand(log, &simulations, &baseDir, &outputDir, &targetMetric, &exportFormat),
		newFragmentationCommand(log, &baseDir),
		newReportCommand(log, &baseDir, &outputDir),
	)

	return cmd
}

func newCompareCommand(log *logger.Logger, simulations *[]string, baseDir, outputDir, targetMetric, exportFormat *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare multiple simulation results",
		Long: `Compare multiple simulation runs to analyze the impact of scheduler weights on performance metrics.

Examples:
  # Compare three simulations
  keg analyze compare -s sim1 sim2 sim3

  # Compare with custom target metric
  keg analyze compare -s sim1 sim2 -m fragmentation_index

  # Export to specific format
  keg analyze compare -s sim1 sim2 --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompareAnalysis(log, *simulations, *baseDir, *outputDir, *targetMetric, *exportFormat)
		},
	}

	cmd.Flags().StringSliceVarP(simulations, "simulations", "s", []string{}, "Simulation directories to compare (required)")
	cmd.Flags().StringVarP(baseDir, "base-dir", "b", "results", "Base directory containing simulation results")
	cmd.Flags().StringVarP(outputDir, "output", "o", "analysis_report", "Output directory for analysis report")
	cmd.Flags().StringVarP(targetMetric, "metric", "m", "scheduling_efficiency", "Target metric to optimize for")
	cmd.Flags().StringVar(exportFormat, "format", "all", "Export format (csv, json, html, all)")

	cmd.MarkFlagRequired("simulations")

	return cmd
}

func newFragmentationCommand(log *logger.Logger, dataDir *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fragmentation [simulation]",
		Short: "Analyze resource fragmentation for a simulation",
		Long:  `Calculate and display resource fragmentation metrics for a single simulation.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			simDir := filepath.Join(*dataDir, args[0])
			return runFragmentationAnalysis(log, simDir)
		},
	}

	cmd.Flags().StringVarP(dataDir, "base-dir", "b", "results", "Base directory containing simulation results")

	return cmd
}

func newReportCommand(log *logger.Logger, dataDir, outputDir *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report [simulation]",
		Short: "Generate detailed report for a simulation",
		Long:  `Generate a comprehensive analysis report for a single simulation run.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			simDir := filepath.Join(*dataDir, args[0])
			return runSingleReport(log, simDir, *outputDir)
		},
	}

	cmd.Flags().StringVarP(dataDir, "base-dir", "b", "results", "Base directory containing simulation results")
	cmd.Flags().StringVarP(outputDir, "output", "o", "", "Output directory (default: <simulation>/report)")

	return cmd
}

func runCompareAnalysis(log *logger.Logger, simulations []string, baseDir, outputDir, targetMetric, exportFormat string) error {
	// Check if Python analyzer exists
	analyzerPath := filepath.Join("analyzer", "compare_simulations.py")
	if _, err := os.Stat(analyzerPath); os.IsNotExist(err) {
		return fmt.Errorf("analyzer not found at %s", analyzerPath)
	}

	// Build command
	args := []string{
		analyzerPath,
		"--simulations",
	}
	args = append(args, simulations...)
	args = append(args,
		"--base-dir", baseDir,
		"--output", outputDir,
		"--target-metric", targetMetric,
		"--export-format", exportFormat,
	)

	// Execute Python analyzer
	cmd := exec.Command("python3", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.WithFields(map[string]interface{}{
		"simulations": len(simulations),
		"base_dir":    baseDir,
		"output":      outputDir,
		"metric":      targetMetric,
	}).Info("Running comparison analysis")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	return nil
}

func runFragmentationAnalysis(log *logger.Logger, simDir string) error {
	// Check if simulation directory exists
	if _, err := os.Stat(simDir); os.IsNotExist(err) {
		return fmt.Errorf("simulation directory not found: %s", simDir)
	}

	// Build command
	analyzerPath := filepath.Join("analyzer", "main.py")
	cmd := exec.Command("python3", analyzerPath, "--data-dir", simDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.WithFields(map[string]interface{}{
		"simulation": simDir,
	}).Info("Running fragmentation analysis")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("fragmentation analysis failed: %w", err)
	}

	return nil
}

func runSingleReport(log *logger.Logger, simDir, outputDir string) error {
	// Check if simulation directory exists
	if _, err := os.Stat(simDir); os.IsNotExist(err) {
		return fmt.Errorf("simulation directory not found: %s", simDir)
	}

	// Default output directory
	if outputDir == "" {
		outputDir = filepath.Join(simDir, "report")
	}

	// For now, use the fragmentation analyzer
	// TODO: Extend to generate full report
	return runFragmentationAnalysis(log, simDir)
}

// Helper function to check if Python is available
func checkPythonDependencies() error {
	// Check Python
	if _, err := exec.LookPath("python3"); err != nil {
		return fmt.Errorf("python3 not found in PATH")
	}

	// Check required packages
	cmd := exec.Command("python3", "-c", "import pandas, numpy, matplotlib, scipy, yaml")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("required Python packages not installed. Run: pip install -r analyzer/requirements.txt")
	}

	return nil
}

// ListSimulations lists available simulation results
func ListSimulations(baseDir string) ([]string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read results directory: %w", err)
	}

	var simulations []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			// Check if it contains simulation data
			eventFile := filepath.Join(baseDir, entry.Name(), "event_history.csv")
			if _, err := os.Stat(eventFile); err == nil {
				simulations = append(simulations, entry.Name())
			}
		}
	}

	return simulations, nil
}