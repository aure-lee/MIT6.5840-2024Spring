# Lab3C - solution

## Solution

根据给的例子实现 `persist` 和 `readPersist`，同时需要持久化的内容是 `currentTerm`，`votedFor` 和 `logs`。在每次这三个内容修改的时候执行持久化操作。

测试后 `Figure 8 Unreliable` 跑不过，经过打印日志调试发现，在这个测试中，`Leader` 只有两次发送 `AppendEntries` 的机会，因此需要优化冲突条目的处理。
根据 `Students' Guide to Raft` 的优化思路，在 `Reply` 中增加两个字段：`ConflictIndex` 和 `ConflictTerm`，并在 `AppendEntries` 和 `handleAppendEntries` 中增加新的处理思路。

```go
	// 如果有附加日志
	if args.PrevLogIndex > rf.GetLastLogIndex() {
		reply.Term, reply.Success = rf.currentTerm, false
		reply.ConflictIndex = len(rf.logs)
		reply.ConflictTerm = 0
		return
	}

	if args.PrevLogIndex >= 0 && rf.logs[args.PrevLogIndex].Term != args.PrevLogTerm {
		reply.Term, reply.Success = rf.currentTerm, false
		reply.ConflictTerm = rf.logs[args.PrevLogIndex].Term
		index := args.PrevLogIndex
		for index >= 0 && rf.logs[index].Term >= reply.ConflictTerm {
			index--
		}
		reply.ConflictIndex = index + 1
		return
	}
```

```go
	if !reply.Success {
		if reply.ConflictTerm != 0 {
			index := rf.nextIndex[server] - 1
			for index >= 0 && rf.logs[index].Term > reply.ConflictTerm {
				index--
			}
			if index > 0 && rf.logs[index].Term == reply.ConflictTerm {
				rf.nextIndex[server] = index + 1
				return
			}
		}
		rf.nextIndex[server] = reply.ConflictIndex
		return
	}
```

调整完后可以通过测试，但是当测试 $30$ 次后，可能会有 `Apply Error` 的错误：

```text
2024/03/19 22:14:49 1: log map[1:3925858605185290878 2:6408371276025774760 3:2054451560272905856 4:2364716476040125810 5:6789850213568667680 6:7554945423124268051 7:5373716142603459906]; server map[1:3925858605185290878 2:6408371276025774760 3:2054451560272905856 4:2364716476040125810 5:6789850213568667680 6:7554945423124268051 7:5373716142603459906 8:810241063673241059 9:5284359830032938821 10:292187576632357111 11:2624624078460021631]
2024/03/19 22:14:49 apply error: commit index=8 server=1 4206067583696703960 != server=4 810241063673241059
```

- [x] 2024/03/19
