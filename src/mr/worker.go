package mr

import (
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
	"os"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
// 使用 ihash(key) % NReduce 来选择 Map 发出的每个 KeyValue 的 reduce 任务编号。
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.
	for {
		reqReplys, ok := requestTask()
		if !ok {
			fmt.Println("Request task error!")
			break
		}

		switch reqReplys.TaskType {
		case WaitTask:
			{
				time.Sleep(time.Second)
				fmt.Println("Wait for task.")
			}
		case ExitTask:
			{
				fmt.Printf("Worker %v exit.", os.Getpid())
				os.Exit(0)
			}
		case MapTask:
			{

			}
		case ReduceTask:
			{

			}
		}
	}
}

func requestTask() (*RequestTaskReplys, bool) {
	args := RequestTaskArgs{os.Getpid()}
	replys := RequestTaskReplys{}
	ok := call("Coordinator.DistTask", &args, &replys)

	return &replys, ok
}

func doMap(mapf func(string, string) []KeyValue, replys *RequestTaskReplys) {}

func doReduce(reducef func(string, []string) string, replys *RequestTaskReplys) {}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
