package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
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
			message, err := b.db.GetMessage(int64(msg.ReplyTo.ID))
			if err != nil {
				log.Errorf("reply: get message: %v", err)
				return err
			}
			if message.ListId == 0 { //TODO add relevant check on reply to to unsaved message
				return nil
			}

			if message.IsRecipientMessage == 1 {
				// sentMessage, err := b.client.Send(telebot.ChatID(message.SenderTGId), msg.Text)
			} else {
				sentMessage, err := b.client.Send(telebot.ChatID(message.SenderTGId), msg.Text)
				if err != nil {
					log.Errorf("reply: send recipient reply: %v", err)
					return err
				}
				err = b.db.AddMessage(models.Message{
					MessageTGId:        int64(sentMessage.ID),
					SenderTGId:         message.SenderTGId,
					RecipientId:        ctx.Chat().ID,
					TopicId:            message.TopicId,
					ListId:             message.ListId,
					SendDateTime:       time.Time{},
					Message:            msg.Text,
					Read:               0,
					IsRecipientMessage: 1,
				})
				if err != nil {
					log.Errorf("reply: save recipient reply: %v", err)
					return err
				}
			}
		}
		return nil
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
			Description: "вывести ответы по топику. /show_replies <topic> [search_query_word]",
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
		log.Errorf("start command: recipient insert: %v", err)
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
	if len(args) != 3 {
		return ctx.Send("Пожалуйста, введите данные в формате /send_messages <IdТопика> <MailingListId> <MessageBody>")
	}
	topicName := args[0]
	mailingListId, _ := strconv.ParseInt(args[1], 10, 64)
	messageBody := args[2]

	topic, err := b.db.AddTopic(models.Topic{
		SenderTGId: ctx.Chat().ID,
		Topic:      topicName,
	})
	if err != nil {
		log.Errorf("send message: create topic: %v", err)
		return err
	}

	recipients, _ := b.db.GetMailingListRecipientsById(mailingListId)

	for _, recipient := range recipients {
		message, err := b.client.Send(telebot.ChatID(recipient.RecipientTGId), messageBody)
		if err != nil {
			log.Errorf("send message: %v", err)
			return err
		}
		err = b.db.AddMessage(models.Message{
			MessageTGId:        int64(message.ID),
			SenderTGId:         ctx.Chat().ID,
			RecipientId:        recipient.RecipientId,
			TopicId:            topic.TopicId,
			ListId:             mailingListId,
			SendDateTime:       message.Time(),
			Message:            message.Text,
			React:              "",
			Read:               0,
			IsRecipientMessage: 0,
		})
		if err != nil {
			log.Errorf("save message: %v", err)
			return err
		}
	}

	return ctx.Send("Пост отправлен!")
}

// command: /show_replies <topic> [search_query_word]
func (b *Bot) handleShowReplies(ctx telebot.Context) error {
	args := ctx.Args()
	if len(args) < 1 || len(args) > 2 {
		return ctx.Send("command should be in format /show_replies <topic> [search_query_word]")
	}
	topic := args[0]
	searchQuery := ""
	if len(args) == 2 {
		searchQuery = args[1]
	}

	page := 1
	const pagingBy = 5
	var allReplies []mockMessage
	var totalPages int
	tgMessages := make([]*telebot.Message, 0, pagingBy)

	sendOrUpdateMessages := func() error {
		allReplies = getMockRepliesByTopic(topic)
		if searchQuery != "" {
			n := 0
			for _, msg := range allReplies {
				if strings.Contains(msg.Text, searchQuery) {
					allReplies[n] = msg
					n++
				}
			}
			allReplies = allReplies[:n]
		}
		totalPages = len(allReplies) / pagingBy
		if len(allReplies)%pagingBy != 0 {
			totalPages++
		}

		currIdx := 0
		for i := (page - 1) * pagingBy; i < page*pagingBy && i < len(allReplies); i++ {
			reply := allReplies[i]
			const timeLayout = "2006-01-02 15:04:05"
			messageText := fmt.Sprintf("%s (%s):\n\n%s", reply.TGSenderId, reply.SendDate.Format(timeLayout), reply.Text)
			if currIdx >= len(tgMessages) {
				message, err := b.client.Send(ctx.Recipient(), messageText)
				if err != nil {
					return err
				}
				tgMessages = append(tgMessages, message)
			} else {
				message := tgMessages[currIdx]
				message, err := b.client.Edit(message, messageText)
				if err != nil {
					return err
				}
			}
			currIdx++
		}
		// clear messages on last page
		for _, message := range tgMessages[currIdx:] {
			_, err := b.client.Edit(message, "-")
			if err != nil {
				return err
			}
		}
		return nil
	}

	h := xxhash.New()
	_ = binary.Write(h, binary.BigEndian, ctx.Chat().ID)
	sumBytes := h.Sum([]byte(topic))
	uniquePrefix := base64.URLEncoding.EncodeToString(sumBytes)
	uniquePrefix = strings.TrimSuffix(uniquePrefix, "==")

	makeReplyMarkup := func() *telebot.ReplyMarkup {
		prevPage := page - 1
		if prevPage <= 0 {
			prevPage = 1
		}
		nextPage := page + 1
		if nextPage > totalPages {
			nextPage = totalPages
		}
		var replyMarkup = &telebot.ReplyMarkup{}
		var btnFirst = replyMarkup.Data("«1", uniquePrefix+"_first", "1")
		var btnPrev = replyMarkup.Data(fmt.Sprintf("< %d", prevPage), uniquePrefix+"_prev", strconv.Itoa(prevPage))
		if prevPage == page {
			btnPrev = replyMarkup.Data("-", uniquePrefix+"_prev")
		}
		var btnCurr = replyMarkup.Data(fmt.Sprintf("· %d ·", page), uniquePrefix+"_curr")
		var btnNext = replyMarkup.Data(fmt.Sprintf("%d >", nextPage), uniquePrefix+"_next", strconv.Itoa(nextPage))
		if nextPage == page {
			btnNext = replyMarkup.Data("-", uniquePrefix+"_next")
		}
		var btnLast = replyMarkup.Data(fmt.Sprintf("%d »", totalPages), uniquePrefix+"_last", strconv.Itoa(totalPages))
		replyMarkup.Inline(replyMarkup.Row(btnFirst, btnPrev, btnCurr, btnNext, btnLast))

		return replyMarkup
	}

	err := sendOrUpdateMessages()
	if err != nil {
		return err
	}

	str := fmt.Sprintf("Сообщения по топику '%s'. Выбери страницу", topic)
	keyboardMessage, err := b.client.Send(ctx.Recipient(), str, makeReplyMarkup())
	if err != nil {
		return err
	}

	for _, btn := range makeReplyMarkup().InlineKeyboard[0] {
		b.client.Handle(btn.CallbackUnique(), func(ctx telebot.Context) error {
			data := strings.Split(ctx.Callback().Data, "|")
			if len(data) == 0 {
				return ctx.Respond()
			}
			newPage, err := strconv.Atoi(data[0])
			if err != nil {
				log.Errorf("invalid data in inline keyboard callback: '%s'", ctx.Callback().Data)
				return ctx.Respond()
			}

			// this will re-calculate totalPages
			_ = sendOrUpdateMessages()

			page = newPage
			if page > totalPages {
				page = totalPages
			}

			_ = sendOrUpdateMessages()
			_, _ = b.client.EditReplyMarkup(keyboardMessage, makeReplyMarkup())

			return ctx.Respond()
		})
	}

	return nil
}

func (b *Bot) handleNotificationsConfig(ctx telebot.Context) error {
	// TODO: show keyboard buttons with 3 config options; add handlers on each of the buttons;
	//  hide keyboard after tapping on button
	return ctx.Send("test")
}

// TODO: untested
func (b *Bot) handleTopicsStats(ctx telebot.Context) error {
	topics, err := b.db.GetUserTopicsBySender(ctx.Chat().ID)
	if err != nil {
		return err
	}

	type Stats struct {
		Sent, Received int
	}

	topicsStats := make(map[string]Stats)
	for _, topic := range topics {
		messages, err := b.db.GetMessagesByTopicId(topic.TopicId)
		if err != nil {
			return err
		}
		stat := Stats{}
		for _, msg := range messages {
			if msg.IsRecipientMessage == 0 {
				stat.Sent++
			} else {
				stat.Received++
			}
		}
		topicsStats[topic.Topic] = stat
	}

	sort.Slice(topics, func(i, j int) bool {
		return topics[i].Topic < topics[j].Topic
	})

	str := new(strings.Builder)
	str.WriteString("Статистика по топикам:")
	for _, topic := range topics {
		stat := topicsStats[topic.Topic]
		fmt.Fprintf(str, "\n%s: отправлено %d; получено %d", topic.Topic, stat.Sent, stat.Received)
	}

	return ctx.Send(str.String())
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

type mockMessage struct {
	TGSenderId string
	SendDate   time.Time
	Text       string
}

// TODO: select from DB
func getMockRepliesByTopic(topic string) []mockMessage {
	return []mockMessage{
		{
			TGSenderId: "@risinglight",
			SendDate:   time.Now().Add(time.Minute * -5),
			Text:       "Test from leshya 1",
		},
		{
			TGSenderId: "@risinglight",
			SendDate:   time.Now().Add(time.Minute * -5),
			Text:       "Test from leshya 2",
		},
		{
			TGSenderId: "@risinglight",
			SendDate:   time.Now().Add(time.Minute * -4),
			Text:       "Test from leshya 3",
		},
		{
			TGSenderId: "@grishanya_win",
			SendDate:   time.Now().Add(time.Minute * -4),
			Text:       "Test from non-leshya 1",
		},
		{
			TGSenderId: "@risinglight",
			SendDate:   time.Now().Add(time.Minute * -3),
			Text:       "Test from leshya 4",
		},
		{
			TGSenderId: "@grishanya_win",
			SendDate:   time.Now().Add(time.Minute * -1),
			Text:       "Test from non-leshya 2",
		},
	}
}
