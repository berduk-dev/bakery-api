package service

import (
	"context"
	"errors"
	"fmt"
	"tgbot-bad-da-yo/internal/repo"
	"tgbot-bad-da-yo/internal/repo/errs"
	"tgbot-bad-da-yo/model"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Service struct {
	repo      repo.Repository
	bot       *tgbotapi.BotAPI
	rateLimit time.Duration
}

func New(repo repo.Repository, bot *tgbotapi.BotAPI) Service {
	return Service{
		repo:      repo,
		bot:       bot,
		rateLimit: 50 * time.Millisecond,
	}
}

func (s *Service) CreateUser(ctx context.Context, userID int64) error {
	err := s.repo.CreateUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("error repo.CreateUser: %w", err)
	}

	return nil
}

func (s *Service) GetPrizeByUserID(ctx context.Context, userID int64) (*model.Prize, error) {
	prize, err := s.repo.GetPrizeByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error repo.GetPrizeByCode: %w", err)
	}
	return prize, nil
}

func (s *Service) GetPrizeByCode(ctx context.Context, code string) (model.Prize, error) {
	prize, err := s.repo.GetPrizeByCode(ctx, code)
	if err != nil {
		return model.Prize{}, fmt.Errorf("error repo.GetPrizeByCode: %w", err)
	}
	return prize, nil
}

func (s *Service) ActivateCode(ctx context.Context, code string) error {
	err := s.repo.ActivateCode(ctx, code)
	if err != nil {
		return fmt.Errorf("error repo.ActivateCode: %w", err)
	}

	return nil
}

func (s *Service) GetTelegramIDs(ctx context.Context) ([]int64, error) {
	telegramIDs, err := s.repo.GetTelegramIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("error repo.GetTelegramIDs: %w", err)
	}

	return telegramIDs, nil
}

func (s *Service) Broadcast(ctx context.Context, text, mediaID, mediaType string) error {
	ids, err := s.repo.GetTelegramIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to load telegram ids: %w", err)
	}

	for _, id := range ids {
		var msg tgbotapi.Chattable

		switch mediaType {
		case "photo":
			photo := tgbotapi.NewPhoto(id, tgbotapi.FileID(mediaID))
			photo.Caption = text
			msg = photo
		case "video":
			video := tgbotapi.NewVideo(id, tgbotapi.FileID(mediaID))
			video.Caption = text
			msg = video
		case "audio":
			audio := tgbotapi.NewAudio(id, tgbotapi.FileID(mediaID))
			audio.Caption = text
			msg = audio
		case "voice":
			voice := tgbotapi.NewVoice(id, tgbotapi.FileID(mediaID))
			msg = voice
		default:
			msg = tgbotapi.NewMessage(id, text)
		}

		if _, err := s.bot.Send(msg); err != nil {
			fmt.Printf("failed to send message to %d: %v\n", id, err)
		}

		time.Sleep(s.rateLimit)
	}

	return nil
}

func (s *Service) AddTelegramIdIntoPrize(ctx context.Context, telegramID int64, code string) error {
	err := s.repo.AddTelegramIdIntoPrize(ctx, telegramID, code)
	switch {
	case errors.Is(err, errs.ErrTelegramIDAlreadySet):
		return fmt.Errorf("юзер уже получил этот приз")

	case errors.Is(err, errs.ErrPrizeNotFound):
		return fmt.Errorf("приз с таким кодом не найден")

	case err != nil:
		return err
	}

	return nil
}

func (s *Service) IsValidByCode(ctx context.Context, code string) (bool, error) {
	_, err := s.repo.IsValidByCode(ctx, code)
	if err != nil {
		return false, fmt.Errorf("error repo.IsValidByCode: %w", err)
	}

	return true, nil
}

func (s *Service) GetUsers(ctx context.Context) ([]model.User, error) {
	users, err := s.repo.GetUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("error repo.GetUsers: %w", err)
	}

	return users, nil
}

func (s *Service) UpdateUserPhone(ctx context.Context, userID int64, phone string) error {
	err := s.repo.UpdateUserPhone(ctx, userID, phone)
	if err != nil {
		return fmt.Errorf("error repo.UpdateUserPhone: %w", err)
	}

	return nil
}
