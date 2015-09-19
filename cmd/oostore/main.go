package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"gopkg.in/errgo.v1"

	"github.com/cmars/oostore"
	"github.com/cmars/oostore/postgres"
)

const (
	defaultAddr = "127.0.0.1:20080"
)

func main() {
	app := cli.NewApp()
	app.Name = "oostore"
	app.Usage = "oostore [--addr ADDR] [--prefix PREFIX] [database connect string]"
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
		dsn := "host=/var/run/postgresql database=oostore"
		if c.Args().Present() {
			dsn = strings.Join(c.Args(), " ")
		}
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			log.Fatalf("cannot connect to database: %s", errgo.Details(err))
		}
		objectStore, err := postgres.NewObjectStorage(db)
		if err != nil {
			log.Fatalf("failed to instantiate object storage: %s", errgo.Details(err))
		}
		bakeryStore, err := postgres.NewBakeryStorage(db)
		if err != nil {
			log.Fatalf("failed to instantiate bakery storage: %s", errgo.Details(err))
		}
		service, err := oostore.NewService(oostore.ServiceConfig{
			ObjectStore: objectStore,
			BakeryStore: bakeryStore,
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
