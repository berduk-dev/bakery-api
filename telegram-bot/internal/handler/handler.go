package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"tgbot-bad-da-yo/internal/repo/errs"
	"tgbot-bad-da-yo/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5"
)

type adminState string

const (
	StateIdle             adminState = "idle"
	StateComposingMailing adminState = "composing_text"
	StateConfirmMailing   adminState = "confirm_mailing"
)

type mediaType string

const (
	MediaNone  mediaType = ""
	MediaPhoto mediaType = "photo"
	MediaVideo mediaType = "video"
	MediaAudio mediaType = "audio"
	MediaVoice mediaType = "voice"
)

type Handler struct {
	service service.Service
	bot     *tgbotapi.BotAPI

	adminID     int64
	developerID int64
	adminChatID int64

	mailText    string
	mailMediaID string
	mailMedia   mediaType
	adminState  adminState

	// Хранилище кодов призов для пользователей, ожидающих отправки номера
	userPrizeCodes map[int64]string
}

func New(bot *tgbotapi.BotAPI, service service.Service, adminID, developerID, adminChatID int64) Handler {
	return Handler{
		service:        service,
		bot:            bot,
		adminID:        adminID,
		developerID:    developerID,
		adminChatID:    adminChatID,
		userPrizeCodes: make(map[int64]string),
	}
}

// Start 🚀 Основной запуск бота
func (h *Handler) Start() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := h.bot.GetUpdatesChan(u)

	for update := range updates {
		switch {
		case update.Message != nil:
			h.handleMessage(update.Message)
		case update.CallbackQuery != nil:
			h.handleCallback(update.CallbackQuery)
		}
	}
	return nil
}

// 💬 Обработка обычных сообщений
func (h *Handler) handleMessage(msg *tgbotapi.Message) {
	ctx := context.Background()

	// Обработка полученного контакта
	if msg.Contact != nil {
		// Проверяем, что пользователь отправил свой контакт
		if msg.Contact.UserID != msg.From.ID {
			reply := tgbotapi.NewMessage(msg.Chat.ID, "Пожалуйста, отправьте свой номер телефона")
			_, _ = h.bot.Send(reply)
			return
		}

		// Проверяем, что у пользователя есть сохраненный код приза
		code, exists := h.userPrizeCodes[msg.From.ID]
		if !exists {
			return
		}

		// Сохраняем номер телефона
		err := h.service.UpdateUserPhone(ctx, msg.From.ID, msg.Contact.PhoneNumber)
		if err != nil {
			if errors.Is(err, errs.ErrPhoneAlreadyExists) {
				reply := tgbotapi.NewMessage(msg.Chat.ID, "Этот номер телефона уже использовался для получения приза")
				reply.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				_, _ = h.bot.Send(reply)
				delete(h.userPrizeCodes, msg.From.ID)
				return
			}

			log.Println("error service.UpdateUserPhone: ", err)
			reply := tgbotapi.NewMessage(msg.Chat.ID, "Ошибка при сохранении номера телефона")
			reply.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			_, _ = h.bot.Send(reply)
			delete(h.userPrizeCodes, msg.From.ID)
			return
		}

		// Присваиваем приз
		err = h.service.AddTelegramIdIntoPrize(ctx, msg.From.ID, code)
		if err != nil {
			log.Println("error service.AddTelegramIdIntoPrize:", err)
			reply := tgbotapi.NewMessage(msg.Chat.ID, "Ошибка при получении приза")
			reply.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			_, _ = h.bot.Send(reply)
			delete(h.userPrizeCodes, msg.From.ID)
			return
		}

		// Получаем информацию о призе
		prize, err := h.service.GetPrizeByUserID(ctx, msg.From.ID)
		if err != nil {
			log.Println("error service.GetPrizeByUserID:", err)
			delete(h.userPrizeCodes, msg.From.ID)
			return
		}

		// Отправляем сообщение о получении приза
		text := fmt.Sprintf("Приз '%s' получен! Ваш код - %s.\nПолучите свой призом с 1 по 31 января.", prize.Prize, code)
		prizeMessage := tgbotapi.NewMessage(msg.Chat.ID, text)
		prizeMessage.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		_, _ = h.bot.Send(prizeMessage)

		// Удаляем код из хранилища
		delete(h.userPrizeCodes, msg.From.ID)
		return
	}

	if msg.From.ID == h.adminID || msg.From.ID == h.developerID {
		switch {
		case msg.IsCommand() && msg.Command() == "info":
			users, err := h.service.GetUsers(ctx)
			if err != nil {
				log.Println("error service.CreateUser: ", err)
				message := tgbotapi.NewMessage(msg.Chat.ID, "Ошибка при получении информации ❌")
				message.ReplyToMessageID = msg.MessageID
				_, _ = h.bot.Send(message)
				return
			}

			// Разбиваем на части по 4000 символов (лимит Telegram - 4096)
			const maxLen = 4000
			var messages []string
			current := ""

			for i, u := range users {
				line := fmt.Sprintf("%d. ID: %d, Телефон: %d, Создан: %s\n",
					i+1,
					u.TelegramID,
					u.Phone,
					u.CreatedAt.Format("2006-01-02"),
				)
				if len(current)+len(line) > maxLen {
					messages = append(messages, current)
					current = line
				} else {
					current += line
				}
			}
			if current != "" {
				messages = append(messages, current)
			}

			// Отправляем все части
			for i, text := range messages {
				message := tgbotapi.NewMessage(msg.Chat.ID, text)
				if i == 0 {
					message.ReplyToMessageID = msg.MessageID
				}
				_, _ = h.bot.Send(message)
			}
			return

		case msg.IsCommand() && msg.Command() == "mail":
			h.adminState = StateComposingMailing

			reply := tgbotapi.NewMessage(msg.Chat.ID, "Отправьте сообщение для рассылки (текст, фото, видео или аудио):")
			_, _ = h.bot.Send(reply)
			return

		case h.adminState == StateComposingMailing:
			// Сброс предыдущих данных
			h.mailText = ""
			h.mailMediaID = ""
			h.mailMedia = MediaNone

			// Определяем тип контента
			switch {
			case msg.Photo != nil && len(msg.Photo) > 0:
				h.mailMediaID = msg.Photo[len(msg.Photo)-1].FileID
				h.mailMedia = MediaPhoto
				h.mailText = msg.Caption
			case msg.Video != nil:
				h.mailMediaID = msg.Video.FileID
				h.mailMedia = MediaVideo
				h.mailText = msg.Caption
			case msg.Audio != nil:
				h.mailMediaID = msg.Audio.FileID
				h.mailMedia = MediaAudio
				h.mailText = msg.Caption
			case msg.Voice != nil:
				h.mailMediaID = msg.Voice.FileID
				h.mailMedia = MediaVoice
			default:
				h.mailText = msg.Text
			}

			h.adminState = StateConfirmMailing

			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Да", "mail_confirm"),
					tgbotapi.NewInlineKeyboardButtonData("Нет", "mail_cancel"),
				),
			)

			reply := tgbotapi.NewMessage(msg.Chat.ID, "Отправить это сообщение всем пользователям?")
			reply.ReplyMarkup = keyboard
			_, _ = h.bot.Send(reply)
			return
		}
	}
	switch msg.Command() {
	case "start":
		code := msg.CommandArguments()
		if code == "" {
			return
		}

		err := h.service.CreateUser(ctx, msg.From.ID)
		if err != nil {
			if !errors.Is(err, errs.ErrUserAlreadyExists) {
				log.Println("error service.CreateUser: ", err)
				return
			}

			// юзер уже существует
			prize, err := h.service.GetPrizeByUserID(ctx, msg.From.ID)
			if err != nil {
				log.Println("error service.GetPrizeByUserID:", err)
				return
			}
			if prize != nil {
				prizeState := "Еще не использован"
				if prize.UsedAt != nil {
					prizeState = "Использован"
				}
				msg := tgbotapi.NewMessage(msg.From.ID, fmt.Sprintf("Ваш приз: %s\n❗Статус: %s", prize.Prize, prizeState))
				_, _ = h.bot.Send(msg)
				return
			}
		}

		// Сохраняем код для дальнейшего использования после получения номера
		h.userPrizeCodes[msg.From.ID] = code

		// Отправляем запрос на получение номера телефона
		phoneRequestBtn := tgbotapi.NewKeyboardButton("Поделиться номером телефона")
		phoneRequestBtn.RequestContact = true
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(phoneRequestBtn),
		)
		keyboard.OneTimeKeyboard = true
		keyboard.ResizeKeyboard = true

		phoneMessage := tgbotapi.NewMessage(msg.Chat.ID, "Поделитесь номером телефона для получения приза")
		phoneMessage.ReplyMarkup = keyboard
		_, _ = h.bot.Send(phoneMessage)
		return
	}

	// Только для группы администраторов
	if msg.Chat.ID != h.adminChatID {
		return
	}

	code := strings.TrimSpace(msg.Text)

	prize, err := h.service.GetPrizeByCode(ctx, code)
	if err != nil {
		// Логируем все ошибки, включая pgx. ErrNoRows
		log.Printf("error service.GetPrizeByCode for code '%s': %v", code, err)

		if errors.Is(err, pgx.ErrNoRows) {
			message := tgbotapi.NewMessage(msg.Chat.ID, "Код не найден ❌")
			message.ReplyToMessageID = msg.MessageID
			_, _ = h.bot.Send(message)
			return
		}
		return
	}

	// Проверка, привязан ли приз к телеграм айди или old_user
	isValid, err := h.service.IsValidByCode(ctx, code)
	if !isValid || err != nil {
		message := tgbotapi.NewMessage(msg.Chat.ID, "Код не привязан к телеграм айди ❌")
		message.ReplyToMessageID = msg.MessageID
		_, _ = h.bot.Send(message)
		return
	}

	var text string
	if prize.UsedAt != nil {
		text = fmt.Sprintf(
			"🎁 Приз: %s\n✅ Активирован: %s (МСК)",
			prize.Prize,
			prize.UsedAt.Format("02.01.2006 15:04"),
		)
	} else {
		text = fmt.Sprintf("🎁 Приз: %s\n❗ Код не активирован", prize.Prize)
	}

	resp := tgbotapi.NewMessage(msg.Chat.ID, text)
	resp.ReplyToMessageID = msg.MessageID

	// Добавляем кнопку, если код не активирован
	if prize.UsedAt == nil {
		btn := tgbotapi.NewInlineKeyboardButtonData("Использовать", fmt.Sprintf("activate_%s", code))
		keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btn))
		resp.ReplyMarkup = keyboard
	}

	if _, err := h.bot.Send(resp); err != nil {
		log.Printf("error bot.Send: %v", err)
	}
}

// ⚙️ Обработка нажатий на кнопки
func (h *Handler) handleCallback(cb *tgbotapi.CallbackQuery) {
	ctx := context.Background()
	data := cb.Data

	switch data {

	case "mail_confirm":
		err := h.service.Broadcast(ctx, h.mailText, h.mailMediaID, string(h.mailMedia))
		if err != nil {
			_, _ = h.bot.Send(tgbotapi.NewMessage(h.adminID, "Ошибка рассылки: "+err.Error()))
		} else {
			_, _ = h.bot.Send(tgbotapi.NewMessage(h.adminID, "Рассылка завершена."))
		}

		h.adminState = StateIdle
		h.mailText = ""
		h.mailMediaID = ""
		h.mailMedia = MediaNone
		return

	case "mail_cancel":
		h.adminState = StateIdle
		h.mailText = ""
		h.mailMediaID = ""
		h.mailMedia = MediaNone

		_, _ = h.bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, "Рассылка отменена."))
	}

	if strings.HasPrefix(data, "activate_") {
		code := strings.TrimPrefix(data, "activate_")

		err := h.service.ActivateCode(ctx, code)
		if err != nil {
			log.Printf("error ActivateCode: %v", err)
			_, _ = h.bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, "⚠️ Не удалось активировать код"))
			return
		}

		// Получаем обновлённый prize
		prize, err := h.service.GetPrizeByCode(ctx, code)
		if err != nil {
			log.Printf("error service.GetPrizeByCode for code '%s' after activation: %v", code, err)
			_, _ = h.bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, "⚠️ Ошибка при обновлении данных"))
			return
		}

		text := fmt.Sprintf("🎁 Приз: %s\n✅ Активирован: %s (МСК)", prize.Prize, prize.UsedAt.Format("02.01.2006 15:04"))

		// Обновляем текст того же сообщения
		edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text)
		_, _ = h.bot.Send(edit)
	}

	_, _ = h.bot.Request(tgbotapi.NewCallback(cb.ID, ""))
}
