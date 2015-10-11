/*
 * Copyright 2015 Casey Marshall
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"gopkg.in/errgo.v1"
	"gopkg.in/tomb.v2"

	"github.com/cmars/oostore"
	"github.com/cmars/oostore/postgres"
)

const (
	defaultHTTP  = "127.0.0.1:20080"
	defaultHTTPS = ":20443"
)

func main() {
	app := cli.NewApp()
	app.Name = "oostore"
	app.Usage = "Opaque Object Storage Service"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "http",
			Value: defaultHTTP,
			Usage: "HTTP listen address",
		},
		cli.StringFlag{
			Name:  "https",
			Usage: "HTTPS listen address",
		},
		cli.StringFlag{
			Name:  "cert",
			Usage: "TLS certificate and certification chain, PEM encoded",
		},
		cli.StringFlag{
			Name:  "key",
			Usage: "TLS private key, PEM encoded",
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

		var t tomb.Tomb

		httpAddr := c.String("http")
		if httpAddr != "" {
			t.Go(func() error {
				log.Printf("listening for HTTP requests on %q", httpAddr)
				err := http.ListenAndServe(httpAddr, service)
				if err != nil {
					log.Fatalf("server error: %s", errgo.Details(err))
				}
				return err
			})
		}
		httpsAddr := c.String("https")
		if httpsAddr != "" {
			t.Go(func() error {
				certFile := c.String("cert")
				if certFile == "" {
					log.Fatalf("missing --cert flag")
				}
				keyFile := c.String("key")
				if keyFile == "" {
					log.Fatalf("missing --key flag")
				}
				log.Printf("listening for HTTPS requests on %q", httpsAddr)
				err := http.ListenAndServeTLS(httpsAddr, certFile, keyFile, service)
				if err != nil {
					log.Fatalf("server error: %s", errgo.Details(err))
				}
				return err
			})
		}
		t.Wait()
	}
	app.Run(os.Args)
}
