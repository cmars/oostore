package main

import (
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/cli"
	"gopkg.in/errgo.v1"

	"github.com/cmars/oostore"
)

const (
	defaultAddr = "127.0.0.1:20080"
)

func main() {
	app := cli.NewApp()
	app.Name = "oostore"
	app.Usage = "opaque object storage service"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "addr",
			Value: defaultAddr,
		},
		cli.StringFlag{
			Name: "prefix",
		},
	}
	app.Action = func(c *cli.Context) {
		// TODO: persistent object & macaroon storage
		store := oostore.NewMemStorage()
		service, err := oostore.NewService(oostore.ServiceConfig{
			ObjectStore: store,
			Prefix:      c.String("prefix"),
		})
		if err != nil {
			log.Fatalf("failed to create service: %s", errgo.Details(err))
		}
		addr := c.String("addr")
		log.Printf("listening for requests on %s", addr)
		err = http.ListenAndServe(addr, service)
		if err != nil {
			log.Fatalf("server error: %s", errgo.Details(err))
		}
	}
	app.Run(os.Args)
}
