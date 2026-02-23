package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"ping-tracker/tracker"
	"ping-tracker/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	interval := flag.Duration("interval", 3*time.Second, "scan interval")
	noPing := flag.Bool("no-ping", false, "disable ping measurements (faster, no TCP probes)")
	filter := flag.String("filter", "", "initial app name filter (substring match)")
	flag.Parse()

	checkPrivileges()

	t := tracker.NewTracker(*interval, !*noPing)
	t.Start()
	defer t.Stop()

	model := tui.NewModel(t)
	if *filter != "" {
		model.SetFilter(*filter)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
