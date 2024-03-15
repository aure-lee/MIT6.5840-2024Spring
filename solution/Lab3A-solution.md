# Lab3A

Raft 协议运行的基础？分布式网络中有大部分可以运行的服务器？为什么？由协议决定的吗？

Raft 协议要求大多数节点参与决策过程，以确保系统的一致性。这意味着在任何给定时刻，必须至少有半数以上的节点是可访问的和正常运行的，才能进行日志条目的复制和提交。因此，为了在可能的网络分区或节点故障情况下保持系统的可用性和一致性，Raft 需要有奇数个服务器节点。

## Hints

1. 遵循论文中的图 2。在这一点上，你需要关注发送和接收 `RequestVote RPCs`，与 **选举相关的服务器规则**，以及与 **领导者选举相关的状态**。
2. 在 `raft.go` 中的 `Raft` 结构体中 **添加** 图 2 中 **用于领导者选举的状态**。你还需要 **定义一个结构体** 来 **保存每个日志条目的信息**。
3. 填写 `RequestVoteArgs` 和 `RequestVoteReply` 结构。修改 `Make()`，创建一个后台 `goroutine`，该程序会在一段时间内没有收到另一个对等服务器的消息时，通过发送 `RequestVote` RPC 来定期启动领导者选举。实现 `RequestVote()` RPC 处理程序，以便服务器能相互投票。
4. 要实现心跳，请定义一个 `AppendEntries` RPC 结构（尽管可能还不需要所有参数），并让领导者定期发送心跳。编写一个 `AppendEntries` RPC 处理程序方法。
5. 测试程序要求领导者每秒发送心跳 RPC 的次数不超过十次。  
测试程序要求在旧领导者失败后的五秒内（如果大多数对等体仍然可以通信），你的 Raft 必须选举出一个新的领导者。
6. 论文的第5.2节提到选举超时范围为150至300毫秒。这样的范围只有在领导者发送心跳远远超过每150毫秒一次（例如，每10毫秒一次）时才有意义。由于测试程序限制了你每秒钟发送的心跳数量，你将不得不使用比论文中的150至300毫秒更长的选举超时时间，但不能太长，否则你可能无法在五秒内选举出一个领导者。
7. Go的 `rand` 包可能会有用。
8. 你需要编写定期或延迟一段时间后执行操作的代码。最简单的方法是创建一个带有循环的 `goroutine`，调用 `time.Sleep()`；参见 `Make()` 创建的 `ticker() goroutine`，专门用于此目的。不要使用 Go 的 `time.Timer` 或 `time.Ticker`，它们难以正确使用。
9. 如果你的代码在通过测试时遇到问题，请再次阅读论文中的图 2；领导者选举的全部逻辑分散在图的多个部分中。
10. 不要忘记实现 GetState()。
11. 测试程序在永久关闭一个实例时会调用你的 Raft 的 `rf.Kill()`。你可以使用 `rf.killed()` 来检查是否调用了 `Kill()`。你可能希望在所有循环中都这样做，以避免已停止的 Raft 实例打印混乱的消息。
12. Go RPC 只发送以大写字母开头的字段名的结构字段。子结构体也必须具有大写字母开头的字段名（例如，数组中的日志记录的字段）。labgob 包会对此发出警告；不要忽略这些警告。
13. 这个实验中最具挑战性的部分可能是调试。花一些时间使你的实现易于调试。

[Raft论文中文版](https://github.com/maemual/raft-zh_cn/blob/master/raft-zh_cn.md)

## Rules for mutex

1. 每当有多个 goroutine 使用数据，并且至少有一个 goroutine 可能修改数据时，应该使用锁来防止数据的同时使用。Go 语言的竞态检测器在检测到这种规则的违反时表现得相当好（尽管它对下面的任何规则都不起作用）。
2. 每当代码对共享数据进行一系列修改，并且其他 goroutine 如果在序列的中间位置查看数据可能会出现故障时，应该在整个序列周围使用锁。
3. 每当代码对共享数据进行一系列读取（或读取和写入），并且如果另一个 goroutine 在序列的中间修改数据可能会导致故障时，应该在整个序列周围使用锁。
4. 通常不应该在执行可能会等待的操作时持有锁：读取 Go 通道、在通道上发送、等待定时器、调用 time.Sleep()，或发送 RPC（并等待回复）。一个原因是你可能希望其他 goroutine 在等待期间能够继续执行。另一个原因是避免死锁。想象两个对等节点在持有锁的情况下互相发送 RPC；两个 RPC 处理程序都需要接收对等节点的锁；由于它们需要等待 RPC 调用持有的锁，因此两个 RPC 处理程序都无法完成。
5. 要谨慎处理在释放锁后和重新获取锁之间的假设。一个可能出现这种情况的地方是在持有锁时避免等待。

## Structure Advice

1. 对于 Raft，使用共享数据和锁是最直接的方法。
2. 一个 Raft 实例有两种基于时间的活动：领导者必须发送心跳，其他实例在听不到领导者的消息后必须启动选举。最好是用专门的长时间运行的 goroutine 来驱动这些活动，而不是将多个活动合并到单个 goroutine 中。
3. 管理选举超时是常见的头痛之源。也许最简单的方法是在 Raft 结构体中维护一个变量，其中包含对等节点最后一次听到领导者的时间，并且让选举超时的 goroutine 定期检查是否距离上次超时的时间大于超时期限。
4. 你会希望有一个单独的长时间运行的 goroutine，在 applyCh 上按顺序发送已提交的日志条目。它必须是单独的，因为在 applyCh 上发送可能会阻塞；而且它必须是一个单独的 goroutine，否则可能很难确保按日志顺序发送日志条目。推进 commitIndex 的代码将需要启动应用 goroutine；最简单的方法可能是使用条件变量（Go 的 sync.Cond）。
5. 每个 RPC 可能最好在自己的 goroutine 中发送（并处理其回复），原因有两个：一是为了避免无法到达的对等节点延迟了大多数回复的收集时间，二是为了确保心跳和选举定时器能够持续不断地工作。最简单的做法是在同一个 goroutine 中处理 RPC 回复，而不是通过通道发送回复信息。
6. 请记住，网络可能会延迟 RPC 和 RPC 回复，并且当你发送并发的 RPC 时，网络可能会重新排序请求和回复。图 2 在指出 RPC 处理程序需要注意这一点时做得相当好（例如，RPC 处理程序应该忽略带有旧任期的 RPC）。但图 2 并不总是明确提及 RPC 回复处理。领导者在处理回复时必须小心；它必须检查自从发送 RPC 以来任期是否发生了变化，并且必须考虑到同一追随者的并发 RPC 的回复可能已经改变了领导者的状态（例如，nextIndex）。

## 选主

需要把图 2 仔细看一遍，同时完整阅读第五章的内容，有一些比较细微的边界条件在第五章的论文部分有详细说明，例如：何为最新的日志。

为了选出 `Leader`，需要实现的是 `RequestVote` 和 `election timeout`，以及当选成功后的发送心跳包部分。

`Raft` 的定义如下所示：

```go
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Persistent state on all servers
	currentTerm int
	votedFor    int
	logs        []LogEntry

	// Volatile state on all servers
	commitIndex int
	lastApplied int

	// Volatile state on leaders
	nextIndex  []int
	matchIndex []int

	role       Role
	heartChan  chan struct{}
	lastUpdate time.Time
}
```

`heartChan` 用来控制发送心跳包，`lastUpdate` 用来记录上次更新的时间，检测选举超时。

`RequestVote` 和 `AppendEntries` 的数据结构如图2所示。

`RequestVote` 的 `handle` 函数如下：

```go
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (3A, 3B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// 1. term < currentTerm: reply false
	// 2. term == currentTerm and have voted: reply false
	// 3. term > currentTerm: convert to follower, reset rf
	// 4. term == currentTerm and not vote: compare log
	// 5. compare lastLogIndex and lastLogTerm

	if args.Term < rf.currentTerm {
		reply.Term, reply.VoteGranted = rf.currentTerm, false
		return
	} else if args.Term == rf.currentTerm && rf.votedFor != -1 && rf.votedFor != args.CandidateId {
		reply.Term, reply.VoteGranted = rf.currentTerm, false
		return
	}

	if args.Term > rf.currentTerm {
		if rf.role == Leader {
			rf.stepDown()
		}
		rf.role = Follower // TODO: 把 role 互相转变的过程封装起来，convert默认在加锁情况下转变
		rf.votedFor = -1
	}

	// 如果符合条件的candidate的日志不是最新的，不投票给他，同时重置votedFor
	if !rf.isLogUpToDate(args.LastLogTerm, args.LastLogIndex) {
		reply.Term, reply.VoteGranted = rf.currentTerm, false
        rf.votedFor = -1
		return
	}

	reply.Term, reply.VoteGranted = rf.currentTerm, true

	rf.votedFor = args.CandidateId
	rf.lastUpdate = time.Now()
	DPrintf("{Server %v} Term %v Vote for Server %v\n", rf.me, rf.currentTerm, rf.votedFor)
}
```

由提示可知，要求我们一秒钟之内收到的心跳包不超过10个，同时选举超时时间大于心跳包频率一个数量级，修改 `ticker` 内的超时时间，选举超时时间使用 `time.Sleep()` 实现（我才不会说我还没搞明白 `time.Timer`）：

```go
func (rf *Raft) ticker() {
	for rf.killed() == false {
        ...
		rf.mu.Lock()

		// elecion timeout elapses: start new elections
		if !isLeader && time.Since(rf.lastUpdate) > 500*time.Millisecond {
			rf.startElection()
		}
		rf.mu.Unlock()

		// pause for a random amount of time between 50 and 350
		// milliseconds.
		// ms := 50 + (rand.Int63() % 300)
		ms := 600 + (rand.Int63() % 300)
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}
```

启动选举的函数，使用闭包来计算获得票数，注意这几个函数之间的 `mutex` 关系，不会出现死锁。


```go
func (rf *Raft) startElection() {
    ...
	// send RequestVote RPCs to all other servers
	voteCount := 1

	for i := range rf.peers {
		if i != rf.me {
			// 将i作为闭包传递给goroutine，因为直接在goroutine中调用i，可能会调用已经修改过的i
			DPrintf("{Server %v} Term %v Send RequestVoteArgs to server %v\n",
				rf.me, rf.currentTerm, i)
			go func(server int) {
				reply := RequestVoteReply{}
				if rf.sendRequestVote(server, &args, &reply) {
					rf.mu.Lock()
					defer rf.mu.Unlock()
					if rf.role == Candidate && reply.VoteGranted {
						voteCount++
						// if votes received from majority of servers: become leader
						if voteCount > len(rf.peers)/2 {
							rf.role = Leader
							DPrintf("{Server %v} Term %v Success become leader.\n",
								rf.me, rf.currentTerm)
							rf.becomeLeader()
							return
						}
					} else if reply.Term > rf.currentTerm {
						DPrintf("{Server %v} Old Term %v discovers a new term %v, convert to follower\n",
							rf.me, rf.currentTerm, reply.Term)
						rf.currentTerm = reply.Term
						rf.role = Follower // TODO
					}
				}

			}(i)
		}
	}
}
```

接下来是 `startHeartbeat()`:

```go
func (rf *Raft) startHeartbeat() {
	// 每秒钟发送8次，测试程序要求leader每秒发送的心跳包次数不超过10次
	interval := time.Second / 8

	go func() {
		for {
			select {
			case <-rf.heartChan:
				return
			default:
				term, isLeader := rf.GetState()

				// 当服务器是Leader时，发送心跳包
				if isLeader {
					args := AppendEntriesArgs{Term: term, LeaderId: rf.me}

					// 并行向所有Follower send AppendEntries
					for i := range rf.peers {
						if i != rf.me {
							go func(server int) {
								reply := AppendEntriesReply{}
								rf.sendAppendEntries(server, &args, &reply)
							}(i)
						}
					}
				} else {
					return
				}
				// wait for next send AppendEntries
				time.Sleep(interval)
			}
		}
	}()
}
```

3A 部分的心跳包处理比较简单，暂不列出。
