package telegram

import (
	"context"
	"fmt"
	"strings"

	"go-service-template/internal/notify"

	tgmodels "github.com/go-telegram/bot/models"
)

// ── /help ────────────────────────────────────────────────────────────────────
//
// Lists the available commands. Admin-only commands are grouped and marked.
func (b *Bot) handleHelp(ctx context.Context, msg *tgmodels.Message) {
	chatID := msg.Chat.ID

	doc := notify.New().
		Heading(1, "PRODHugs — команды").
		Bold("/me").Text(" — профиль и последние обнимашки").Line().
		Bold("/stats").Text(" — активность за 24 часа").Line().
		Bold("/daily").Text(" — забрать ежедневную награду").Line().
		Bold("/hug").Text(" — обнять кого-нибудь").Line().
		Bold("/help").Text(" — эта справка").Line().
		Line().
		Bold("Для администраторов").Line().
		Bold("/grant").Text(" @user <сумма> — установить баланс").Line().
		Bold("/userinfo").Text(" @user — карточка пользователя").Line().
		Bold("/ban").Text(" / ").Bold("/unban").Text(" @user — модерация").Line().
		Bold("/announce").Text(" / ").Bold("/unannounce").Text(" — объявления").Line().
		Line().
		Quote("Команды для админов доступны только с ролью admin и привязанным Telegram.").
		Build()

	b.replyRich(ctx, chatID, doc)
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

	title := displayName(user)
	if user.DisplayName != nil && *user.DisplayName != "" {
		title = fmt.Sprintf("%s · @%s", displayName(user), user.Username)
	}
	doc := notify.New().Heading(1, title)

	doc.Text("Ранг: ").Bold(stats.Rank).Line()
	doc.Text(fmt.Sprintf("Всего обнимашек: %d (отдано %d, принято %d)",
		stats.TotalHugs, stats.HugsGiven, stats.HugsReceived))

	if len(hugs) > 0 {
		doc.Line().Line().Heading(2, "Последние обнимашки")
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
			doc.Text(fmt.Sprintf("  %s %s · %s", direction, name, hugTypeShortLabel(h.HugType))).Line()
		}
		doc.Quote("Комментарии не показываю — они приватные для получателя.")
	} else {
		doc.Line().Line().Text("Пока что обнимашек нет — самое время кому-нибудь написать!")
	}

	b.replyRich(ctx, chatID, doc.Build())
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

	doc := notify.New().Heading(1, "Обнимашки за 24 часа")
	doc.Text("Всего принято: ").Bold(fmt.Sprintf("%d", total))

	if total == 0 {
		doc.Line().Line().Italic("Пока тихо — никто никого не обнял за сутки.")
		b.replyRich(ctx, chatID, doc.Build())
		return
	}

	// Sparkline-style ASCII bars. Each bar is at most 8 chars wide,
	// scaled to the busiest hour.
	var maxCount int64 = 1
	for _, h := range activity {
		if h.Count > maxCount {
			maxCount = h.Count
		}
	}
	const barWidth = 8
	var sb strings.Builder
	for _, h := range activity {
		hour := h.Timestamp.Format("15:04")
		bars := int((float64(h.Count) / float64(maxCount)) * float64(barWidth))
		if h.Count > 0 && bars == 0 {
			bars = 1
		}
		fmt.Fprintf(&sb, "%s %s %d\n", hour, strings.Repeat("█", bars)+strings.Repeat("·", barWidth-bars), h.Count)
	}
	doc.Line().Line().CodeBlock(strings.TrimRight(sb.String(), "\n"), "")
	b.replyRich(ctx, chatID, doc.Build())
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

	doc := notify.New().Heading(1, "🎁 Ежедневная награда")
	if already {
		doc.Text("На сегодня награда уже у вас. Серия: ").Bold(fmt.Sprintf("%d", streak)).Text(" дн.").Line()
		doc.Text("Возвращайтесь завтра — я напомню сама.")
	} else {
		doc.Text("Получено ").Bold(fmt.Sprintf("+%d", amount)).Text(" обниманий. Серия: ").
			Bold(fmt.Sprintf("%d", streak)).Text(" дн.").Line()
		doc.Text("Баланс: ").Bold(fmt.Sprintf("%d", newBalance)).Text(".")
	}
	b.replyRich(ctx, chatID, doc.Build())
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
