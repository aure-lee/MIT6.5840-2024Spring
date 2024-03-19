package raft

import "log"

// Debugging
const Debug = false

func DPrintf(format string, a ...interface{}) {
	if Debug {
		log.SetFlags(log.Ltime)
		log.Printf(format, a...)
	}
}
