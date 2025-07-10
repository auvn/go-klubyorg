package tg

import (
	"context"
	"fmt"
)

func (b *BotController) applyActions(
	ctx context.Context,
	actions ...*Actions,
) error {
	for _, a := range actions {
		if a == nil {
			continue
		}

		if a := a.EditMessage; a != nil {
			if _, err := b.me.EditMessageText(ctx, a); err != nil {
				return fmt.Errorf("edit message: %w", err)
			}

		}

		if a := a.SendMessage; a != nil {
			if _, err := b.me.SendMessage(ctx, a); err != nil {
				return fmt.Errorf("send message: %w", err)
			}
		}
	}

	return nil
}
