# Lab3B - solution

## Hints

1. 你的第一个目标是通过实现 `TestBasicAgree3B()` 来通过测试。首先实现 `Start()` 方法，然后编写代码通过 AppendEntries RPCs 发送和接收新的日志条目，遵循图2的指导。在每个对等体上将每个新提交的条目发送到 `applyCh` 上。
2. 你需要实现选举限制（论文中的第5.4.1节）。
3. 你的代码可能会有循环，重复检查某些事件。不要让这些循环无休止地执行，因为这样会使你的实现变慢，导致测试失败。使用 Go 的条件变量，或在每次循环迭代中插入 `time.Sleep(10 * time.Millisecond)`。
4. 如果测试失败，请查看 `test_test.go` 和 `config.go`，了解正在测试什么。`config.go` 还展示了测试器如何使用 Raft API。

## Solution

这个部分主要需要修改的是 `AppendEntriesArgs` 部分。

对于 `Follower` 处理 `Leader` 发来的 `AppendEntriesArgs`，在之前的逻辑后加上 **如果有附加日志** 的判断：
1. 如果 `PrevLogIndex` 和 `PrevLogTerm` 不匹配，则返回 `false`，此时 `Leader` 收到相同任期的 `false reply`，会有相应的处理方式。
2. 如果 `PrevLogIndex` 和 `PrevLogTerm` 匹配，（注：在我的实现中，`PrevLogIndex` 可能会是 $-1$ ，此时认为匹配，把Follower所有的已知条目都覆盖）把 `PrevLogIndex` 之后的条目都覆盖，同时根据 `LeaderCommit` 和 `commitIndex` 比较，应用符合的条目的状态机中。

对于 `Leader` 先要生成 `AppendEntriesArgs`，之后要处理 `AppendEntriesReply`：
1. 生成 `AppendEntriesArgs`：由于 `PrevLogIndex` 可能为 $-1$，此时的 `PrevLogTerm` 设为 $0$。
2. 处理 `AppendEntriesReply`：
    - 相同任期的 `false reply`：服务器对应的 `nextIndex--`。
    - 相同任期的 `true reply`：更新 `nextIndex` 和 `matchIndex`。统计，找出一个被大部分 `Follower` 提交的 `N` 值，把直到这个 `index` 的 `log`应用到 `Leader` 自己的状态机中。

同时，在测试时发现，可能会有前一个测试的 `Server` 发送心跳包的 `goroutine` 没有正确关闭，影响之后的test，在发送心跳包的循环中添加 `rf.killed() == false` 的判断。

```Go
func (rf *Raft) handleAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	rf.lastUpdate = time.Now()

	if rf.role != Leader {
		return
	}

	if reply.Term > rf.currentTerm {
		rf.currentTerm = reply.Term
		rf.convertRole(Follower)
		return
	}

	if !reply.Success {
		if rf.nextIndex[server] > 0 {
			rf.nextIndex[server]--
		}
		return
	}

	rf.nextIndex[server] = len(rf.logs)
	rf.matchIndex[server] = args.PrevLogIndex + len(args.Entries)

	serversCount := len(rf.peers)

	for n := rf.GetLastLogIndex(); n > rf.commitIndex; n-- {
		count := 1
		if rf.logs[n].Term == rf.currentTerm {
			for i := 0; i < serversCount; i++ {
				if i != rf.me && rf.matchIndex[i] >= n {
					count++
				}
			}
		}
		if count > len(rf.peers)/2 {
			rf.commitIndex = n
			go rf.applyLogs()
			break
		}
	}
}
```
