package telegram

import (
	"context"
	"fmt"
	"strings"

	tgmodels "github.com/go-telegram/bot/models"
)

// ── /help ────────────────────────────────────────────────────────────────────
//
// Lists the available commands. Admin-only commands are grouped and marked.
func (b *Bot) handleHelp(ctx context.Context, msg *tgmodels.Message) {
	chatID := msg.Chat.ID

	var sb strings.Builder
	sb.WriteString("<b>PRODHugs — команды</b>\n\n")
	sb.WriteString("<b>/me</b> — профиль и последние обнимашки\n")
	sb.WriteString("<b>/stats</b> — активность за 24 часа\n")
	sb.WriteString("<b>/daily</b> — забрать ежедневную награду\n")
	sb.WriteString("<b>/hug</b> — обнять кого-нибудь\n")
	sb.WriteString("<b>/help</b> — эта справка\n\n")
	sb.WriteString("<b>Для администраторов</b>\n")
	sb.WriteString("<b>/grant</b> @user &lt;сумма&gt; — установить баланс\n")
	sb.WriteString("<b>/userinfo</b> @user — карточка пользователя\n")
	sb.WriteString("<b>/ban</b> / <b>/unban</b> @user — модерация\n")
	sb.WriteString("<b>/announce</b> / <b>/unannounce</b> — объявления\n\n")
	sb.WriteString("<blockquote>Команды для админов доступны только с ролью admin и привязанным Telegram.</blockquote>")

	b.reply(ctx, chatID, sb.String())
}

// ── /me ────────────────────────────────────────────────────────────────────
//
// Shows the caller's profile and their five most recent hugs. Comments on
// those hugs are intentionally NOT included — comments are private to the
// recipient, and even /me wouldn't show the sender what comment they wrote.
func (b *Bot) handleMe(ctx context.Context, msg *tgmodels.Message) {
	chatID := msg.Chat.ID

	user, err := b.userRepo.GetByTelegramID(ctx, chatID)
	if err != nil {
		b.reply(ctx, chatID, "Ваш Telegram пока не привязан к аккаунту. Откройте настройки на сайте — там есть кнопка для привязки.")
		return
	}

	stats, err := b.hugSvc.GetUserStats(ctx, user.ID, user.Gender)
	if err != nil {
		b.logger.Error("telegram bot: /me — failed to load stats", "user_id", user.ID, "error", err)
		b.reply(ctx, chatID, "Не получилось собрать статистику, попробуйте позже.")
		return
	}

	hugs, err := b.hugSvc.GetHugHistory(ctx, user.ID, 5, 0)
	if err != nil {
		b.logger.Error("telegram bot: /me — failed to load hugs", "user_id", user.ID, "error", err)
		hugs = nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>%s</b>", htmlEscape(displayName(user)))
	if user.DisplayName != nil && *user.DisplayName != "" {
		fmt.Fprintf(&sb, " · @%s", htmlEscape(user.Username))
	}
	sb.WriteString("\n\n")

	fmt.Fprintf(&sb, "Ранг: <b>%s</b>\n", htmlEscape(stats.Rank))
	fmt.Fprintf(&sb, "Всего обнимашек: %d (отдано %d, принято %d)\n",
		stats.TotalHugs, stats.HugsGiven, stats.HugsReceived)

	if len(hugs) > 0 {
		sb.WriteString("\n<b>Последние обнимашки:</b>\n")
		for _, h := range hugs {
			var direction, otherName string
			var otherDN *string
			if h.GiverID == user.ID {
				direction = "→"
				otherName = h.ReceiverUsername
				otherDN = h.ReceiverDisplayName
			} else {
				direction = "←"
				otherName = h.GiverUsername
				otherDN = h.GiverDisplayName
			}
			name := otherName
			if otherDN != nil && *otherDN != "" {
				name = *otherDN
			}
			fmt.Fprintf(&sb, "  %s %s · %s\n",
				direction,
				htmlEscape(name),
				hugTypeShortLabel(h.HugType),
			)
		}
		sb.WriteString("\n<blockquote>Комментарии не показываю — они приватные для получателя.</blockquote>")
	} else {
		sb.WriteString("\nПока что обнимашек нет — самое время кому-нибудь написать!")
	}

	b.reply(ctx, chatID, sb.String())
}

// ── /stats ────────────────────────────────────────────────────────────────
//
// Mirrors the /feed page's 24-hour activity widget: total accepted hugs in
// the last day plus a tiny ASCII histogram by hour. No new SQL — reuses the
// same GetHugActivity query the website chart consumes.
func (b *Bot) handleStats(ctx context.Context, msg *tgmodels.Message) {
	chatID := msg.Chat.ID

	activity, err := b.hugSvc.GetHugActivity(ctx)
	if err != nil {
		b.logger.Error("telegram bot: /stats — failed", "error", err)
		b.reply(ctx, chatID, "Не получилось собрать статистику, попробуйте позже.")
		return
	}

	var total int64
	for _, h := range activity {
		total += h.Count
	}

	var sb strings.Builder
	sb.WriteString("<b>Обнимашки за последние 24 часа</b>\n\n")
	fmt.Fprintf(&sb, "Всего принято: <b>%d</b>\n", total)

	if total == 0 {
		sb.WriteString("\n<i>Пока тихо — никто никого не обнял за сутки.</i>")
		b.reply(ctx, chatID, sb.String())
		return
	}

	// Sparkline-style ASCII bars. Each bar is at most 8 chars wide,
	// scaled to the busiest hour.
	var max int64 = 1
	for _, h := range activity {
		if h.Count > max {
			max = h.Count
		}
	}
	const barWidth = 8
	sb.WriteString("\n<code>")
	for _, h := range activity {
		hour := h.Timestamp.Format("15:04")
		bars := int((float64(h.Count) / float64(max)) * float64(barWidth))
		if h.Count > 0 && bars == 0 {
			bars = 1
		}
		fmt.Fprintf(&sb, "%s %s %d\n", hour, strings.Repeat("█", bars)+strings.Repeat("·", barWidth-bars), h.Count)
	}
	sb.WriteString("</code>")
	b.reply(ctx, chatID, sb.String())
}

// ── /daily ────────────────────────────────────────────────────────────────
//
// Claims today's daily reward for the caller. Same backend path as the
// website's claim button.
func (b *Bot) handleDaily(ctx context.Context, msg *tgmodels.Message) {
	chatID := msg.Chat.ID
	user, err := b.userRepo.GetByTelegramID(ctx, chatID)
	if err != nil {
		b.reply(ctx, chatID, "Ваш Telegram пока не привязан. Откройте настройки на сайте — там есть кнопка для привязки.")
		return
	}

	amount, streak, newBalance, already, err := b.hugSvc.ClaimDailyReward(ctx, user.ID)
	if err != nil {
		b.logger.Error("telegram bot: /daily failed", "user_id", user.ID, "error", err)
		b.reply(ctx, chatID, "Не получилось забрать награду, попробуйте позже.")
		return
	}

	var sb strings.Builder
	if already {
		fmt.Fprintf(&sb, "На сегодня награда уже у вас. Серия: <b>%d</b> дн.\n", streak)
		fmt.Fprintf(&sb, "Возвращайтесь завтра — я напомню сама.")
	} else {
		fmt.Fprintf(&sb, "Получено <b>+%d</b> обниманий. Серия: <b>%d</b> дн.\n", amount, streak)
		fmt.Fprintf(&sb, "Баланс: <b>%d</b>.", newBalance)
	}
	b.reply(ctx, chatID, sb.String())
}

// hugTypeShortLabel returns a tiny label for /me's hug list.
func hugTypeShortLabel(t string) string {
	switch t {
	case "bear":
		return "медвежья"
	case "group":
		return "групповая"
	case "warm":
		return "тёплая"
	case "soul":
		return "душевная"
	default:
		return "обычная"
	}
}
