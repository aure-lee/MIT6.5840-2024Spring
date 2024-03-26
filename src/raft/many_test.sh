#!/bin/bash

# 检查参数数量
if [ $# -ne 2 ]; then
    echo "Usage: $0 <number-of-times> <test-case>"
    exit 1
fi

# 运行次数和测试用例
times=$1
test_case=$2
# 成功次数
success=0
warn=0

# 启动测试并行运行
for ((i=1; i<=times; i++))
do
    # 在后台运行测试，并将输出重定向到临时文件
    go test -run $test_case > tmp_$i.out 2>&1 &
done

# 等待所有后台进程完成
wait

# 检查每个测试的结果
for ((i=1; i<=times; i++))
do
    # 检查测试输出，这里假设"PASS"出现在输出中表示成功
    if grep -q "ok" tmp_$i.out; then
        ((success++))
    fi

    if grep -q "warn" tmp_$i.out; then
        ((warn++))
    fi
    # 清理临时文件
    rm tmp_$i.out
done

# 打印成功次数
echo "Success: $success out of $times"
echo "Warn: $warn out of $times"

# grep -r "exit" ./tmp*
