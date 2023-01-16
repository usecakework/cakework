package fly

import "fmt"

func DeployMachine(appName string, image string) error {
	fmt.Println("Deploying fly machine with appName " + appName)
	return nil
}