# Lab3D - solution

3D的解法仅供参考，因为在实现时发现在实现3A-3C时没有考虑到索引的问题，同时对于锁和通道的掌握不够熟练，导致整体的代码就是一坨，而且积重难返，只能接着在屎山上拉屎。

甚至考虑过把整体的代码重写一遍，但是还是放弃了，只能说碰上的坑会在后面慢慢说。

## Hints

1. 一个好的起点是修改你的代码，使其能够仅存储从某个索引 `X` 开始的日志部分。最初你可以将 `X` 设置为零，并运行 `3B/3C` 测试。然后，让 `Snapshot(index)` 丢弃索引之前的日志，并将 `X` 设置为该索引。如果一切顺利，你现在应该能通过第一个3D测试。
2. 接下来：如果领导者没有必要的日志条目来使追随者更新到最新状态，就让它发送一个 `InstallSnapshot RPC`。
3. 通过单个 `InstallSnapshot RPC` 发送整个快照。不要实现图13的偏移机制来分割快照。
4. Raft必须以一种方式丢弃旧的日志条目，以允许Go垃圾收集器释放和重新使用内存；这要求没有可达的引用（指针）指向被丢弃的日志条目。
5. 对于完整的 Lab3 test（3A+3B+3C+3D），在没有 `-race` 的情况下，合理的消耗时间是 `real` 的6分钟和 `CPU` 的1分钟。在使用 `-race` 时，大约是`real` 的10分钟和 `CPU` 的2分钟。

## Solution

### 坑1 - 日志压缩后的索引问题

每次保存快照后，都会丢弃一部分的日志，这样原先 `index=6` 的 `log` 可能会提前到实际存储位置 `index=0` 的位置。因此需要逻辑索引和存储索引的转换，这样就需要在 Raft 中保存已经压缩的最后日志索引，同时，该索引也需要持久化。

```go
// getLastStorageIndex get the newest storage index
func (rf *Raft) getLastStorageIndex() int {
	return len(rf.logs) - 1
}

// getLastLogicalIndex get the newest logical index
func (rf *Raft) getLastLogicalIndex() int {
	return rf.convertToLogicalIndex(len(rf.logs) - 1)
}

// converts a logical index into a storage index.
func (rf *Raft) convertToStorageIndex(logicalIndex int) int {
	return logicalIndex - rf.lastIncludedIndex
}

// converts a storage index into a logical index.
func (rf *Raft) convertToLogicalIndex(storageIndex int) int {
	return storageIndex + rf.lastIncludedIndex
}
```

对 AppendEntries 和 Start 中有关索引的部分进行修改，:sweat_smile: :sweat_smile: :sweat_smile:，我操了，这几把索引改了我三天。有个地方不小心逻辑索引改成了存储索引，debug了一整天，我操了！！！！！！！！！！！

:sweat_smile: 选举的索引也改错了，有个很短的Follower在日志压缩后当选了Leader。

注意：如果要修改raft中的 `lastIncludedIndex` 和 `lastIncludedTerm`，必须要在所有的逻辑索引和存储索引判断完才能修改。这个比较好debug：如果你发现安装完snapshot后的服务器，最新日志的 `Index` 比 `Leader` 还大，就是有地方的 `lastIncludedIndex` 改早了。

### 坑2 - 应用Snapshot与AppendEntries发来的包之间的冲突

`Test 3D-1` 不会测试发送 `InstallSnapshot RPC` 的功能，只需要实现 `Snapshot()` 函数。每当日志条目数+1 模 10 为 0 时，就会触发压缩函数。

当我实现 `Snapshot()` 函数并运行测试时，发现服务器会陷入死锁，当我把 `Snapshot()` 的锁注释后，就不会产生死锁。为了找到哪一个函数与 `Snapshot` 有死锁关系，通过注释每一对锁，发现了 `ApplyLogs()` 函数和 `Snapshot()` 产生了死锁。

可能在 `ApplyLogs()` 持有锁的同时，`applych` 被阻塞了，导致了死锁，通过查阅资料（看别人的代码），应该有三个解决方向：
1. 改造收到 AppendEntries 时的同步机制，在 `ApplyLogs()` 内使用条件变量 `sync.Cond`。
2. 把 `ApplyLogs()` 进行更详细的拆分，对具体每一步的代码进行加锁和释放的操作。
3. 把 `ApplyLogs()` 中传输到 `applych` 的部分提取出来，每次把要发送到 `applych` 的信息进行封装，使用 goroutine 将封装好的信息 `Msgs` 发送给 `applych`。

总结：在使用 `channel` 时，不要持有锁。


