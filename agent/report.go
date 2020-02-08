package agent

import (
	"fmt"
)

func (a *Agent) PrintReport() {
	printReportOnce.Do(func() {
		if a.testingMode && a.recorder.stats.totalSpans > 0 {
			fmt.Printf("\n** Scope Test Report **\n")
			if a.recorder.stats.spansSent == a.recorder.stats.totalSpans && a.recorder.stats.spansRejected == 0 {
				fmt.Println("Access the detailed test report for this build at:")
				fmt.Printf("   %s\n\n", a.getUrl(fmt.Sprintf("external/v1/results/%s", a.agentId)))
			} else {
				a.recorder.writeStats()
				fmt.Println("There was a problem sending data to Scope.")
				if a.recorder.stats.spansSent > 0 {
					fmt.Println("Partial results for this build are available at:")
					fmt.Printf("   %s\n\n", a.getUrl(fmt.Sprintf("external/v1/results/%s", a.agentId)))
				}
				fmt.Printf("Check the agent logs at %s for more information.\n\n", a.recorderFilename)
			}
		}
	})
}
