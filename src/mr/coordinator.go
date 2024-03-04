package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const TempDir = "./tmp"

type TaskType int // the type of task
type TaskStat int // the status of the task
type Phase int    // current phase of mapreduce

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
	MapPhase Phase = iota
	ReducePhase
	AllDone
)

type Task struct {
	TaskType   TaskType
	TaskStat   TaskStat
	InputFiles []string
	TaskID     int
	WorkerID   int
}

type Coordinator struct {
	// Your definitions here.
	inputFiles []string
	nMap       int
	nReduce    int
	mapTask    []Task
	reduceTask []Task
	phase      Phase
	mu         sync.Mutex
}

// Your code here -- RPC handlers for the worker to call.

// handle the RPC request: handle worker's request and reply
func (c *Coordinator) RequestTask(args *RequestTaskArgs, replys *RequestTaskReplys) error {
	if c.phase == MapPhase {
		// distribute the map task
		task := c.selectTask(c.phase)
		task.WorkerID = args.WorkerID
		replys.TaskType = task.TaskType
		replys.InputFiles = task.InputFiles
		replys.TaskID = task.TaskID
		replys.ReduceNum = c.nReduce

		// fmt.Println("RequestTask: selected task: ", *task)

		// after distributing the task, start timing, and after timeout the task is marked as NotAssign
		go c.checkTimeout(args.WorkerID, task)

	} else if c.phase == ReducePhase {
		// distribut the reduce task
		// TODO:
		task := c.selectTask(c.phase)
		task.WorkerID = args.WorkerID
		replys.TaskType = task.TaskType
		replys.InputFiles = task.InputFiles
		replys.TaskID = task.TaskID
		replys.ReduceNum = c.nReduce

		// fmt.Println("RequestTask: selected task: ", *task)

		go c.checkTimeout(args.WorkerID, task)
	} else {
		replys.TaskType = ExitTask
	}

	return nil
}

func (c *Coordinator) selectTask(phase Phase) *Task {
	c.mu.Lock()
	defer c.mu.Unlock()

	var task *Task
	if phase == MapPhase {
		for i := range c.mapTask {
			task = &c.mapTask[i]
			if task.TaskStat == NotAssign {
				task.TaskStat = Assigned
				return task
			}
		}
	} else if phase == ReducePhase {
		for i := range c.reduceTask {
			task = &c.reduceTask[i]
			if task.TaskStat == NotAssign {
				task.TaskStat = Assigned
				return task
			}
		}
	}

	return &Task{WaitTask, Finished, []string{}, -1, -1}
}

func (c *Coordinator) checkTimeout(workerId int, task *Task) {
	<-time.After(10 * time.Second)

	c.mu.Lock()
	defer c.mu.Unlock()

	if task.TaskStat == Finished {
		return
	}

	if (task.WorkerID == workerId && task.TaskStat == Assigned) || task.WorkerID != workerId {
		task.TaskStat = NotAssign
		task.WorkerID = -1
		// fmt.Printf("%v task %v do not finish.\n", task.TaskType, task.TaskID)
	}
}

// handle the RPC report: according to the report status, change the task status
func (c *Coordinator) ReportTask(args *ReportTaskArgs, replys *ReportTaskReplys) error {
	var task *Task

	if args.TaskType == MapTask {
		task = &c.mapTask[args.TaskId]
	} else if args.TaskType == ReduceTask {
		task = &c.reduceTask[args.TaskId]
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if task.WorkerID == args.WorkerID && task.TaskStat == Assigned {
		// fmt.Println("Task has finished.")
		task.TaskStat = Finished
		if task.TaskType == MapTask {
			c.nMap++
		} else if task.TaskType == ReduceTask {
			c.nReduce++
		}
	}

	return nil
}

func (c *Coordinator) makeMapTask() {
	for mapId := 0; mapId < c.nMap; mapId++ {
		inputFiles := make([]string, 0)
		inputFiles = append(inputFiles, c.inputFiles[mapId])

		task := &Task{MapTask, NotAssign, inputFiles, mapId, -1}
		c.mapTask = append(c.mapTask, *task)
	}

	c.nMap = 0 // nMap置0,标记有多少个task已完成
}

func (c *Coordinator) makeReduceTask() {
	// read all name like mr-X-Y files
	// all same Y files are in one group inputFiles

	for reduceId := 0; reduceId < c.nReduce; reduceId++ {
		// read all files in TmpPath
		pattern := fmt.Sprintf("%v/mr-*-%d", TempDir, reduceId)

		// 使用Glob找到所有匹配的文件
		files, err := filepath.Glob(pattern)
		if err != nil {
			fmt.Println("Error using Glob:", err)
			return
		}

		task := &Task{ReduceTask, NotAssign, files, reduceId, -1}
		c.reduceTask = append(c.reduceTask, *task)
	}

	c.nReduce = 0 // nReduce置0,记录有多少个task已完成
}

// Check the coordinator phase periodically
func (c *Coordinator) checkPhase() {

	for c.phase != AllDone {

		time.Sleep(time.Millisecond * 500)

		c.mu.Lock()

		if c.phase == MapPhase {
			if c.nMap == len(c.mapTask) {
				c.phase = ReducePhase
				c.makeReduceTask()
			}
		} else if c.phase == ReducePhase {
			if c.nReduce == len(c.reduceTask) {
				c.phase = AllDone
			}
		}

		c.mu.Unlock()
	}
}

// 可以使用系统自身的/tmp文件夹，自己创建文件夹时，需要注意文件夹的权限
func createTmpDir() {
	if _, err := os.Stat(TempDir); err == nil {
		// files, err := ioutil.ReadDir()
		files, err := os.ReadDir(TempDir)
		if err != nil {
			log.Fatalf("Failed to read directory: %s", err)
		}

		for _, file := range files {
			os.RemoveAll(filepath.Join(TempDir, file.Name()))
		}
	} else if os.IsNotExist(err) {
		os.Mkdir(TempDir, 0777)
	}

	os.Chmod(TempDir, 0777)
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

	// Your code here.
	return c.phase == AllDone
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// Your code here.
	c.inputFiles = files
	c.nMap = len(files)
	c.nReduce = nReduce
	c.phase = MapPhase
	createTmpDir()
	c.makeMapTask()

	go c.checkPhase()

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
