package main

import (
	"fmt"
	"io"
	"strings"
)

// RunOutputTask will read in the analysis.json file and output
// any useful information to o or any errors to e.
func RunOutputTask(o io.Writer, e io.Writer, analysisResult AnalysisResult) {
	// TODO: handle analysisResult.Platform.
	for _, projectResult := range analysisResult.Projects {
		for _, checkResult := range projectResult.Results {
			if !checkResult.Ok {
				fmt.Fprintf(o, "Potential issue discovered in project %s: %s\n", projectResult.Project, checkResult.CheckName)
				fmt.Fprintf(o, "  Details:\n")
				for _, info := range checkResult.Info {
					fmt.Fprintf(o, "    %s\n", strings.Replace(strings.TrimSpace(info.Message), "\n", "\n    ", -1))
				}
				for _, event := range checkResult.Events {
					fmt.Fprintf(o, "    %s\n", strings.Replace(strings.TrimSpace(event.Message), "\n", "\n    ", -1))
				}
			}
		}
	}
}
