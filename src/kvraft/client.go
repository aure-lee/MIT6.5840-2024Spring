package kvraft

import (
	"crypto/rand"
	"math/big"

	"6.5840/labrpc"
)

type Clerk struct {
	servers []*labrpc.ClientEnd
	// You will have to modify this struct.
	clientId int64
	seq      int64 // 序列号：不重复的RPC发送不重复的seq，重复的RPC发送同一个seq
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.servers = servers

	// You'll have to add code here.
	ck.clientId = nrand()
	ck.seq = 0

	return ck
}

// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) Get(key string) string {
	// You will have to modify this function.

	var reply GetReply
	ok := false

	ck.seq++

	for !ok {
		args := GetArgs{key, ck.clientId, ck.seq}
		reply = GetReply{}
		ok = ck.servers[0].Call("KVServer.Get", &args, &reply)
	}

	return reply.Value
}

// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) PutAppend(key string, value string, op string) {
	// You will have to modify this function.

	ck.seq++

	// Client need to send message to every server
	for i := 0; i < len(ck.servers); i++ {

		ok := false
		for !ok {
			args := PutAppendArgs{key, value, op, ck.clientId, ck.seq}
			reply := PutAppendReply{}
			ok = ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
		}
	}

	// if reply.Err != OK {
	// 	log.Fatal(reply.Err)
	// }
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
