package main

import (
	"fmt"
	"log"

	"github.com/youngfr/dcls/internal/simpleserver"
)

func main() {

	// 追加一条记录 => 返回新追加的记录的下标
	// curl -X POST 127.0.0.1:8080 -d '{"record": {"value": "TGV0J3MgR28gIzEK"}}'
	// 根据下标获取记录 => 返回获取到的值
	// curl -X GET  127.0.0.1:8080 -d '{"offset": 0}'

	fmt.Println("Listening in port 8080...")
	log.Fatal(simpleserver.NewHTTPServer("127.0.0.1:8080").ListenAndServe())
}
