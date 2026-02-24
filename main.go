package main

import (
	"fmt"

	"a-sys.one/client"
	"a-sys.one/server"
)

const filename = "a.txt"

func main() {

	server := server.NewServer("localhost", 8085, "D:\\Upload")
	go server.Listen()

	client := client.New()
	err := client.SendFile("http://localhost:8085", filename)
	if err != nil {
		fmt.Println(err)
		return
	}
}
