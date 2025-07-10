package tg

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/auvn/go-klubyorg/internal/service/tg/dataenc"
	"github.com/auvn/go-klubyorg/internal/service/tg/tgstorage"
	tgbotv1 "github.com/auvn/go-klubyorg/pkg/gen/proto/tgbot/v1"
	"github.com/go-telegram/bot/models"
	"google.golang.org/protobuf/proto"
)

func messageTextWithState(prefix string, state proto.Message) (string, models.ParseMode) {
	return fmt.Sprintf(
		"%s\n||%s||",
		EscapeMarkdownV2(prefix),
		EscapeMarkdownV2(
			dataenc.EncodeToString(state),
		),
	), models.ParseModeMarkdown
}

func extractEntityText(
	text string,
	e models.MessageEntity,
) (string, bool) {
	runes := []rune(text)
	start := utf16OffsetToRuneIndex(runes, e.Offset)
	end := utf16OffsetToRuneIndex(runes, e.Offset+e.Length)
	if start < 0 || end > len(runes) {
		return "", false
	}

	return string(runes[start:end]), true
}

func utf16OffsetToRuneIndex(
	runes []rune,
	utf16Offset int,
) int {
	count := 0
	for i, r := range runes {
		if r >= 0x10000 {
			count += 2
		} else {
			count += 1
		}
		if count > utf16Offset {
			return i
		}
	}
	return len(runes)
}

func (b *BotController) getState(
	ctx context.Context,
	msg *models.Message,
) (*tgbotv1.State, error) {
	for _, e := range msg.Entities {
		if e.Type == models.MessageEntityTypeSpoiler {
			encodedMd, ok := extractEntityText(msg.Text, e)
			if !ok {
				slog.Warn("invalid entity", "entity", e, "text", msg.Text)
				continue
			}

			var state tgbotv1.State
			if err := dataenc.DecodeString(encodedMd, &state); err != nil {
				return nil, fmt.Errorf("decode state: %w", err)
			}

			return &state, nil
		}
	}

	return nil, nil
}

func (b *BotController) loadState(
	ctx context.Context,
	fileReceipt *tgbotv1.StorageMetadata_FileReceipt,
) (*tgbotv1.State, error) {
	encodedState, err := b.storage.Get(ctx, &tgstorage.Receipt{
		MessageID: int(fileReceipt.GetMessageId()),
	})
	if err != nil {
		return nil, fmt.Errorf("get encoded state: %w", err)
	}

	var state tgbotv1.State
	if err := dataenc.Decode(encodedState, &state); err != nil {
		return nil, fmt.Errorf("decode state: %w", err)
	}

	return &state, nil
}
