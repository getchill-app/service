package main

import (
	"github.com/getchill-app/service"
)

// build flags passed from goreleaser
var (
	version = "0.0.0-dev"
	commit  = "snapshot"
	date    = ""
)

func main() {
	build := service.Build{
		Version:        version,
		Commit:         commit,
		Date:           date,
		ServiceName:    "chills",
		DefaultAppName: "Chill",
		DefaultPort:    49405,
	}
	service.Run(build)
}
