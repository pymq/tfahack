package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	APIToken     string
	AdminIDs     []int64
	LogAllEvents bool
}

func main() {
	cfg := Config{
		// TODO from file?
		AdminIDs:     nil,
		APIToken:     "",
		LogAllEvents: true,
	}

	bot, err := NewBot(cfg)
	if err != nil {
		log.Panicf("init bot: %v", err)
	}

	quitCh := make(chan os.Signal, 1)
	signal.Notify(quitCh, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-quitCh
	log.Infof("received exit signal '%s'", sig)

	bot.Close()
}
