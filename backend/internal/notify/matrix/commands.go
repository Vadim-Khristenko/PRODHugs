package matrix

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go-service-template/internal/errorz"
	"go-service-template/internal/models"
	"go-service-template/internal/notify"
)

// notLinkedMsg is the shared reply when a command needs a linked account but
// the sender's Matrix id isn't linked yet.
const notLinkedMsg = "Ваш Matrix пока не привязан — используйте !link или привяжите через настройки на сайте."

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

// requireAdmin resolves the sender's Matrix id to a user and verifies the
// admin role. Returns (nil, false) and replies with a refusal if not linked
// or not an admin.
func (b *Bot) requireAdmin(ctx context.Context, roomID, sender string) (*models.User, bool) {
	caller, err := b.userRepo.GetByMatrixID(ctx, sender)
	if err != nil {
		b.reply(ctx, roomID, "Команда только для администраторов с привязанным Matrix.")
		return nil, false
	}
	if caller.Role != "admin" {
		b.reply(ctx, roomID, "Команда только для администраторов.")
		return nil, false
	}
	return caller, true
}

// handleHelp sends a richtext help card listing the slash commands.
func (b *Bot) handleHelp(ctx context.Context, roomID string) {
	doc := notify.New().
		Heading(1, "PRODHugs — команды").
		Bold("!help").Text(" — эта справка").Line().
		Bold("!link <токен>").Text(" — привязать Matrix к аккаунту").Line().
		Bold("!login <токен>").Text(" — войти на сайт через Matrix").Line().
		Bold("!me").Text(" — профиль и последние обнимашки").Line().
		Bold("!stats").Text(" — активность за 24 часа").Line().
		Bold("!daily").Text(" — забрать ежедневную награду").Line().
		Line().
		Heading(2, "Для администраторов").
		Bold("!grant @user <сумма>").Text(" — установить баланс").Line().
		Bold("!userinfo @user").Text(" — карточка пользователя").Line().
		Line().
		Quote("Команды для админов доступны только с ролью admin.").
		Line().
		Text("Обнимашки принимаются реакцией под сообщением: ").
		Text(reactAccept).Text(" — принять, ").Text(reactDecline).Text(" — отклонить.").
		Build()
	b.replyDoc(ctx, roomID, doc)
}

// grantAmountArg extracts the amount from a /grant command's fields. Scans
// from the end so it works whether or not an explicit @username precedes it.
// Returns ok=false when no valid non-negative int32 amount is present.
func grantAmountArg(fields []string) (int32, bool) {
	for i := len(fields) - 1; i >= 1; i-- {
		n, err := strconv.ParseInt(strings.TrimSpace(fields[i]), 10, 32)
		if err == nil && n >= 0 {
			return int32(n), true
		}
	}
	return 0, false
}

// resolveTarget resolves a /grant or /userinfo target by APP username from the
// first non-numeric argument (stripping a leading @). Returns ok=false when no
// username argument is present or the user isn't found.
func (b *Bot) resolveTarget(ctx context.Context, fields []string) (*models.User, bool) {
	for i := 1; i < len(fields); i++ {
		arg := strings.TrimSpace(strings.TrimPrefix(fields[i], "@"))
		if arg == "" {
			continue
		}
		if _, err := strconv.ParseInt(arg, 10, 32); err == nil {
			continue // numeric — that's the amount, not a username
		}
		if u, err := b.userRepo.GetByUsername(ctx, arg); err == nil {
			return u, true
		}
		return nil, false
	}
	return nil, false
}

// handleGrant SETS the target user's coin balance to an absolute amount.
func (b *Bot) handleGrant(ctx context.Context, roomID, sender string, fields []string) {
	caller, ok := b.requireAdmin(ctx, roomID, sender)
	if !ok {
		return
	}
	if b.adminSvc == nil {
		b.reply(ctx, roomID, "Сервис баланса не подключён.")
		return
	}

	amount, ok := grantAmountArg(fields)
	if !ok {
		b.reply(ctx, roomID, "Сколько начислить? !grant @user <сумма>.")
		return
	}
	target, ok := b.resolveTarget(ctx, fields)
	if !ok {
		b.reply(ctx, roomID, "Не понимаю, кому. Напишите !grant @user <сумма>.")
		return
	}

	bal, err := b.adminSvc.AdminUpdateBalance(ctx, target.ID, amount)
	if err != nil {
		b.logger.Error("matrix bot: /grant failed", "actor", caller.ID, "target", target.ID, "amount", amount, "error", err)
		b.reply(ctx, roomID, "Не получилось изменить баланс, попробуйте позже.")
		return
	}
	set := amount
	if bal != nil {
		set = bal.Amount
	}
	b.logger.Info("matrix bot: /grant", "actor", caller.ID, "target", target.ID, "amount", set)
	doc := notify.New().Heading(1, "💰 Баланс обновлён").
		Text("Баланс ").Bold(displayName(target)).Text(" установлен: ").Bold(fmt.Sprintf("%d", set)).Text(".").
		Build()
	b.replyDoc(ctx, roomID, doc)
}

// handleUserinfo shows the target user's role, balance, and linked platforms.
func (b *Bot) handleUserinfo(ctx context.Context, roomID, sender string, fields []string) {
	if _, ok := b.requireAdmin(ctx, roomID, sender); !ok {
		return
	}
	target, ok := b.resolveTarget(ctx, fields)
	if !ok {
		b.reply(ctx, roomID, "О ком? Напишите !userinfo @user.")
		return
	}

	var balAmount int32
	if bal, err := b.hugSvc.GetBalance(ctx, target.ID); err != nil {
		b.logger.Warn("matrix bot: /userinfo balance lookup failed", "target", target.ID, "error", err)
	} else if bal != nil {
		balAmount = bal.Amount
	}

	tgLinked := "нет"
	if target.TelegramID != nil {
		tgLinked = "да"
	}
	matrixLinked := "нет"
	if target.MatrixID != nil {
		matrixLinked = "да"
	}

	doc := notify.New().Heading(1, displayName(target)+" · @"+target.Username)
	doc.Text("Роль: ").Bold(target.Role).Line()
	doc.Text("Баланс: ").Bold(fmt.Sprintf("%d", balAmount)).Line()
	doc.Text(fmt.Sprintf("Telegram: %s · Matrix: %s", tgLinked, matrixLinked))
	if target.BannedAt != nil {
		doc.Line().Text("Статус: ").Bold("забанен")
	}
	b.replyDoc(ctx, roomID, doc.Build())
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

	title := displayName(user)
	if user.DisplayName != nil && *user.DisplayName != "" {
		title = displayName(user) + " · @" + user.Username
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
		doc.Line().Quote("Комментарии не показываю — они приватные для получателя.")
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

	doc := notify.New().Heading(1, "Обнимашки за 24 часа")
	doc.Text("Всего принято: ").Bold(fmt.Sprintf("%d", total))

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

	doc := notify.New().Heading(1, "🎁 Ежедневная награда")
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
