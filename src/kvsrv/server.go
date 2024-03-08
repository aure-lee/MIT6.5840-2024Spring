package kvsrv

import (
	"log"
	"sync"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

// 先保存完整的dupTable，不考虑如何减小dupTable的规模
// PutReply 保存 ""
// AppendReply/GetReply 保存 应该返回的string
// 优化方向：duplicateTable只保存一个客户端最近的Reply，因为对某个特定的客户端来说，
// 他的请求一定是顺序发送的，某个请求之前的回复客户端都已经收到了
// 因此，dupTable只需要存储client[i]最近完成的Reply即可
type dupTable struct {
	seq   int
	value string
}

type KVServer struct {
	mu sync.Mutex

	// Your definitions here.
	data        map[string]string
	clientTable map[int64]*dupTable // 根据每个ClientId映射到对应的duplicateTable
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.

	kv.mu.Lock()
	defer kv.mu.Unlock()

	// duplicate detection
	// if client is new, create the map of clientid -> duplicateTable
	if kv.clientTable[args.ClientId] == nil {
		// 当这个client没有写入操作时，它读取的数据永远是当前的服务器Value
		reply.Value = kv.data[args.Key]
		return
	}

	// 这里有个坑，注意指针引用
	dt := kv.clientTable[args.ClientId]

	// if seq exists
	if dt.seq == args.Seq {
		reply.Value = dt.value
		return
	}

	dt.seq = args.Seq
	dt.value = kv.data[args.Key]
	reply.Value = dt.value
	DPrintf("Server: ClientId %v Get reply %v and dt.seq is %v, value is %v\n",
		args.ClientId, args.Seq, dt.seq, dt.value)
}

func (kv *KVServer) Put(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.

	kv.mu.Lock()
	defer kv.mu.Unlock()

	// duplicate detection
	// if client is new, create the map of clientid -> duplicateTable
	if kv.clientTable[args.ClientId] == nil {
		kv.clientTable[args.ClientId] = &dupTable{-1, ""}
	}

	dt := kv.clientTable[args.ClientId]

	// if seq exists
	if dt.seq == args.Seq {
		return
	}

	dt.seq = args.Seq
	dt.value = ""
	kv.data[args.Key] = args.Value
}

func (kv *KVServer) Append(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.

	kv.mu.Lock()
	defer kv.mu.Unlock()

	// duplicate detection
	// if client is new, create the map of clientid -> duplicateTable
	if kv.clientTable[args.ClientId] == nil {
		kv.clientTable[args.ClientId] = &dupTable{-1, ""}
	}

	dt := kv.clientTable[args.ClientId]
	DPrintf("Server: ClientId %v Append reply %v and dt.seq is %v\n",
		args.ClientId, args.Seq, dt.seq)

	// if seq exists
	if dt.seq == args.Seq {
		reply.Value = dt.value
		return
	}

	// return old value
	dt.seq = args.Seq
	dt.value = kv.data[args.Key]
	reply.Value = dt.value
	// append map[key]
	kv.data[args.Key] = kv.data[args.Key] + args.Value
	DPrintf("Server: ClientId %v Append reply %v and dt.seq is %v, value is %v\n",
		args.ClientId, args.Seq, dt.seq, dt.value)
}

func StartKVServer() *KVServer {
	kv := new(KVServer)

	// You may need initialization code here.
	kv.data = map[string]string{}
	kv.clientTable = map[int64]*dupTable{}

	return kv
}
