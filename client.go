package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/pymq/tfahack/db"
	"github.com/pymq/tfahack/models"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/utf8string"
	"gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
)

type Bot struct {
	client *telebot.Bot
	poller *telebot.LongPoller
	db     *db.DB
	cfg    Config
}

func NewBot(cfg Config, db *db.DB) (*Bot, error) {
	poller := &telebot.LongPoller{Timeout: 10 * time.Second}
	b, err := telebot.NewBot(telebot.Settings{
		OnError: func(err error, ctx telebot.Context) {
			log.Errorf("telegram bot: %s", err)
		},
		Token:  cfg.APIToken,
		Poller: poller,
	})
	if err != nil {
		return nil, fmt.Errorf("create client: %v", err)
	}

	bot := &Bot{
		client: b,
		poller: poller,
		cfg:    cfg,
		db:     db,
	}
	if cfg.LogAllEvents {
		b.Use(middleware.Logger())
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
	adminsOnly := b.client.Group()
	adminsOnly.Use(IgnoreNonPrivateMessages)
	if len(b.cfg.AdminIDs) > 0 {
		// TODO: reply to user that he can't use this command
		adminsOnly.Use(middleware.Whitelist(b.cfg.AdminIDs...))
	}

	b.client.Handle("/start", b.handleStart, IgnoreNonPrivateMessages)
	b.client.Handle("/help", b.handleHelp, IgnoreNonPrivateMessages)
	adminsOnly.Handle("/create_mailing_list", b.handleCreateMailingList)
	adminsOnly.Handle("/send_messages", b.handleSendMessages)
	adminsOnly.Handle("/show_replies", b.handleShowReplies)
	adminsOnly.Handle("/notifications_config", b.handleNotificationsConfig)
	adminsOnly.Handle("/topics_stats", b.handleTopicsStats)
	// rest text messages
	b.client.Handle(telebot.OnText, func(ctx telebot.Context) error {
		msg := ctx.Message()
		if reply := msg.ReplyTo; reply != nil {
			// TODO: check if he replied on message to recipient; save it to DB; send it to sender
		}
		return ctx.Send("Unknown command")
	})

	err := b.client.SetCommands([]telebot.Command{
		{
			Text:        "start",
			Description: "запустить бота. TODO: description",
		},
		{
			Text:        "help",
			Description: "запустить бота. TODO: description",
		},
		{
			Text:        "create_mailing_list",
			Description: "создать список людей на рассылку. формат: /create_mailing_list <mailing_list_name> <recipient1> <recipient2> <...>",
		},
		{
			Text:        "send_messages",
			Description: "отправить рассылку по указанному топику и списку рассылки. формат: /send_messages <topic> <mailing_list_name>",
		},
		{
			Text:        "show_replies",
			Description: "вывести ответы по топику. TODO: description",
		},
		{
			Text:        "notifications_config",
			Description: "настройка уведомлений. TODO: descriptions",
		},
		{
			Text:        "topics_stats",
			Description: "вывод статистики по топикам. TODO: descriptions",
		},
	})

	return err
}

func (b *Bot) handleStart(ctx telebot.Context) error {
	recipients, err := b.db.GetRecipientsByIds([]int64{ctx.Chat().ID})
	if err != nil {
		log.Errorf("stat command: recipients select: %v", err)
		return err
	}
	if len(recipients) > 0 {
		return ctx.Send("Вы уже в списке, как только для вас будет сообщение мы вам напишем!")
	}
	err = b.db.AddRecipient(models.Recipient{
		RecipientName:   fmt.Sprintf("%s %s", ctx.Chat().FirstName, ctx.Chat().LastName),
		RecipientTGId:   ctx.Chat().ID,
		RecipientTGName: ctx.Chat().Username,
	})
	if err != nil {
		log.Errorf("stat command: recipient insert: %v", err)
		return err
	}
	return ctx.Send("Рады видеть вас в нашем боте! Теперь вы сможете получать рассылки от партнеров!")
}

func (b *Bot) handleHelp(ctx telebot.Context) error {
	// TODO: write help text
	return ctx.Send("Hello!")
}

// command: /create_mailing_list <mailing_list_name> <recipient1> <recipient2> <...>
func (b *Bot) handleCreateMailingList(ctx telebot.Context) error {
	args := ctx.Args()
	if len(args) < 2 {
		return ctx.Send("Пожалуйста, введите данные в формате /create_mailing_list <Название_списка> <Получатель1> <Получатель2> <...>")
	}
	recipients := args[1:]

	uniqueRecipients := make(map[string]struct{})
	errors := make([]string, 0)
	n := 0
	for _, recipient := range args[1:] {
		recipient = strings.TrimPrefix(recipient, "@")
		if _, ok := uniqueRecipients[recipient]; ok {
			continue
		}
		uniqueRecipients[recipient] = struct{}{}
		recipients[n] = recipient
		n++
	}
	recipients = recipients[:n]
	recipientsInfo, err := b.db.GetRecipientsByTGNames(recipients)
	if err != nil {
		log.Errorf("create mailing list: load recipients: %v", err)
		return err
	}

	recipientsIds := make([]int64, len(recipientsInfo))
	for id, recipient := range recipientsInfo {
		recipientsIds[id] = recipient.RecipientId
		delete(uniqueRecipients, recipient.RecipientTGName)
	}

	for recipient := range uniqueRecipients {
		errors = append(errors, fmt.Sprintf("@%s - Пользователь не подключен к боту", recipient))
	}

	err = b.db.AddMailingList(models.MailingList{ListName: args[0], SenderTGId: ctx.Chat().ID}, recipientsIds)
	if err != nil {
		log.Errorf("create mailing list: %v", err)
		return err
	}

	if len(errors) > 0 {
		return ctx.Send(fmt.Sprintf("Список создан!\n\nНе удалось добавить некоторых пользователей:\n%s", strings.Join(errors, ",\n")))
	}
	return ctx.Send("Список создан!")
}

// command: /send_messages <topic> <mailing_list_name>
func (b *Bot) handleSendMessages(ctx telebot.Context) error {
	args := ctx.Args()
	if len(args) != 2 {
		return ctx.Send("command should be in format /send_messages <topic> <mailing_list_name>")
	}
	topic := args[0]
	mailingListName := args[1]
	// TODO: send, save to db

	return ctx.Send("test " + topic + " " + mailingListName)
}

func (b *Bot) handleShowReplies(ctx telebot.Context) error {
	// TODO: show as one message with inline buttons as pagination
	return ctx.Send("test")
}

func (b *Bot) handleNotificationsConfig(ctx telebot.Context) error {
	// TODO: show keyboard buttons with 3 config options; add handlers on each of the buttons;
	//  hide keyboard after tapping on button
	return ctx.Send("test")
}

func (b *Bot) handleTopicsStats(ctx telebot.Context) error {
	// TODO: show table with stats?
	return ctx.Send("test")
}

func IgnoreNonPrivateMessages(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(ctx telebot.Context) error {
		msg := ctx.Message()
		if msg != nil && !msg.Private() {
			return nil
		}

		return next(ctx)
	}
}
