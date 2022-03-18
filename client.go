package main

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/utf8string"
	"gopkg.in/tucnak/telebot.v2"
)

type Bot struct {
	client *telebot.Bot
	poller *telebot.LongPoller
}

func NewBot(apiToken string) (*Bot, error) {
	poller := &telebot.LongPoller{Timeout: 10 * time.Second}
	b, err := telebot.NewBot(telebot.Settings{
		Reporter: func(err error) {
			errStr := err.Error()
			// TODO: upstream
			if strings.Contains(errStr, apiToken) {
				return
			}
			log.Errorf("telegram bot: %s", errStr)
		},
		Token:  apiToken,
		Poller: poller,
	})
	if err != nil {
		return nil, fmt.Errorf("create client: %v", err)
	}

	bot := &Bot{
		client: b,
		poller: poller,
	}
	err = bot.initHandlers()
	if err != nil {
		return nil, fmt.Errorf("init handlers: %v", err)
	}
	go b.Start()

	return bot, nil
}

// TODO: remove?
func (b *Bot) SendLongMessageInParts(to telebot.Recipient, message string, silent bool) error {
	const maxMessageSize = 4096
	utf8Msg := utf8string.NewString(message)
	startIndex := 0
	opts := &telebot.SendOptions{DisableNotification: silent, DisableWebPagePreview: true}

	for {
		if startIndex >= utf8Msg.RuneCount() {
			break
		}
		lastIndex := startIndex + maxMessageSize
		if lastIndex > utf8Msg.RuneCount() {
			lastIndex = utf8Msg.RuneCount()
		}
		str := utf8Msg.Slice(startIndex, lastIndex)

		_, err := b.client.Send(to, str, opts)
		if err != nil {
			return err
		}
		startIndex = lastIndex
	}

	return nil
}

func (b *Bot) Close() {
	b.client.Stop()
	// TODO: save b.poller.LastUpdateID and set on restart ?
}

func (b *Bot) initHandlers() error {
	b.client.Handle("/start", func(m *telebot.Message) {
		_, _ = b.client.Send(m.Chat, "Hello!")
	})
	b.client.Handle("/send_messages", func(m *telebot.Message) {
		_, _ = b.client.Send(m.Chat, "test test")
		//if m.Chat.ID != b.adminChat.ID {
		//	_, _ = b.client.Send(m.Chat, "Unknown command")
		//	return
		//}
	})
	// rest text messages
	b.client.Handle(telebot.OnText, func(m *telebot.Message) {
		_, _ = b.client.Send(m.Chat, "Unknown command")
	})

	err := b.client.SetCommands([]telebot.Command{
		{
			Text:        "start",
			Description: "запустить бота. TODO: description",
		},
		{
			Text:        "send_messages",
			Description: "отправить рассылку по указанному топику",
		},
	})

	return err
}
