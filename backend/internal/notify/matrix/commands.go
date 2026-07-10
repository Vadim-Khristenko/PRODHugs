package matrix

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go-service-template/internal/errorz"
	"go-service-template/internal/models"
	"go-service-template/internal/notify"
)

// notLinkedMsg is the shared reply when a command needs a linked account but
// the sender's Matrix id isn't linked yet.
const notLinkedMsg = "Ваш Matrix пока не привязан — используйте /link или привяжите через настройки на сайте."

// displayName returns the user's display name, falling back to username.
func displayName(u *models.User) string {
	if u.DisplayName != nil && *u.DisplayName != "" {
		return *u.DisplayName
	}
	return u.Username
}

// hugTypeShortLabel returns a tiny label for the /me hug list.
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

// handleHelp sends a richtext help card listing the slash commands.
func (b *Bot) handleHelp(ctx context.Context, roomID string) {
	doc := notify.New().
		Heading(3, "PRODHugs — команды").
		Line().
		Bold("/help").Text(" — эта справка").Line().
		Bold("/link <токен>").Text(" — привязать Matrix к аккаунту").Line().
		Bold("/login <токен>").Text(" — войти на сайт через Matrix").Line().
		Bold("/me").Text(" — профиль и последние обнимашки").Line().
		Bold("/stats").Text(" — активность за 24 часа").Line().
		Bold("/daily").Text(" — забрать ежедневную награду").Line().
		Line().
		Text("Обнимашки принимаются реакцией под сообщением: ").
		Text(reactAccept).Text(" — принять, ").Text(reactDecline).Text(" — отклонить.").
		Build()
	b.replyDoc(ctx, roomID, doc)
}

// handleMe shows the caller's profile and their five most recent hugs.
func (b *Bot) handleMe(ctx context.Context, roomID, sender string) {
	user, err := b.userRepo.GetByMatrixID(ctx, sender)
	if err != nil {
		if errors.Is(err, errorz.ErrUserNotFound) {
			b.reply(ctx, roomID, notLinkedMsg)
			return
		}
		b.logger.Error("matrix bot: /me — lookup failed", "sender", sender, "error", err)
		b.reply(ctx, roomID, "Не получилось собрать статистику, попробуйте позже.")
		return
	}

	stats, err := b.hugSvc.GetUserStats(ctx, user.ID, user.Gender)
	if err != nil {
		b.logger.Error("matrix bot: /me — failed to load stats", "user_id", user.ID, "error", err)
		b.reply(ctx, roomID, "Не получилось собрать статистику, попробуйте позже.")
		return
	}

	hugs, err := b.hugSvc.GetHugHistory(ctx, user.ID, 5, 0)
	if err != nil {
		b.logger.Error("matrix bot: /me — failed to load hugs", "user_id", user.ID, "error", err)
		hugs = nil
	}

	doc := notify.New()
	doc.Bold(displayName(user))
	if user.DisplayName != nil && *user.DisplayName != "" {
		doc.Text(" · @" + user.Username)
	}
	doc.Line().Line()

	doc.Text("Ранг: ").Bold(stats.Rank).Line()
	doc.Text(fmt.Sprintf("Всего обнимашек: %d (отдано %d, принято %d)",
		stats.TotalHugs, stats.HugsGiven, stats.HugsReceived)).Line()

	if len(hugs) > 0 {
		doc.Line().Bold("Последние обнимашки:").Line()
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
		doc.Italic("Комментарии не показываю — они приватные для получателя.")
	} else {
		doc.Line().Text("Пока что обнимашек нет — самое время кому-нибудь написать!")
	}

	b.replyDoc(ctx, roomID, doc.Build())
}

// handleStats mirrors the /feed 24-hour activity widget.
func (b *Bot) handleStats(ctx context.Context, roomID string) {
	activity, err := b.hugSvc.GetHugActivity(ctx)
	if err != nil {
		b.logger.Error("matrix bot: /stats — failed", "error", err)
		b.reply(ctx, roomID, "Не получилось собрать статистику, попробуйте позже.")
		return
	}

	var total int64
	for _, h := range activity {
		total += h.Count
	}

	doc := notify.New().Bold("Обнимашки за последние 24 часа").Line().Line()
	doc.Text(fmt.Sprintf("Всего принято: %d", total))

	if total == 0 {
		doc.Line().Line().Italic("Пока тихо — никто никого не обнял за сутки.")
		b.replyDoc(ctx, roomID, doc.Build())
		return
	}

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
	b.replyDoc(ctx, roomID, doc.Build())
}

// handleDaily claims today's daily reward for the caller.
func (b *Bot) handleDaily(ctx context.Context, roomID, sender string) {
	user, err := b.userRepo.GetByMatrixID(ctx, sender)
	if err != nil {
		if errors.Is(err, errorz.ErrUserNotFound) {
			b.reply(ctx, roomID, notLinkedMsg)
			return
		}
		b.logger.Error("matrix bot: /daily — lookup failed", "sender", sender, "error", err)
		b.reply(ctx, roomID, "Не получилось забрать награду, попробуйте позже.")
		return
	}

	amount, streak, newBalance, already, err := b.hugSvc.ClaimDailyReward(ctx, user.ID)
	if err != nil {
		b.logger.Error("matrix bot: /daily failed", "user_id", user.ID, "error", err)
		b.reply(ctx, roomID, "Не получилось забрать награду, попробуйте позже.")
		return
	}

	doc := notify.New()
	if already {
		doc.Text("На сегодня награда уже у вас. Серия: ").Bold(fmt.Sprintf("%d", streak)).Text(" дн.").Line()
		doc.Text("Возвращайтесь завтра — я напомню сама.")
	} else {
		doc.Text("Получено ").Bold(fmt.Sprintf("+%d", amount)).Text(" обниманий. Серия: ").
			Bold(fmt.Sprintf("%d", streak)).Text(" дн.").Line()
		doc.Text("Баланс: ").Bold(fmt.Sprintf("%d", newBalance)).Text(".")
	}
	b.replyDoc(ctx, roomID, doc.Build())
}
