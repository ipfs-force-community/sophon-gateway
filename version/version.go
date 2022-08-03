package version

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "v1.6.1"

var UserVersion = BuildVersion + CurrentCommit
