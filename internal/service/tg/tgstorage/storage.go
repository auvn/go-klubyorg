package tgstorage

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/auvn/go-klubyorg/internal/service/tg/dataenc"
	tgstoragev1 "github.com/auvn/go-klubyorg/pkg/gen/proto/tgstorage/v1"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
)

type Storage struct {
	storageChatID int64
	bot           *tgbot.Bot
}

func NewStorage(
	storageChatID int64,
	bot *tgbot.Bot,
) *Storage {
	return &Storage{
		storageChatID: storageChatID,
		bot:           bot,
	}
}

type Receipt struct {
	MessageID int
}

func (s *Storage) Put(
	ctx context.Context,
	b []byte,
) (*Receipt, error) {
	doc := tgbot.SendDocumentParams{
		ChatID: s.storageChatID,
		Document: &models.InputFileUpload{
			Filename: fmt.Sprintf("%s.data", uuid.New()),
			Data:     bytes.NewReader(dataenc.Compress(b)),
		},
	}
	sentDoc, err := s.bot.SendDocument(ctx, &doc)
	if err != nil {
		return nil, fmt.Errorf("bot.SendMessage: %w", err)
	}

	f, err := s.bot.GetFile(ctx, &tgbot.GetFileParams{
		FileID: sentDoc.Document.FileID,
	})
	if err != nil {
		return nil, fmt.Errorf("bot.GetFile: %w", err)
	}

	md := tgstoragev1.SlotMetadata{
		StorageFileMessageId: int64(sentDoc.ID),
		StorageFile: &tgstoragev1.SlotMetadata_File{
			FileId:       f.FileID,
			FileUniqueId: f.FileUniqueID,
			FileSize:     f.FileSize,
			FilePath:     f.FilePath,
		},
	}

	msg := tgbot.SendMessageParams{
		ChatID: s.storageChatID,
		Text:   dataenc.EncodeToString(&md),
		ReplyParameters: &models.ReplyParameters{
			MessageID: sentDoc.ID,
		},
	}

	sentMsg, err := s.bot.SendMessage(ctx, &msg)
	if err != nil {
		return nil, fmt.Errorf("bot.SendMessage: %w", err)
	}

	return &Receipt{
		MessageID: sentMsg.ID,
	}, nil
}

func (s *Storage) Get(
	ctx context.Context,
	receipt *Receipt,
) ([]byte, error) {
	sentMsg, err := s.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: s.storageChatID,
		Text:   fmt.Sprintf("Dummy reply: %d", time.Now().Unix()),
		ReplyParameters: &models.ReplyParameters{
			MessageID: receipt.MessageID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("send dummy reply message: %w", err)
	}

	if sentMsg.ReplyToMessage == nil {
		return nil, fmt.Errorf("original message is unknown")
	}

	encodedMd := sentMsg.ReplyToMessage.Text

	var md tgstoragev1.SlotMetadata
	if err := dataenc.DecodeString(encodedMd, &md); err != nil {
		return nil, fmt.Errorf("decode str: %w", err)
	}

	tgfile := models.File{
		FileID:       md.StorageFile.FileId,
		FileUniqueID: md.StorageFile.FileUniqueId,
		FileSize:     md.StorageFile.FileSize,
		FilePath:     md.StorageFile.FilePath,
	}

	linkToDownload := s.bot.FileDownloadLink(&tgfile)

	resp, err := http.Get(linkToDownload)
	if err != nil {
		return nil, fmt.Errorf("fetch storage file: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non 200 status code: %s", tgfile.FilePath)
	}

	defer resp.Body.Close()

	b, err := dataenc.DecompressReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("DecompressReader: %w", err)
	}

	return b, nil
}
