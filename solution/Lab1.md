# Lab1

## Topic

实现分布式mr,一个`coordinator`,一个`worker`（启动多个）,在这次实验都在一个机器上运行。`worker`通过`rpc`和`coordinator`交互。`worker`请求任务,进行运算,写出结果到文件。`coordinator`需要关心`worker`的任务是否完成，在超时情况下将任务重新分配给别的`worker`。

## Rules

1. 映射(`map`)阶段应将中间键分成多个桶，供 `nReduce` 减缩任务使用，其中 `nReduce` 是减缩任务的数量，也就是 `main/mrcoordinator.go` 传递给 `MakeCoordinator()` 的参数。每个映射器(`mapper`)应创建 `nReduce` 中间文件，供还原任务使用。
2. `Worker` 实现应将第 X 个还原任务的输出放到 `mr-out-X` 文件中。
3. `mr-out-X` 文件应包含每个 `Reduce` 函数输出的一行。这一行应该以 Go 的 `"%v %v"`格式生成，并以键和值调用。请查看 `main/mrsequential.go` 中注释为 **"这是正确格式"** 的一行。如果您的实现与此格式偏差过大，测试脚本就会失败。
4. 您可以修改 `mr/worker.go`、`mr/coordinator.go` 和 `mr/rpc.go`。您可以临时修改其他文件进行测试，但请确保您的代码能在原始版本上运行；我们将使用原始版本进行测试。
5. `Worker` 应将中间的 `Map` 输出放到当前目录下的文件中，这样 `Worker` 就能将其作为 `Reduce` 任务的输入进行读取。
6. `main/mrcoordinator.go` 希望 `mr/coordinator.go` 实现一个 `Done()` 方法，当 `MapReduce` 作业完全完成时返回 `true`；此时，`mrcoordinator.go` 将退出。
7. 任务完全完成后，工作进程应退出。实现这一点的一个简单方法是使用 `call()` 的返回值：如果 `worker` 无法联系`coordinator`，它可以认为 `coordinator` 已经退出，因为工作已经完成，所以 `worker` 也可以终止。根据您的设计，您可能还会发现，`coordinator` 向 `worker` 下达一个 **"请退出"** 的伪任务也很有帮助。

## Hints

1. 一种开始的方法是修改 `mr/worker.go` 中的 `Worker()` 函数，使其发送一个 RPC 到协调器请求一个任务。然后修改协调器以响应一个尚未开始的 map 任务的文件名。接着修改 worker 来读取该文件并调用应用程序的 Map 函数，如 `mrsequential.go` 中所示。（提示使用：Worker请求master的方法）
2. Map 和 Reduce 函数是在运行时使用 Go `plugin` 包从名称以 .so 结尾的文件中动态加载的。`.so`是Unix的动态链接库，win中是 `.dll` 。
3. 如果修改了mr文件夹的内容，需要重新构建 `wc.so`。
4. 这个实验在同一台机器上运行，依赖本地的文件系统，但如果工作者运行在不同的机器上，则需要一个类似 GFS 的全局文件系统。
5. 中间文件命名为 `mr-X-Y`，其中 X 是 map 任务号，Y 是 reduce 任务号。(一个map生成的中间文件分成 `nReduce` 份。)
6. Worker 的 map 任务代码需要一种方法，将中间 `key/value pair` 存储在文件中，以便在 reduce 任务中正确读回。其中一种方法是使用 Go 的 `encoding/json` 包。将 JSON 格式的 `key/value pair` 写入开放文件：

    enc := json.NewEncoder(file)
    for _, kv := ... {
        err := enc.Encode(&kv)

并读回这样的文件：

    dec := json.NewDecoder(file)
    for {
        var kv KeyValue
        if err := dec.Decode(&kv); err != nil {
            break
        }
        kva = append(kva, kv)
    }

因为真实的分布式系统都是在网络上的，网络数据传输使用的是 `json` 格式。这样可以模拟网络传输。
7. Worker 的 map 部分可以使用 `ihash(key)` 函数为给定 key 挑选还原任务。为了减小输出文件的数量，可以使用 `ihash(key) % nReduce` ，让相同的key传入到一个文件中。
8. 参考 `mrsequential.go` 文件，实现 Map 阶段读取文件，Map和Reduce阶段对 `key/value pair` 进行排序，Reduce 阶段输出文件。
9. 作为 RPC 服务器，协调器将是并发的；不要忘记锁定共享数据。(锁定任务)
10. `go run -race`: 使用Go的竞态检测器，`test-mr.sh` 有关于如何使用的说明。
11. Workers 有时需要等待，例如在最后一个 `map task` 完成之前，`reduce task` 不能开始。一种方案是，Worker定期向coordinator请求任务，等待时间使用 `time.Sleep()` 。另一种方案是，Coordinator中的相关RPC处理程序可以使用 `time.Sleep()` 或 `sync.Cond` 循环等待。在 Coordinator 中，新建一个线程处理每个RPC，因此一个处理程序正在等待时，不妨碍 Coordinator 处理其他的RPC。
12. Coordinator 无法区分崩溃的 Worker，存活但停滞的Worker，以及正在执行但速度太慢的 Worker。最好的办法是让 Coordinator 等待一段时间，然后放弃并重新分配。在本实验中，让 Coordinator 等待10秒。
13. 如果选择执行备份任务（Section 3.6），会测试代码是否在 Worker 执行任务而不崩溃的情况下安排无关的任务。备份任务只能在一段相对较长的时间后安排（如10秒）。
14. 使用 `mrapps/crash.go` 应用插件，可以测试崩溃恢复功能。
15. 为了确保没有Worker在执行 Reduce 时观察到崩溃的部分写入的文件，MapReduce论文提到了使用临时文件，并在完全写入后原子重命名的技巧。可以使用 `ioutil.TempFile` 或 `os.CreateTemp` (Go 1.17 and later)创建临时文件，并使用 `os.Rename` 原子重命名临时文件。
16. `test-mr.sh` 的所有进程都在 `mr-tmp` 子目录下运行，因此如果出现问题，可以在该目录下查看中间文件。可以随意修改 `test-mr.sh`，使其在测试失败后退出，这样脚本就不会继续执行（并覆盖输出文件）。
17. `test-mr-many.sh` 会连续多次运行 `test-mr.sh`，这是为了发现低概率错误。它的参数是运行测试的次数。不应并行运行多个 `test-mr.sh`，因为 coordinator 会重复使用同一个 socket，从而导致冲突。

## Your Job

Your job is to implement a distributed MapReduce, consisting of two programs, the coordinator and the worker. There will be just one coordinator process, and one or more worker processes executing in parallel. In a real system the workers would run on a bunch of different machines, but for this lab you'll run them all on a single machine. The workers will talk to the coordinator via RPC. Each worker process will, in a loop, ask the coordinator for a task, read the task's input from one or more files, execute the task, write the task's output to one or more files, and again ask the coordinator for a new task. The coordinator should notice if a worker hasn't completed its task in a reasonable amount of time (for this lab, use ten seconds), and give the same task to a different worker.

We have given you a little code to start you off. The "main" routines for the coordinator and worker are in `main/mrcoordinator.go` and `main/mrworker.go`; don't change these files. You should put your implementation in `mr/coordinator.go`, `mr/worker.go`, and `mr/rpc.go`.

Here's how to run your code on the word-count MapReduce application. First, make sure the word-count plugin is freshly built:

    $ go build -buildmode=plugin ../mrapps/wc.go

In the `main` directory, run the coordinator.

    $ rm mr-out*
    $ go run mrcoordinator.go pg-*.txt

The `pg-*.txt` arguments to `mrcoordinator.go` are the input files; each file corresponds to one "split", and is the input to one Map task.
In one or more other windows, run some workers:

    $ go run mrworker.go wc.so

When the workers and coordinator have finished, look at the output in `mr-out-*`. When you've completed the lab, the sorted union of the output files should match the sequential output, like this:

    $ cat mr-out-* | sort | more
    A 509
    ABOUT 2
    ACT 8
    ...

We supply you with a test script in `main/test-mr.sh`. The tests check that the `wc` and `indexer` MapReduce applications produce the correct output when given the `pg-xxx.txt` files as input. The tests also check that your implementation runs the Map and Reduce tasks in parallel, and that your implementation recovers from workers that crash while running tasks.

If you run the test script now, it will hang because the coordinator never finishes:

    $ cd ~/6.5840/src/main
    $ bash test-mr.sh
    *** Starting wc test.

You can change `ret := false` to true in the Done function in `mr/coordinator.go` so that the coordinator exits immediately. Then:

    $ bash test-mr.sh
    *** Starting wc test.
    sort: No such file or directory
    cmp: EOF on mr-wc-all
    --- wc output is not the same as mr-correct-wc.txt
    --- wc test: FAIL
    $

The test script expects to see output in files named `mr-out-X`, one for each reduce task. The empty implementations of `mr/coordinator.go` and `mr/worker.go` don't produce those files (or do much of anything else), so the test fails.

When you've finished, the test script output should look like this:

    $ bash test-mr.sh
    *** Starting wc test.
    --- wc test: PASS
    *** Starting indexer test.
    --- indexer test: PASS
    *** Starting map parallelism test.
    --- map parallelism test: PASS
    *** Starting reduce parallelism test.
    --- reduce parallelism test: PASS
    *** Starting job count test.
    --- job count test: PASS
    *** Starting early exit test.
    --- early exit test: PASS
    *** Starting crash test.
    --- crash test: PASS
    *** PASSED ALL TESTS
    $

You may see some errors from the Go RPC package that look like

    2019/12/16 13:27:09 rpc.Register: method "Done" has 1 input parameters; needs exactly three

Ignore these messages; registering the coordinator as an RPC server checks if all its methods are suitable for RPCs (have 3 inputs); we know that `Done` is not called via RPC.

Additionally, depending on your strategy for terminating worker processes, you may see some errors of the form

    2024/02/11 16:21:32 dialing:dial unix /var/tmp/5840-mr-501: connect: connection refused

It is fine to see a handful of these messages per test; they arise when the worker is unable to contact the coordinator RPC server after the coordinator has exited.