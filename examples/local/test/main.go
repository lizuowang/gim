package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lizuowang/gim/pkg/helper"
)

var lock sync.RWMutex

type MyStruct struct {
	Name string
	Age  int
}

func mapToStruct2(m map[string]interface{}, s *MyStruct) error {
	//通过json序列化
	bytes, err := json.Marshal(m)
	if err != nil {
		return err
	}
	json.Unmarshal(bytes, s)
	return nil
}

func main() {

	// 假设我们有一个 map
	data := map[string]interface{}{
		"Name": "lizi",
		"Age":  30.2,
	}
	var myStruct = &MyStruct{}

	// 启动10000个goroutine
	// lock := sync.RWMutex{}

	// a := "2342sdfsdf"
	// b := ":23422"

	// c := a + b
	// m := map[string]map[string]string{
	// 	a: {"n": a},

	// 	a + b: {"n": a + b},
	// }

	// d := map[string]string{}
	start := time.Now()
	for i := 0; i < 1000000; i++ {
		helper.MapToStruct(data, myStruct)

	}

	elapsed := time.Since(start)
	fmt.Printf("Goroutine startup took %s\n", elapsed)

	fmt.Println(myStruct)
}

func test() {
	// defer func() {
	// 	if r := recover(); r != nil {

	// 	}
	// }()

	lock.RLock()
	lock.RUnlock()
}
