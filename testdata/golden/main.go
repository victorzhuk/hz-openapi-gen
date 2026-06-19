package main

import (
	"github.com/cloudwego/hertz/pkg/app/server"

	"example.com/service/biz/router"
)

func main() {
	h := server.Default(server.WithHostPorts(":8888"))
	router.GeneratedRegister(h)
	h.Spin()
}
