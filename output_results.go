package main

import (
	"fmt"
	"io"
	"strings"
)

// RunOutputTask will read in the analysis.json file and output
// any useful information to o or any errors to e.
func RunOutputTask(o io.Writer, e io.Writer, analysisResults AnalysisResults) {
	for _, projects := range analysisResults {
		for project, results := range projects {
			for _, result := range results {
				if result.Status != 0 {
					fmt.Fprintf(o, "Potential issue discovered in %s: %s\n", project, result.CheckName)
					fmt.Fprintf(o, "  Details:\n")
					for _, info := range result.Info {
						fmt.Fprintf(o, "    %s\n", strings.Replace(strings.TrimSpace(info.Message), "\n", "\n    ", -1))
					}
					for _, event := range result.Events {
						fmt.Fprintf(o, "    %s\n", strings.Replace(strings.TrimSpace(event.Message), "\n", "\n    ", -1))
					}
				}
			}
		}
	}
}
