package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	APIToken string
}

func main() {
	cfg := Config{
		// TODO from file?
		APIToken: "",
	}

	bot, err := NewBot(cfg.APIToken)
	if err != nil {
		log.Panicf("init bot: %v", err)
	}

	quitCh := make(chan os.Signal, 1)
	signal.Notify(quitCh, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-quitCh
	log.Infof("received exit signal '%s'", sig)

	bot.Close()
}
