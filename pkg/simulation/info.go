package simulation

import (
	"cmp"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/maczg/kube-event-generator/pkg/scenario"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"slices"
	"time"
)

func (s *Simulation) Info() {
	t := table.NewWriter()

	file, err := os.Create(fmt.Sprintf("%s/simulation-info.txt", s.resultDir))
	if err != nil {
		logrus.Errorf("error creating file: %v", err)
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			logrus.Errorf("error closing file: %v", err)
		}
	}(file)

	writers := io.MultiWriter(file, os.Stdout)
	t.SetOutputMirror(writers)
	//t.SetAllowedRowLength(80)
	t.AppendRow(table.Row{"Simulation Information", "", "", ""})
	t.AppendSeparator()
	t.AppendRow(table.Row{"Simulation ID", "Scenario", "Start Time", "Kube Host"})

	var host string
	if s.kubeMgr.RestCfg().Host != "" {
		host = s.kubeMgr.RestCfg().Host
	} else {
		host = "unknown"
	}
	t.AppendRow(table.Row{
		s.ID,
		s.Scenario.Metadata.Name,
		s.startTime.Format(time.RFC3339),
		host,
	})
	t.AppendSeparator()
	t.AppendRow(table.Row{"Cluster Nodes", "", "", ""})
	t.AppendSeparator()
	t.AppendRow(table.Row{"Node", "CPU", "Memory", ""})
	for _, n := range s.Scenario.Cluster.Nodes {
		t.AppendRow(table.Row{
			n.Name,
			n.Status.Capacity.Cpu(),
			n.Status.Capacity.Memory(),
			"",
		})
	}
	t.AppendSeparator()
	t.AppendRow(table.Row{"Events", "", "", ""})
	t.AppendSeparator()
	t.AppendRow(table.Row{"# events", "longest", "long running", "sim end"})
	numEvents := len(s.Scenario.Events)
	// Using slices.MaxFunc to get the event with the longest duration (assumed to be the one with the maximum From value)
	longest := slices.MaxFunc(s.Scenario.Events, func(a, b scenario.Event) int {
		return cmp.Compare(a.From, b.From)
	})
	endAt := s.startTime.Add(longest.From)

	durationEvent := make([]scenario.Event, 0)
	for _, ev := range s.Scenario.Events {
		if ev.Duration <= 0 {
			durationEvent = append(durationEvent, ev)
			logrus.Errorf("Event %s has a negative or non-zero duration", ev.Name)
		}
	}

	t.AppendRow(table.Row{
		numEvents,
		longest.From,
		len(durationEvent),
		endAt.Format(time.RFC3339),
	})
	t.Render()
}
