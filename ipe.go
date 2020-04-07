// Copyright 2015 Claudemiro Alves Feitosa Neto. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ipe

import (
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	log "github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
	"ipe/api"
	"ipe/app"
	"ipe/config"
	"ipe/storage"
	"ipe/websockets"
)

// Start Parse the configuration file and starts the ipe server
// It Panic if could not start the HTTP or HTTPS server
func Start(filename string) {
	rand.Seed(time.Now().Unix())

	conf, _ := getConfFromFile(filename)
	router, _ := getRouter(*conf)
	var wg = sync.WaitGroup{}

	serverErr := make(chan error)
	wg.Add(1)
	go func() {
		log.Infof("Starting HTTP service on %s ...", conf.Host)
		serverErr <- http.ListenAndServe(conf.Host, router)
	}()

	if conf.SSL.Enabled {
		wg.Add(1)
		go func() {
			log.Infof("Starting HTTPS service on %s ...", conf.SSL.Host)
			serverErr <- http.ListenAndServeTLS(conf.SSL.Host, conf.SSL.CertFile, conf.SSL.KeyFile, router)
		}()
	}

	log.Info(<-serverErr)
	wg.Wait()
}

func getConfFromFile(filename string) (*config.File, error) {
	var conf config.File

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Expand env vars
	data = []byte(os.ExpandEnv(string(data)))

	// Decoding config
	if err := yaml.UnmarshalStrict(data, &conf); err != nil {
		log.Error(err)
		return nil, err
	}
	return &conf, nil
}

func getRouter(conf config.File) (*mux.Router, error) {
	// Using a in memory database
	inMemoryStorage := storage.NewInMemory()

	// Adding applications
	for _, a := range conf.Apps {
		application := app.NewApplication(
			a.Name,
			a.AppID,
			a.Key,
			a.Secret,
			a.OnlySSL,
			a.Enabled,
			a.UserEvents,
			a.WebHooks.Enabled,
			a.WebHooks.URL,
		)

		if err := inMemoryStorage.AddApp(application); err != nil {
			return nil, err
		}
	}

	router := mux.NewRouter()
	router.Use(handlers.RecoveryHandler())

	router.Path("/app/{key}").Methods("GET").Handler(
		websockets.NewWebsocket(inMemoryStorage),
	)

	appsRouter := router.PathPrefix("/apps/{app_id}").Subrouter()
	appsRouter.Use(
		api.CheckAppDisabled(inMemoryStorage),
		api.Authentication(inMemoryStorage),
	)

	appsRouter.Path("/events").Methods("POST").Handler(
		api.NewPostEvents(inMemoryStorage),
	)
	appsRouter.Path("/channels").Methods("GET").Handler(
		api.NewGetChannels(inMemoryStorage),
	)
	appsRouter.Path("/channels/{channel_name}").Methods("GET").Handler(
		api.NewGetChannel(inMemoryStorage),
	)
	appsRouter.Path("/channels/{channel_name}/users").Methods("GET").Handler(
		api.NewGetChannelUsers(inMemoryStorage),
	)
	return router, nil
}
