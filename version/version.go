package version

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "1.8.0"

var UserVersion = BuildVersion + CurrentCommit
