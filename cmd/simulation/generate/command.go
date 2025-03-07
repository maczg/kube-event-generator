package generate

import (
	"errors"
	"fmt"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"math/rand"
	"os"
	"time"
)

var (
	outputFile         string
	outputDir          string
	seed               int64
	timeNow            = time.Now()
	baseScenarioFile   string
	rng                = rand.New(rand.NewSource(seed))
	arrivalLambda      float64
	arrivalScaleFactor float64
	numEvents          int
)

var Cmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen"},
	Short:   "Generate a simulation scenario",
	Long:    `Generate a simulation`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var scn *scenario.Scenario
		if _, err := os.Open(baseScenarioFile); err != nil {
			logrus.Fatalf("base scenario file not found, err: %s", err)
		} else {
			scn, err = scenario.LoadYaml(baseScenarioFile)
			if err != nil {
				logrus.Fatalf("failed to load scenario file, err: %s. Assuming new scenario", err)
			}
		}

		logrus.Infoln("base scenario file loaded, generating new scenario")
		setScenarioDefaultValue(scn)

		params := NewGenerationParams(*scn,
			WithNumEvents(numEvents),
			WithRand(rng),
			WithArrivalLambda(arrivalLambda),
			WithArrivalScaleFactor(arrivalScaleFactor),
		)

		events := generateEvents(params)
		for _, event := range events {
			scn.Events = append(scn.Events, event)
		}

		err := os.Mkdir(outputDir, 0755)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		if outputFile == "" {
			outputFile = fmt.Sprintf("%s/%s.yaml", outputDir, scn.Metadata.Name)
		} else {
			outputFile = fmt.Sprintf("%s/%s", outputDir, outputFile)
		}
		err = scn.Dump(outputFile)
		if err != nil {
			return err
		}

		scn.Describe()
		return nil
	},
}

func init() {
	Cmd.Flags().StringVarP(&outputDir, "output-dir", "d", "scenarios", "Output directory")
	Cmd.Flags().StringVarP(&baseScenarioFile, "default-scenario", "i", "base-scenario.yaml", "default scenario file")
	Cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file name")
	Cmd.Flags().Int64VarP(&seed, "seed", "s", time.Now().Unix(), "Random seed (0 means use current time)")
	Cmd.Flags().Float64VarP(&arrivalLambda, "arrival-lambda", "a", 1.0, "Arrival lambda")
	Cmd.Flags().Float64VarP(&arrivalScaleFactor, "arrival-scale-factor", "b", 5.0, "Arrival scale factor")
	Cmd.Flags().IntVarP(&numEvents, "event-count", "e", 10, "Number of events to generate")
}
