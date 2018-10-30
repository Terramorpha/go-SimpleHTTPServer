package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	b := bufio.NewReader(os.Stdin)
	var (
		s    string
		err  error
		line []byte
	)
	for {
		line, _, err = b.ReadLine()
		if err != nil {
			panic(err)
		}
		s = string(line)
		addr, port, _ := net.SplitHostPort(s)
		fmt.Printf("%s:%s\n", addr, port)
		fmt.Println(net.ParseIP(addr))
		fmt.Println(net.LookupPort("tcp", port))

	}
}
