package main

//
// a word-count application "plugin" for MapReduce.
//
// go build -buildmode=plugin wc.go
//

import (
	"strconv"
	"strings"
	"unicode"

	"6.5840/mr"
)

// The map function is called once for each file of input. The first
// argument is the name of the input file, and the second is the
// file's complete contents. You should ignore the input file name,
// and look only at the contents argument. The return value is a slice
// of key/value pairs.
func Map(filename string, contents string) []mr.KeyValue {
	// filename：文件名 contents：文件内的所有内容

	// function to detect word separators.
	ff := func(r rune) bool { return !unicode.IsLetter(r) }
	// 找出文件中不是字母的unicode码点
	// rune类型的参数（在Go中，rune代表一个Unicode码点）
	// 这个函数对于处理多语言文本特别有用，因为它能够识别
	// 包括拉丁字母、希腊字母、阿拉伯字母、汉字、日文假名等在内的各种语言的字母字符。

	// split contents into an array of words.
	words := strings.FieldsFunc(contents, ff)

	kva := []mr.KeyValue{} // 总kv键值对（需要return的kv键值对）
	for _, w := range words {
		kv := mr.KeyValue{w, "1"} // 每个单词生成对应的kv键值对
		kva = append(kva, kv)
	}
	return kva
}

// The reduce function is called once for each key generated by the
// map tasks, with a list of all the values created for that key by
// any map task.
// reduce函数对由map任务生成的每个键调用一次，带有任何map任务为该键创建的所有值的列表。
// return 这个word出现数量的string形式
func Reduce(key string, values []string) string {
	// return the number of occurrences of this word.
	return strconv.Itoa(len(values))
	// 计算key这个word对应kv键值对的数量并转成string类型
}
