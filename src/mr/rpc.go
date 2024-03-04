package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

// Worker requests tasks
// 这个 WorkerId 在这个实验中可以省略
type RequestTaskArgs struct {
	WorkerID int
}

type RequestTaskReplys struct {
	TaskType   TaskType
	TaskID     int
	InputFiles []string
	ReduceNum  int
}

// Worker reports tasks
type ReportTaskArgs struct {
	TaskType TaskType
	TaskId   int
	WorkerID int
}

type ReportTaskReplys struct {
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
