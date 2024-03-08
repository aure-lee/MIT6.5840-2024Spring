package kvsrv

import (
	"crypto/rand"
	"math/big"

	"6.5840/labrpc"
)

type Clerk struct {
	server *labrpc.ClientEnd

	// You will have to modify this struct.
	clientId int64
	seq      int // 序列号：不重复的RPC发送不重复的seq，重复的RPC发送同一个seq
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(server *labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.server = server

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
// ok := ck.server.Call("KVServer.Get", &args, &reply)
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
		ok = ck.server.Call("KVServer.Get", &args, &reply)

		// DPrintf("Client: ClientId %v Get args %v send\n",
		// 	ck.clientId, ck.seq)
	}

	// DPrintf("Client ClientId receive %v Get reply %v\n",
	// 	ck.clientId, reply.Value)
	return reply.Value
}

// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.server.Call("KVServer."+op, &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) PutAppend(key string, value string, op string) string {
	// You will have to modify this function.
	var reply PutAppendReply
	ck.seq++

	ok := false
	for !ok {
		args := PutAppendArgs{key, value, ck.clientId, ck.seq}
		reply = PutAppendReply{}

		// DPrintf("Client: ClientId %v %v args %v send, value is %v\n",
		// 	ck.clientId, op, ck.seq, value)

		ok = ck.server.Call("KVServer."+op, &args, &reply)
	}

	// DPrintf("Client: ClientId %v receive %v reply %v\n",
	// 	ck.clientId, op, reply.Value)

	return reply.Value
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}

// Append value to key's value and return that value
func (ck *Clerk) Append(key string, value string) string {
	return ck.PutAppend(key, value, "Append")
}
