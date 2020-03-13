package handlers

import "github.com/kylelemons/godebug/pretty"

var prettyConfig = &pretty.Config{
	TrackCycles: true,
}
var prettyConfigIgnoreUnexported = &pretty.Config{
	IncludeUnexported: false,
	TrackCycles:       true,
}
