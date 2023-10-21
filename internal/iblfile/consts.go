package iblfile

const Protocol = "frostpaw-rev5-e1" // The exact protocol version to use

var FormatVersionMap = map[string]string{}

// The number of keys to encrypt the data with
//
// Note that changing keycount does not need a change in protocol version
const KeyCount = 16
