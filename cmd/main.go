package main

import (
	"log"

	"github.com/faceair/betproxy"
)

func main() {
	service, err := betproxy.NewService(3128)
	if err != nil {
		panic(err)
	}
	log.Fatal(service.Listen())
}
