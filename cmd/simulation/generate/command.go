package generate

import (
	"errors"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/logger"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/spf13/cobra"
	"math/rand"
	"os"
	"time"
)

var (
	outputFile          string
	outputDir           string
	seed                int64
	timeNow             = time.Now()
	defaultScenarioFile string
	rng                 = rand.New(rand.NewSource(seed))
)

var Cmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen"},
	Short:   "Generate a simulation scenario",
	Long:    `Generate a simulation`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logger.NewLogger(logger.LevelInfo, "generate")

		var scn *scenario.Scenario

		if _, err := os.Open(defaultScenarioFile); err != nil {
			logger.Error("scenario file not found, err: %s. Assuming new scenario", err)
			scn = generateDefaultScenario()
		} else {
			scn, err = scenario.LoadYaml(defaultScenarioFile)
			if err != nil {
				logger.Error("failed to load scenario file, err: %s. Assuming new scenario", err)
				scn = generateDefaultScenario()
			}
			scn.Name = fmt.Sprintf("base-generated-%s", timeNow.Format("2006-01-02-15-04-05"))
		}
		logger.Info("working with scenario %s", scn.Name)
		params := NewGenerationParams(*scn, WithRand(rng))

		events := generateEvents(params)
		for _, event := range events {
			scn.Events = append(scn.Events, event)
		}

		err := os.Mkdir(outputDir, 0755)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}

		outputFile = fmt.Sprintf("%s/%s", outputDir, outputFile)
		err = scn.Dump(outputFile)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	Cmd.Flags().StringVarP(&outputDir, "output-dir", "d", "scenarios", "Output directory")
	Cmd.Flags().StringVarP(&defaultScenarioFile, "def-scn", "f", "", "Default scenario file")
	Cmd.Flags().StringVarP(&outputFile, "output", "o", fmt.Sprintf("scenario-%s.yaml", timeNow.Format("2006-01-02-15-04-05")), "Output file name")
	Cmd.Flags().Int64VarP(&seed, "seed", "s", time.Now().Unix(), "Random seed (0 means use current time)")
}
