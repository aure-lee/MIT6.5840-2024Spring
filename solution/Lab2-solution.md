# Lab 2: Key/Value Server

呃呃，Lab2做到Lab4上了，花一天重做一遍，我说我怎么跑不过测试 :sweat_smile:

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
		return
	}
}
```

```go
func (kv *KVServer) Append(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()

	// return old value
	reply.Value = kv.data[args.Key]
	// append map[key]
	kv.data[args.Key] = kv.data[args.Key] + args.Value
}
```

Client端实现：

```go
func (ck *Clerk) Get(key string) string {
	// You will have to modify this function.

    args := GetArgs{key}
    reply = GetReply{}
    ok := ck.server.Call("KVServer.Get", &args, &reply)

	return reply.Value
}
```

```go
func (ck *Clerk) PutAppend(key string, value string, op string) string {
	// You will have to modify this function.

	args := PutAppendArgs{key, value}
	reply := PutAppendReply{}
	ok := ck.server.Call("KVServer."+op, &args, &reply)

	return reply.Value
}
```

## 2 实现有丢包的Test

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
// 先保存完整的dupTable，不考虑如何减小dupTable的规模
// PutReply 保存 ""
// AppendReply/GetReply 保存 应该返回的string
// 优化方向：duplicateTable只保存一个客户端最近的Reply，因为对某个特定的客户端来说，
// 他的请求一定是顺序发送的，某个请求之前的回复客户端都已经收到了
// 因此，dupTable只需要存储client[i]最近完成的Reply即可
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
		reply.Value = v
		return
	}

	delete(dt, args.Seq-1)
	reply.Value = kv.data[args.Key]
	dt[args.Seq] = reply.Value
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
		ok = ck.server.Call("KVServer."+op, &args, &reply)
	}
```

此时运行`go test`，会提示使用了太多空间，这里有一个Go语言的特性，直接使用 `delete` 不会释放 `map` 里的空间，重新写一个数据结构保存最近的条目。

## 3 优化存储空间

此部分Client不需要更改，只需要专注 Server 即可。

`dupTable` 修改如下：

```go
type dupTable struct {
	seq   int
	value string
}
```
Server 的 Get函数修改如下：为了更好的节省空间，当一个客户端没有发送过数据时，不需要创建 `dupTable` 表，直接返回当前 Server的 `map[key]`。

```go
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
```
