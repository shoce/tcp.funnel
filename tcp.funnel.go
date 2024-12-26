/*
history:
2015-10-13 v1

GoGet
GoFmt
GoBuildNull
GoBuild
GoRun
*/

package main

import (
	"expvar"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

func usage() {
	fmt.Println("usage:")
	fmt.Println("tcpfunnel input_address:port input_capacity output_address:port output_capacity debug_address:port")
	fmt.Println("example:")
	fmt.Println("tcpfunnel 127.0.0.1:6432 10000 127.0.0.1:5432 100 127.0.0.1:8432")
}

func main() {
	var err error
	if len(os.Args) < 6 {
		usage()
		os.Exit(1)
	}
	inaddr := os.Args[1]
	var incap int
	if incap, err = strconv.Atoi(os.Args[2]); err != nil {
		usage()
		os.Exit(1)
	}
	outaddr := os.Args[3]
	var outcap int
	if outcap, err = strconv.Atoi(os.Args[4]); err != nil {
		usage()
		os.Exit(1)
	}
	debugaddr := os.Args[5]

	expinlen := expvar.NewInt("inlen")
	expincap := expvar.NewInt("incap")
	expoutlen := expvar.NewInt("outlen")
	expoutcap := expvar.NewInt("outcap")

	insem := make(chan bool, incap)
	expincap.Set(int64(incap))
	outsem := make(chan bool, outcap)
	expoutcap.Set(int64(outcap))

	go http.ListenAndServe(debugaddr, nil)
	l, err := net.Listen("tcp4", inaddr)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	log.Printf("Listening on `%s`.\n", inaddr)

	go func() {
		for {
			expinlen.Set(int64(len(insem)))
			expoutlen.Set(int64(len(outsem)))
			time.Sleep(1 * time.Second)
		}
	}()

	for {
		insem <- true
		c1, err := l.Accept()
		if err != nil {
			log.Printf("Could not Accept connection: `%s`.\n", err)
			continue
		}
		go func(c1 net.Conn) {
			defer func() {
				<-outsem
				<-insem
			}()
			defer c1.Close()
			outsem <- true
			c2, err := net.Dial("tcp4", outaddr)
			if err != nil {
				log.Printf("Could not Dial `%s`.", outaddr)
				return
			}
			defer c2.Close()
			log.Printf("Connected `%s`.", c1.RemoteAddr())
			go io.Copy(c1, c2)
			io.Copy(c2, c1)
		}(c1)
	}
}
