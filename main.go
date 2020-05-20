package main

import (
	"os"

	"github.com/morentharia/ssrf-sheriff/handler"
	log "github.com/sirupsen/logrus"
	"go.uber.org/fx"
)

func init() {
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetFormatter(&log.TextFormatter{PadLevelText: true})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func main() {
	fx.New(opts()).Run()

}

func opts() fx.Option {
	return fx.Options(
		fx.Provide(
			handler.NewLogger,
			handler.NewConfigProvider,
			handler.NewSlackClient,
			handler.NewSSRFSheriffRouter,
			handler.NewServerRouter,
			handler.NewHTTPServer,
		),
		fx.Invoke(handler.StartServer),
	)
}
