# Solution

## 1 阅读代码并理解

阅读main, mr, mrapps文件夹中的相关代码，取消`worker.go`中 CallExample() 
的注释，并运行测试。

- [x] 2024/02/27

## 2 任务分配机制

`master = coordinator`

### 2.1 任务分配 - master与worker之间的关系

有两种任务分配机制，第一种是 `worker` 向 `master` 请求任务，
第二种是 `master` 向 `worker` 发送任务。

1. Worker主动向Master申请任务：在这种机制中，Worker节点在准备好执行新任务时，会向Master节点发送请求以获取任务。Master节点维护一个待分配任务的队列，并根据到来的请求将任务分配给请求的Worker。
2. Master主动向Worker发送任务：在这种机制中，Master节点负责跟踪所有可用的Worker节点，并根据某种策略（如负载均衡、数据局部性等）主动将任务分配给Worker。（在真实的分布式系统中，当worker机器上有部分任务的相关文件时，master会优先分配这部分任务给这个worker）。这要求Master具有全局视图，并能够实时监控所有Worker的状态。

- Worker主动申请：这种方式相对来说比较简单。此时，Master仅仅是任务调度和分配中心，只需要响应worker的请求，并将代办任务分配给他们。此外，这种方法可以自然地处理Worker节点的异步和非均匀完成时间，因为，Worker只有在准备号接受新任务时，才会请求任务。
- Master主动发送任务：Master需要持续跟踪每个Worker的状态，根据worker的状态来控制任务的分配。这就要求worker需要采用心跳包机制，将worker的加入，故障，离开发送给master。同时，Master中需要一个数据结构存放这些信息。

在这个lab中，我采用了Worker主动申请的机制，因为稍微方便一点。

### 2.2 任务分配 - RPC数据结构定义

在 `rpc.go` 中定义 `worker` 的 `request` 和 `response` 数据结构：

`request` 的 `args` 用来向 `coordinator` 请求task，`request` 的 `replys` 用来保存 `coordinator` 分配给 `worker` 的任务。

`args` 其中应该包括 worker 的 ID，`coordinator` 中保存时将其与分配给该 worker 的 task 绑定在一起。
- 一个问题：WorkerID 应该由 Coordinator 分配，还是由 Worker 申请：
    Coordinator分配：Coordinator维护一个全局WorkerID，每次新的Worker申请task时，分配ID。但是Worker在没有WorkerID时，需要生成一个独特的临时标识来让 `Coordinator` 识别。感觉有点多此一举。
    Worker申请：`Worker`在启动时自行生成一个唯一ID，然后将这个ID连同注册请求一起发送给Coordinator。此时要求Worker有能够生成唯一标识符的能力。在真实分布式系统上，可以使用IP地址、端口和时间戳来生成ID。在此实验中，可以简单地使用pid来表示唯一的WorkerID。

`replys` 应该包括 task 的种类（map, reduce, wait, exit），需要处理的输入文件(map: read one file and output lists of files/reduce: read lists of files and output one result file)，task 的识别码（ID->mapID/reduceID）。

```go
type RequestTaskReplys struct {
	TaskType   TaskType
	TaskID     int
	InputFiles []string
	ReduceNum  int
}
```

对Map任务来说，TaskID = MapID，ReduceNum 即为生成 `mr-tmp-X-Y` 中 `Y` 的数量。

`response` 的 `args` 用来向 `coordinator` 汇报 task 完成情况，`replys` 置空。

`args` 应该包括当前 task 的类型（TaskType），TaskID（第几个map/reduce task），WorkerID（完成这个task的worker）。

### 2.3 任务分配 - master如何分发task

- 这个系统处于 Map 阶段，同时，MapTask 队列中仍然含有 Task：
    master 分配 map task 给 worker。
- 这个系统处于 Map 阶段，但是此时 MapTask 任务队列中没有 Task：
    master 让 worker 处于 waiting 状态。
- 这个系统处于 Reduce 阶段，同时，Reduce 队列中还有 Task：
    master 分配 reduce task 给 worker：
- 这个系统 处于 Reduce 阶段，但是此时 ReduceTask 队列中没有 Task：
    master 让 Worker 处于 waiting 状态。
- 这个系统处于 AllDone 阶段：
    master 让 worker 退出（exit）。

- [x] 2024/02/28

### 2.4 task在master中的存储

存储task可以自己设计一个共享的数据结构，例如一个共享的队列，存储任务时遵守先进先出规则即可。同时在task进出数据结构时加锁，防止同一个任务被分配给两个 worker。
Go中有 `channel` 这个数据结构用来实现安全的共享数据，更适合这个实验，因为自己设计的队列还需要实现 **加锁** 功能。

## 3 task 的设计

### 3.1 task的数据结构

`task` 需要指明任务的类型（TaskType），任务的状态（TaskStat），任务的输入文件（InputFiles），任务的ID（MapID/ReduceID），任务分配的对象（WorkerID），以及任务的开始时间（StartTime）。

```go
type Task struct {
    TaskType TaskType
    TaskStat TaskStat
    InputFiles []string
    TaskID int
    WorkerID int
    StartTime time.Time
}
```

### 3.2 coordinator生成MapTask

当 master 刚开始启动生成所有的 MapTask，存放在 `Coordinator` 的 MapTask Channel 中。

### 3.3 coordinator生成ReduceTask

首先，ReduceTask生成的时间在 `Coordinator` 从 `MapPhase` 转到 `ReducePhase` 的阶段。
同时，ReduceTask的生成需要依赖上一阶段 Map 生成的中间文件 `mr-tmp-X-Y` ，将同一个 `Y` 分成一组，作为一个 IuputFiles，传递给 `worker` 。

## 4 请求task和报告task

### 4.1 worker请求task和报告task

worker通过RPC请求coordinator来请求和报告task，只需要把arg的参数设置好，使用 `call("Coordinator.RequestTask/ReportTask")` 。

### 4.2 coordinator响应请求task和报告task

请求task：根据 coordinator 当前的状态，选择任务给 replys，同时启动一个 goroutine 检测 `timeout`。如果发送的任务是 `wait/exit`，则不用进行任何操作。

报告task：根据 `workerId` 和 `task` 的状态，转换当前 `task` 的状态。1. task信息与report的信息可以对上，`task` 转换为 `Finished`， 同时额外维护一个int型数据，保存Map/Reduce阶段已经完成的任务数。2. 信息不符，`task` 转换为 `NotAssign`。

## 5 Coordinator的状态

在启动 Coordinator Server 之前，新启动一个一直循环的 goroutine，在其中检测当前 `phase`，当已完成的任务数 == 总任务数时，coordinator 转换为下一个阶段。
