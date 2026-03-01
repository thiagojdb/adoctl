package main

import (
	"adoctl/cmd"
)

var (
	Version   string
	BuildTime string
	GitCommit string
)

func main() {
	cmd.Version = Version
	cmd.BuildTime = BuildTime
	cmd.GitCommit = GitCommit

	cmd.Execute()
}
