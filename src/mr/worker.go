package mr

import (
	"bufio"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"os"
	"sort"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

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
		case MapTask:
			doMap(mapf, reqReplys)
			rptArgs := ReportTaskArgs{reqReplys.TaskType, reqReplys.TaskID, os.Getpid()}
			reportTask(&rptArgs)
			// fmt.Printf("Map Task %v is finished.\n", reqReplys.TaskID)
		case ReduceTask:
			doReduce(reducef, reqReplys)
			rptArgs := ReportTaskArgs{reqReplys.TaskType, reqReplys.TaskID, os.Getpid()}
			reportTask(&rptArgs)
			// fmt.Printf("Reduce Task %v is finished.\n", reqReplys.TaskID)
		case WaitTask:
			time.Sleep(time.Second * 3)
			// fmt.Println("Waiting for task.")
		case ExitTask:
			// fmt.Printf("Worker %v exit.\n", os.Getpid())
			os.Exit(0)
		}
	}
}

func requestTask() (*RequestTaskReplys, bool) {
	args := RequestTaskArgs{os.Getpid()}
	replys := RequestTaskReplys{}
	ok := call("Coordinator.RequestTask", &args, &replys)

	if !ok {
		fmt.Println("Request Task error.")
	}

	return &replys, ok
}

func reportTask(args *ReportTaskArgs) bool {
	replys := ReportTaskReplys{}
	ok := call("Coordinator.ReportTask", &args, &replys)

	if !ok {
		fmt.Println("Report task error.")
	}

	return ok
}

func writeIntoTmpFiles(mapId, numFiles int, kva []KeyValue) {

	files := make([]*os.File, 0, numFiles)
	writers := make([]*bufio.Writer, 0, numFiles)
	encoders := make([]*json.Encoder, 0, numFiles)

	for i := 0; i < numFiles; i++ {
		filename := fmt.Sprintf("mr-%v-%v", mapId, i)
		file, err := os.CreateTemp(TempDir, filename)
		if err != nil {
			fmt.Println("Failed to create temp file:", err)
		}
		files = append(files, file)
		writers = append(writers, bufio.NewWriter(file))
		// 每个 json.Encoder 实例通过 json.NewEncoder(buf) 创建，并与一个 bufio.Writer 绑定
		encoders = append(encoders, json.NewEncoder(writers[i]))
	}

	// write map outputs to temp files
	for _, kv := range kva {
		idx := ihash(kv.Key) % numFiles
		err := encoders[idx].Encode(&kv)
		if err != nil {
			log.Fatalln("Cannot encode kv pair.")
		}
	}

	// flush file buffer into disk
	for _, w := range writers {
		err := w.Flush()
		if err != nil {
			log.Fatalln("Cannot flush file buffer")
		}
	}

	for i, file := range files {
		file.Close()
		newName := fmt.Sprintf("%v/mr-%v-%v", TempDir, mapId, i)
		err := os.Rename(file.Name(), newName)
		if err != nil {
			fmt.Printf("File %v rename error\n", newName)
		}
	}
}

func doMap(mapf func(string, string) []KeyValue, replys *RequestTaskReplys) error {
	intermediate := []KeyValue{}
	inputfiles := replys.InputFiles

	for _, filename := range inputfiles {
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("Cannot open %v", filename)
		}
		content, err := io.ReadAll(file)
		if err != nil {
			log.Fatalf("Cannot read %v", filename)
		}
		file.Close()

		kva := mapf(filename, string(content))

		intermediate = append(intermediate, kva...)
	}

	// sort
	sort.Sort(ByKey(intermediate))

	// write into files
	writeIntoTmpFiles(replys.TaskID, replys.ReduceNum, intermediate)

	return nil
}

func doReduce(reducef func(string, []string) string, replys *RequestTaskReplys) {
	fileId := replys.TaskID
	inputFiles := replys.InputFiles

	kva := map[string][]string{}

	for _, filename := range inputFiles {
		file, err := os.Open(filename)
		if err != nil {
			fmt.Printf("%v cannot open.\n", filename)
		}

		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			kva[kv.Key] = append(kva[kv.Key], kv.Value)
		}
	}

	keys := []string{}
	for key := range kva {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	tempName := fmt.Sprintf("mr-out-%v", fileId)
	tempFile, _ := os.CreateTemp(TempDir, tempName)

	for _, key := range keys {
		output := reducef(key, kva[key])

		fmt.Fprintf(tempFile, "%v %v\n", key, output)
	}

	tempFile.Close()
	newName := fmt.Sprintf("./mr-out-%v", fileId)
	err := os.Rename(tempFile.Name(), newName)

	if err != nil {
		fmt.Printf("Reduce file %v rename error.\n", fileId)
	}
}

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
