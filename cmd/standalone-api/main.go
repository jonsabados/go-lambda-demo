package main

import (
	"flag"
	"net/http"

	"github.com/jonsabados/go-lambda-demo/cmd"
)

func main() {
	listenAddress := flag.String("listen-address", ":8080", "address to listen to for inbound requests")
	handler := cmd.CreateAPI()
	err := http.ListenAndServe(*listenAddress, handler)
	if err != nil {
		panic(err)
	}
}
