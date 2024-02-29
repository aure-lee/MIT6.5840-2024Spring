package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"path/filepath"
	"time"
)

const TmpPath = "/var/tmp/"

type Coordinator struct {
	// Your definitions here.
	inputFiles []string
	nMap       int
	nReduce    int
	mapTask    TaskQueue
	reduceTask TaskQueue
	cond       Condition
}

type TaskType int  // the type of task
type TaskStat int  // the status of the task
type Condition int // current phase of mapreduce

const (
	MapTask TaskType = iota // iota = 0
	ReduceTask
	WaitTask
	ExitTask
)

const (
	NotAssign TaskStat = iota // 未分配
	Assigned                  // 已分配，未完成
	Finished                  // 已完成
)

const (
	MapPhase Condition = iota
	ReducePhase
	AllDone
)

type Task struct {
	TaskType   TaskType
	TaskStat   TaskStat
	InputFiles []string
	StartTime  time.Time
	TaskID     int
	WorkerID   int
}

type TaskQueue struct {
	AllTasks chan *Task
	TaskNum  int
}

// Your code here -- RPC handlers for the worker to call.

func logTask(task *Task) {
	// open file, if file doesn't exist, create
	file, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// create a new logger
	logger := log.New(file, "LOG: ", log.LstdFlags)
	logger.Printf("Task: %+v\n", task)
}

func (c *Coordinator) DistTask(args *RequestTaskArgs, replys *RequestTaskReplys) error {

	if c.cond == MapPhase {
		// the list of map tasks has task
		c.distMapTask()

		// the list doesn't have task
		replys.TaskType = WaitTask
	} else if c.cond == ReducePhase {
		// the list of reduce tasks has task
		c.distReduceTask()

		// the list doesn't have task
		replys.TaskType = WaitTask
	} else {
		replys.TaskType = ExitTask
	}

	return nil
}

func (c *Coordinator) distMapTask() {
	if c.mapTask.TaskNum > 0 {

	}
}

func (c *Coordinator) distReduceTask() {}

func (c *Coordinator) makeMapTask() {
	for mapId := 0; mapId < c.nMap; mapId++ {
		inputFiles := make([]string, 0)
		inputFiles = append(inputFiles, c.inputFiles[mapId])

		task := &Task{MapTask, NotAssign, inputFiles, time.Time{}, mapId, -1}

		c.mapTask.AllTasks <- task
		c.mapTask.TaskNum++
	}
}

func (c *Coordinator) makeReduceTask() {
	// read all name like mr-tmp-X-Y files
	// all same Y files are in one group inputFiles

	for reduceId := 0; reduceId < c.nReduce; reduceId++ {
		// read all files in TmpPath
		pattern := fmt.Sprintf("%v/mr-tmp-*-%d", TmpPath, reduceId)

		// 使用Glob找到所有匹配的文件
		files, err := filepath.Glob(pattern)
		if err != nil {
			fmt.Println("Error using Glob:", err)
			return
		}

		task := &Task{ReduceTask, NotAssign, files, time.Time{}, reduceId, -1}

		logTask(task)

		go func() {
			c.reduceTask.AllTasks <- task
			c.reduceTask.TaskNum++
		}()

	}
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname) // unix域套接字，监听同一台机器上的进程间通信
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false

	// Your code here.

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// Your code here.
	TmpPath = "/var/tmp/"

	c.inputFiles = files
	c.nMap = len(files)
	c.nReduce = nReduce
	c.makeMapTask()

	c.server()
	return &c
}

// Example an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}
