package main

// version / buildStamp are set by Makefile with -ldflags "-X main.version=… -X main.buildStamp=…"
var (
	version    = "v1.0.10"
	buildStamp = "dev"
)
