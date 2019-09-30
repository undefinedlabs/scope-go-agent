package agent

import (
	"fmt"
	"os"
)

func (a *Agent) PrintReport() {
	printReportOnce.Do(func() {
		if a.testingMode && a.recorder.totalSend > 0 {
			if a.recorder.koSend == 0 {
				fmt.Printf("\n** Scope Test Report **\n\n")
				fmt.Println("Access the detailed test report for this build at:")
				fmt.Printf("   %s/external/v1/results/%s\n\n", a.apiEndpoint, a.agentId)
			} else if a.recorder.koSend < a.recorder.totalSend {
				fmt.Printf("\n** Scope Test Report **\n\n")
				fmt.Println("There was a problem sending data to Scope, partial test report for this build at:")
				fmt.Printf("   %s/external/v1/results/%s\n\n", a.apiEndpoint, a.agentId)
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "\n** Scope Test Report **\n\n")
				_, _ = fmt.Fprintf(os.Stderr, "There was a problem sending data to Scope\n")
			}
		}
	})
}
