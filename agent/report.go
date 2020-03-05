package agent

import (
	"encoding/json"
	"fmt"
)

func (a *Agent) PrintReport() {
	a.printReportOnce.Do(func() {
		stats := a.recorder.Stats()
		if a.testingMode && stats.HasTests() {
			fmt.Printf("\n** Scope Test Report **\n")
			if !stats.HasTestsNotSent() && !stats.HasTestRejected() {
				fmt.Println("Access the detailed test report for this build at:")
				fmt.Printf("   %s\n\n", a.getUrl(fmt.Sprintf("external/v1/results/%s", a.agentId)))
			} else {
				if !a.debugMode {
					a.logMetadata()
				}
				stats.Write()
				fmt.Println("There was a problem sending data to Scope.")
				if stats.HasTestSent() {
					fmt.Println("Partial results for this build are available at:")
					fmt.Printf("   %s\n\n", a.getUrl(fmt.Sprintf("external/v1/results/%s", a.agentId)))
				}
				fmt.Printf("Check the agent logs at %s for more information.\n\n", a.recorderFilename)
			}
		}
	})
}

func (a *Agent) logMetadata() {
	metaBytes, _ := json.Marshal(a.metadata)
	strMetadata := string(metaBytes)
	a.logger.Println("Agent Metadata:", strMetadata)
}
