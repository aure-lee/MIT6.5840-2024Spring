package kvraft

import (
	"log"
	"sync"
	"sync/atomic"

	"6.5840/labgob"
	"6.5840/labrpc"
	"6.5840/raft"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
}

// 先保存完整的dupTable，不考虑如何减小dupTable的规模
// PutAppendArgs 保存 "Put" / "Append"
// GetArgs 保存 应该返回的string
// 优化方向：duplicateTable只保存未完成的RPC，而不是已完成的RPC
type dupTable map[int64]string

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	maxraftstate int // snapshot if log grows this big

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
		kv.clientTable[args.ClientId] = &dupTable{}
	}

	dt := *kv.clientTable[args.ClientId]

	// if seq exist
	if v, exists := dt[args.Seq]; exists {
		reply.Value = v
		reply.Err = OK
		return
	}

	reply.Value = kv.data[args.Key]
	dt[args.Seq] = reply.Value

	// key do not exist
	if _, exists := kv.data[args.Key]; !exists {
		reply.Err = ErrNoKey
		return
	}

	reply.Err = OK
}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.

	kv.mu.Lock()
	defer kv.mu.Unlock()

	// duplicate detection
	// if client is new, create the map of clientid -> duplicateTable
	if kv.clientTable[args.ClientId] == nil {
		kv.clientTable[args.ClientId] = &dupTable{}
	}

	dt := *kv.clientTable[args.ClientId]

	// if seq exist
	if _, exists := dt[args.Seq]; exists {
		reply.Err = OK
		return
	}

	if args.Op == "Put" {

		kv.data[args.Key] = args.Value
		dt[args.Seq] = "Put"

	} else if args.Op == "Append" {

		kv.data[args.Key] = kv.data[args.Key] + args.Value
		dt[args.Seq] = "Append"
	}

	reply.Err = OK
}

// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
func (kv *KVServer) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
//
// servers[] 包含一组服务器的端口，它们将通过 Raft 协作形成容错的键值服务。
// me 是 servers[] 中当前服务器的索引。
// 键/值服务器应通过底层的 Raft 实现存储快照，该实现应调用 persister.SaveStateAndSnapshot()
// 来原子性地保存 Raft 状态和快照。
// 当 Raft 的保存状态超过 maxraftstate 字节时，键/值服务器应进行快照，
// 以便允许 Raft 对其日志进行垃圾回收。如果 maxraftstate 为 -1，则不需要进行快照。
// StartKVServer() 必须快速返回，因此它应为任何长时间运行的工作启动 goroutine。
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	// 使这个结构体可以被Go的RPC库用于数据的序列化(marshalling)和反序列化(unmarshalling)
	labgob.Register(Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate

	// You may need initialization code here.

	kv.data = map[string]string{}
	kv.clientTable = map[int64]*dupTable{}

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	// You may need initialization code here.

	return kv
}
