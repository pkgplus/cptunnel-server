package main

import (
	"flag"
	"github.com/pkgplus/cmdplus-tunnel/server"
)

var (
	port                   int
	timeout                int64
	bindAddr, domainSuffix string
	signKey                string
)

func init() {
	flag.IntVar(&port, "port", 80, "port")
	flag.StringVar(&bindAddr, "bind", "", "bind addr")
	flag.Int64Var(&timeout, "timeout", 15, "tunnel request timeout")
	flag.StringVar(&domainSuffix, "domain", "cmd.plus", "domain")
	flag.StringVar(&signKey, "key", "c8706bab7db59103a6bfd36e0c6b42e35d3f55d5", "sign key")
	flag.Parse()

	//logrus.SetLevel(logrus.DebugLevel)
	//remotedialer.PrintTunnelData = true
}

func main() {
	s := server.New(port, bindAddr, domainSuffix, signKey, timeout)
	s.Serve()
}
