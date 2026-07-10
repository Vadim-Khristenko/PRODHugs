package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go-service-template/internal/models"

	tgbot "github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
	"github.com/google/uuid"
)

// maxHugCommentLen caps comment length we forward through the bot. The app
// has no hard limit on comments, but TG callback flows benefit from a sane
// upper bound; we truncate longer text rather than reject so a chatty
// /hug message still gets through.
const maxHugCommentLen = 280

// ── /hug ──────────────────────────────────────────────────────────────────
//
// Two targeting modes:
//
//	/hug @app_username             — explicit app username
//	/hug  (reply to TG message)    — target is the replied-to user
//	                                 (must have linked their Telegram)
//
// On success, shows a type picker as an inline keyboard. The picker is
// scoped to the original sender via the callback payload — anyone else
// clicking gets a polite refusal.
func (b *Bot) handleHug(ctx context.Context, msg *tgmodels.Message) {
	chatID := msg.Chat.ID
	if msg.From == nil {
		return
	}
	giver, err := b.userRepo.GetByTelegramID(ctx, msg.From.ID)
	if err != nil {
		b.reply(ctx, chatID, "Чтобы отправлять обнимашки, привяжите аккаунт через настройки на сайте.")
		return
	}

	target, usedReply, resolved := b.resolveHugTargetWithMode(ctx, msg)
	if !resolved {
		b.reply(ctx, chatID, "Кого обнять? Напишите <code>/hug @username</code> или ответьте этой командой на сообщение нужного пользователя.")
		return
	}
	if target.ID == giver.ID {
		b.reply(ctx, chatID, "Себя обнимать тоже полезно, но это нужно делать руками. Я тут не помощница.")
		return
	}

	if comment := extractHugComment(msg.Text, usedReply); comment != "" {
		isGroup := msg.Chat.Type != "private"
		b.stashPendingComment(msg.From.ID, target.ID, pendingHugComment{
			comment:    comment,
			sourceChat: chatID,
			isGroup:    isGroup,
			createdAt:  time.Now(),
		})
	}

	keyboard := hugTypePickerKeyboard(msg.From.ID, target.ID)
	prompt := fmt.Sprintf("Выберите тип обнимашки для <b>%s</b>:", htmlEscape(displayName(target)))
	if !b.enabled {
		_ = b.client.SendMessage(chatID, prompt)
		return
	}
	if _, err := b.tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        prompt,
		ParseMode:   tgmodels.ParseModeHTML,
		ReplyMarkup: keyboard,
	}); err != nil {
		b.logger.Error("telegram bot: /hug picker send failed", "error", err)
	}
}

// resolveHugTargetWithMode resolves /hug's target and reports whether the
// reply-to path was used (vs explicit @username). Knowing the mode lets us
// strip the right prefix when extracting the comment portion.
func (b *Bot) resolveHugTargetWithMode(ctx context.Context, msg *tgmodels.Message) (*models.User, bool, bool) {
	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil {
		if u, err := b.userRepo.GetByTelegramID(ctx, msg.ReplyToMessage.From.ID); err == nil {
			return u, true, true
		}
	}
	parts := strings.Fields(msg.Text)
	if len(parts) >= 2 {
		username := strings.TrimPrefix(parts[1], "@")
		if username != "" {
			if u, err := b.userRepo.GetByUsername(ctx, username); err == nil {
				return u, false, true
			}
		}
	}
	return nil, false, false
}

// extractHugComment returns the trimmed comment portion of a /hug command.
//
//	/hug @user comment...   -> "comment..."    (usedReply=false)
//	/hug comment... (reply) -> "comment..."    (usedReply=true)
//
// The "/hug" token itself may carry an @botname suffix in groups; we strip
// the first whitespace-delimited token wholesale to avoid having to parse it.
func extractHugComment(text string, usedReply bool) string {
	s := strings.TrimSpace(text)
	// Drop the "/hug" (or "/hug@bot") token.
	if i := strings.IndexAny(s, " \t"); i > 0 {
		s = strings.TrimSpace(s[i:])
	} else {
		return ""
	}
	if !usedReply {
		// Drop the "@username" token.
		if i := strings.IndexAny(s, " \t"); i > 0 {
			s = strings.TrimSpace(s[i:])
		} else {
			return ""
		}
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) > maxHugCommentLen {
		// Truncate by rune, not byte — comments are user text and are
		// frequently Cyrillic, where len(s) and rune count diverge.
		r := []rune(s)
		s = string(r[:maxHugCommentLen])
	}
	return s
}

// hugTypePickerKeyboard returns the inline keyboard for the /hug picker.
// Callback payloads use the unified action-token format
// "hug.pick:<type>:<initiator_tg_id>:<target_user_id>" so the callback
// dispatcher (parseAction) routes them alongside notification buttons. The
// initiator id lets us verify it's the original sender clicking — Telegram
// groups happily pass anyone's clicks back to us.
func hugTypePickerKeyboard(initiatorTGID int64, targetUserID uuid.UUID) *tgmodels.InlineKeyboardMarkup {
	mk := func(label, hugType string) tgmodels.InlineKeyboardButton {
		return tgmodels.InlineKeyboardButton{
			Text:         label,
			CallbackData: fmt.Sprintf("hug.pick:%s:%d:%s", hugType, initiatorTGID, targetUserID.String()),
		}
	}
	return &tgmodels.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgmodels.InlineKeyboardButton{
			{mk("Обычная", "standard"), mk("Медвежья", "bear")},
			{mk("Тёплая", "warm"), mk("Душевная", "soul")},
			{mk("Групповая", "group")},
		},
	}
}

// handleHugPickCallback fires when someone clicks a type-picker button.
// The parseAction arg is "<hug_type>:<initiator_tg_id>:<target_user_id>".
func (b *Bot) handleHugPickCallback(ctx context.Context, cb *tgmodels.CallbackQuery, payload string) {
	parts := strings.SplitN(payload, ":", 3)
	if len(parts) != 3 {
		b.answerCallback(ctx, cb.ID, "Не разобрала кнопку, попробуйте ещё раз.")
		return
	}
	hugType := parts[0]
	initiatorTG, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		b.answerCallback(ctx, cb.ID, "Не разобрала кнопку.")
		return
	}
	targetID, err := uuid.Parse(parts[2])
	if err != nil {
		b.answerCallback(ctx, cb.ID, "Не разобрала кнопку.")
		return
	}

	if cb.From.ID != initiatorTG {
		b.answerCallback(ctx, cb.ID, "Эта обнимашка не для вас — пусть выбирает тот, кто её предложил.")
		return
	}

	giver, err := b.userRepo.GetByTelegramID(ctx, cb.From.ID)
	if err != nil {
		b.answerCallback(ctx, cb.ID, "Ваш Telegram пока не привязан.")
		return
	}

	// Pull the pending comment (if any) that was stashed when /hug was typed,
	// and check whether the recipient can receive a PM. If the comment came
	// from a group chat and the recipient hasn't opened a DM with us yet,
	// the comment would silently vanish — degrade by dropping the comment
	// and nudging the group to ask the user to /start.
	pending, hadComment := b.popPendingComment(cb.From.ID, targetID)
	var commentPtr *string
	deliverable := true
	if hadComment {
		targetUser, err := b.userRepo.GetByID(ctx, targetID)
		if err != nil {
			b.logger.Warn("telegram bot: /hug get target failed", "target_id", targetID, "error", err)
			deliverable = false
		} else {
			deliverable = b.canReachPM(ctx, targetUser)
		}
		if deliverable {
			c := pending.comment
			commentPtr = &c
		}
	}

	hug, target, err := b.hugSvc.SuggestHug(ctx, giver.ID, targetID, hugType, commentPtr, nil)
	if err != nil {
		b.logger.Error("telegram bot: /hug suggest failed", "error", err)
		b.answerCallback(ctx, cb.ID, "Не получилось отправить: "+friendlyError(err))
		return
	}
	b.answerCallback(ctx, cb.ID, "Отправила!")

	// If the comment couldn't be delivered (no DM yet / blocked) and the
	// /hug came from a group, post a single nudge in that group so the
	// sender knows their text didn't reach the recipient.
	if hadComment && !deliverable && pending.isGroup {
		nudge := fmt.Sprintf(
			"Объятие для <b>%s</b> ушло, но комментарий не передала — у получателя ещё не открыт диалог со мной. Попросите написать мне <code>/start</code>, и в следующий раз комментарий дойдёт.",
			htmlEscape(displayName(target)),
		)
		b.reply(ctx, pending.sourceChat, nudge)
	}

	// Rewrite the picker message into a confirmation — the keyboard is
	// removed so the buttons can't be re-clicked. cb.Message is
	// MaybeInaccessibleMessage (a value, not a pointer in this lib);
	// the Message field is what we read on a fresh callback.
	if cb.Message.Message != nil {
		text := fmt.Sprintf("🤗 Обнимашка для <b>%s</b> отправлена. Тип: <b>%s</b>.",
			htmlEscape(displayName(target)), htmlEscape(hugTypeShortLabel(hugType)))
		_, err := b.tg.EditMessageText(ctx, &tgbot.EditMessageTextParams{
			ChatID:      cb.Message.Message.Chat.ID,
			MessageID:   cb.Message.Message.ID,
			Text:        text,
			ParseMode:   tgmodels.ParseModeHTML,
			ReplyMarkup: &tgmodels.InlineKeyboardMarkup{InlineKeyboard: [][]tgmodels.InlineKeyboardButton{}},
		})
		if err != nil {
			b.logger.Warn("telegram bot: edit hug picker failed", "error", err)
		}
	}
	_ = hug
}
