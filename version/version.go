package version

var CurrentCommit string

// BuildVersion is the local build version, set by build system
const BuildVersion = "v1.8.0-rc2"

var UserVersion = BuildVersion + CurrentCommit
