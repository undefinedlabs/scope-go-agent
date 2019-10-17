package agent

import (
	"fmt"
)

func (a *Agent) PrintReport() {
	printReportOnce.Do(func() {
		if a.testingMode && a.recorder.totalSend > 0 {
			fmt.Printf("\n** Scope Test Report **\n\n")
			if a.recorder.koSend == 0 {
				fmt.Println("Access the detailed test report for this build at:")
				fmt.Printf("   %s/external/v1/results/%s\n\n", a.apiEndpoint, a.agentId)
			} else if a.recorder.koSend < a.recorder.totalSend {
				fmt.Println("There was a problem sending data to Scope. Partial test report for this build at:")
				fmt.Printf("   %s/external/v1/results/%s\n\n", a.apiEndpoint, a.agentId)
				fmt.Println("Check the agent logs for more information.")
			} else {
				fmt.Println("There was a problem sending data to Scope. Check the agent logs for more information.")
			}
		}
	})
}
