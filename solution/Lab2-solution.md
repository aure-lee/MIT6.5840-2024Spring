# Lab 2: Key/Value Server

## 1 实现One Client和Many Clients - Maybe solve

Server的 KV 添加一个字段：`data map[string]string`，用来存储对应的 `KV pair`。

server 中实现 `PutAppend` 和 `Get` 的 `Reply` 逻辑：

```go
func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.

	kv.mu.Lock()
	defer kv.mu.Unlock()

	reply.Value = kv.data[args.Key]

	// key do not exist
	if _, exists := kv.data[args.Key]; !exists {
		reply.Err = ErrNoKey
		return
	}

	reply.Err = OK
}
```

```go
func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.

	kv.mu.Lock()
	defer kv.mu.Unlock()

	if args.Op == "Put" {

		kv.data[args.Key] = args.Value

	} else if args.Op == "Append" {
        // 查看test_test.go，得知是直接字符串拼接，不是转换成Int型相加
		kv.data[args.Key] = kv.data[args.Key] + args.Value
	}

	reply.Err = OK
}
```

Client端实现：

```go
func (ck *Clerk) Get(key string) string {
	// You will have to modify this function.

    args := GetArgs{key}
    reply = GetReply{}
    ok := ck.servers[0].Call("KVServer.Get", &args, &reply)

	return reply.Value
}
```

```go
func (ck *Clerk) PutAppend(key string, value string, op string) {
	// You will have to modify this function.

	// Client need to send message to every server(this is maybe wrong)
	for i := 0; i < len(ck.servers); i++ {

        args := PutAppendArgs{key, value, op}
        reply := PutAppendReply{}
        ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
	}

    // if reply.Err != OK {
	// 	log.Fatal(reply.Err)
	// }
}
```

## 2 实现有丢包的Test - Maybe solve

丢包有两种：1. `args` 包被丢弃。2. `replys` 包被丢弃。

1. `args` 包被丢弃比较好处理，再次把args发送一次即可。
2. `replys` 包被丢弃：此时服务器已经处理完了 `request`，但是如果简单的重复发送 `args`，将会导致服务器重复操作，引起糟糕的后果。

### hint

您需要独特地标识客户端操作，以确保键/值服务器每个操作只执行一次。

您需要仔细考虑服务器为处理重复的 `Get()`、`Put()` 和 `Append()` 请求而必须保持的状态（如果有的话）。

您的重复检测方案应该能够快速释放服务器内存，例如，通过使每个RPC隐含客户端已经看到了其上一个RPC的回复。可以假设客户端一次只会对一个Clerk进行一次调用。

### 实现

查看duplicate detection页面，提供了一些思路：
在服务器中维护一个重复表，如果发来的请求是已经处理过的（即在重复表中），则直接返回内容（如果有的话）。

同时为了避免客户端之间的冲突，对每个客户端生成一个单独的重复表。

```go
type dupTable map[int64]string

type KVServer struct {
    // ...
	clientTable map[int64]*dupTable // 根据每个ClientId映射到对应的duplicateTable
}
```

修改 `common.go`，使 `Args` 能传输 `ClientId` 和 `Seq`。

Server端需要加上关于 `duplicateTable` 的处理逻辑：

```go
    // ...
	// duplicate detection
	// if client is new, create the map of clientid -> duplicateTable
	if kv.clientTable[args.ClientId] == nil {
		kv.clientTable[args.ClientId] = &dupTable{}
	}

	dt := *kv.clientTable[args.ClientId]

	// if seq exist
	if v, exists := dt[args.Seq]; exists {
        reply.Value = v // if put
		reply.Err = OK
		return
	}
    // ...
    // 未全部给出
```

同时，client也需要生成独特的 `ClientId` 和 `Seq`，处理发送的逻辑，如果有丢包则重复发送直到得到 `reply`。

```go
// Example Get
	var reply GetReply
	ok := false

	ck.seq++

	for !ok {
		args := GetArgs{key, ck.clientId, ck.seq}
		reply = GetReply{}
		ok = ck.servers[0].Call("KVServer.Get", &args, &reply)
	}
```

`go test -race` 的结果如下：

```text
lee@lee-virtual-machine:~/6.5840/src/kvraft$ go test -race
Test: one client (4A) ...
  ... Passed --  15.2  5 29379 9791
Test: ops complete fast enough (4A) ...
  ... Passed --   1.5  3  3002    0
Test: many clients (4A) ...
  ... Passed --  16.0  5 121037 40117
Test: unreliable net, many clients (4A) ...
  ... Passed --  15.4  5  5346 1430
Test: concurrent append to same key, unreliable (4A) ...
  ... Passed --   0.6  3   178   52
Test: progress in majority (4A) ...
```

`Test: progress in majority (4A) ...`，这个测试涉及到了leader的选举，还未完成。
