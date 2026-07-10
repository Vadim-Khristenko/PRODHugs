package telegram

import (
	"context"
	"fmt"
	"strings"

	"go-service-template/internal/models"

	tgmodels "github.com/go-telegram/bot/models"
	"github.com/google/uuid"
)

// ── /announce / /unannounce (admin) ───────────────────────────────────────
//
// /announce <text>          — create a platform announcement (deactivates
//
//	any previous active one)
//
// /unannounce <id>          — deactivate a specific announcement by its UUID
//
// Both require an admin caller with linked Telegram.
func (b *Bot) handleAnnounce(ctx context.Context, msg *tgmodels.Message) {
	chatID := msg.Chat.ID
	caller, ok := b.requireAdmin(ctx, chatID)
	if !ok {
		return
	}
	if b.announceSvc == nil {
		b.reply(ctx, chatID, "Сервис объявлений не подключён, скажите бекенду.")
		return
	}

	body := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/announce"))
	body = strings.TrimSpace(body)
	if body == "" {
		b.reply(ctx, chatID, "Что писать? <code>/announce ваш текст</code>. После создания старое активное объявление автоматически закроется.")
		return
	}

	ann, err := b.announceSvc.CreateAnnouncement(ctx, caller.ID, body)
	if err != nil {
		b.logger.Error("telegram bot: /announce failed", "actor", caller.ID, "error", err)
		b.reply(ctx, chatID, "Не получилось создать объявление: "+friendlyError(err))
		return
	}
	if ann == nil {
		b.reply(ctx, chatID, "Сервис объявлений отключён.")
		return
	}
	b.reply(ctx, chatID, fmt.Sprintf("Объявление создано. ID: <code>%s</code>", ann.ID))
}

func (b *Bot) handleUnannounce(ctx context.Context, msg *tgmodels.Message) {
	chatID := msg.Chat.ID
	caller, ok := b.requireAdmin(ctx, chatID)
	if !ok {
		return
	}
	if b.announceSvc == nil {
		b.reply(ctx, chatID, "Сервис объявлений не подключён.")
		return
	}

	parts := strings.Fields(msg.Text)
	if len(parts) < 2 {
		b.reply(ctx, chatID, "Скажите ID. <code>/unannounce &lt;uuid&gt;</code>.")
		return
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		b.reply(ctx, chatID, "Это не похоже на UUID объявления.")
		return
	}

	if err := b.announceSvc.DeactivateAnnouncement(ctx, id); err != nil {
		b.logger.Error("telegram bot: /unannounce failed", "actor", caller.ID, "id", id, "error", err)
		b.reply(ctx, chatID, "Не получилось снять объявление: "+friendlyError(err))
		return
	}
	b.reply(ctx, chatID, "Объявление снято.")
}

// requireAdmin resolves the caller and verifies they're an admin with
// linked Telegram. Returns (nil, false) and replies with a refusal if not.
func (b *Bot) requireAdmin(ctx context.Context, chatID int64) (*models.User, bool) {
	caller, err := b.userRepo.GetByTelegramID(ctx, chatID)
	if err != nil {
		b.reply(ctx, chatID, "Команда только для администраторов с привязанным Telegram.")
		return nil, false
	}
	if caller.Role != "admin" {
		b.reply(ctx, chatID, "Команда только для администраторов.")
		return nil, false
	}
	return caller, true
}

// ── /ban / /unban ─────────────────────────────────────────────────────────
//
// Admin-only. Two targeting modes:
//
//	/ban @app_username          — explicit username from the app
//	/ban  (as a reply to a msg) — looks up the target by the replied
//	                              message's sender Telegram ID
//
// Bans must run from an admin account; the bot resolves the caller's
// Telegram ID -> user and checks user.Role == "admin".
func (b *Bot) handleBan(ctx context.Context, msg *tgmodels.Message, ban bool) {
	chatID := msg.Chat.ID
	action := "забанить"
	if !ban {
		action = "разбанить"
	}

	caller, err := b.userRepo.GetByTelegramID(ctx, chatID)
	if err != nil {
		b.reply(ctx, chatID, "Команда доступна только администраторам с привязанным Telegram.")
		return
	}
	if caller.Role != "admin" {
		b.reply(ctx, chatID, "Команда доступна только администраторам.")
		return
	}

	target, resolved := b.resolveModerationTarget(ctx, msg)
	if !resolved {
		b.reply(ctx, chatID,
			fmt.Sprintf("Не понимаю, кого %s. Напишите <code>/ban @username</code> или ответьте этой командой на сообщение нужного пользователя.", action))
		return
	}

	if target.Role == "admin" {
		b.reply(ctx, chatID, "Администратора трогать не дам.")
		return
	}

	var updated *models.User
	if ban {
		updated, err = b.userRepo.BanUser(ctx, target.ID)
	} else {
		updated, err = b.userRepo.UnbanUser(ctx, target.ID)
	}
	if err != nil {
		b.logger.Error("telegram bot: ban/unban failed",
			"actor", caller.ID, "target", target.ID, "ban", ban, "error", err)
		b.reply(ctx, chatID, "Не вышло — попробуйте позже.")
		return
	}

	verb := "забанен"
	if !ban {
		verb = "разбанен"
	}
	state := "активен"
	if updated != nil && updated.BannedAt != nil {
		state = "забанен"
	}
	b.reply(ctx, chatID,
		fmt.Sprintf("Готово. <b>%s</b> %s. Сейчас: %s.",
			htmlEscape(displayName(target)), verb, state))
	b.logger.Info("telegram bot: moderation action",
		"actor", caller.ID, "target", target.ID, "ban", ban)
}

// resolveModerationTarget figures out who the command is acting on. Returns
// (user, true) on success, (nil, false) otherwise.
func (b *Bot) resolveModerationTarget(ctx context.Context, msg *tgmodels.Message) (*models.User, bool) {
	// 1) Reply-to-message → look up by the original sender's Telegram ID.
	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil {
		tgID := msg.ReplyToMessage.From.ID
		if u, err := b.userRepo.GetByTelegramID(ctx, tgID); err == nil {
			return u, true
		}
	}

	// 2) Explicit @username argument — look up by app username.
	parts := strings.Fields(msg.Text)
	if len(parts) >= 2 {
		username := strings.TrimPrefix(parts[1], "@")
		username = strings.TrimSpace(username)
		if username != "" {
			if u, err := b.userRepo.GetByUsername(ctx, username); err == nil {
				return u, true
			}
		}
	}
	return nil, false
}
