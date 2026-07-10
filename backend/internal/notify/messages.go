package notify

import "go-service-template/internal/models"

// displayName returns the user's display name, falling back to username.
func displayName(u *models.User) string {
	if u.DisplayName != nil && *u.DisplayName != "" {
		return *u.DisplayName
	}
	return u.Username
}

// hugTypeSuggestionPhrase returns the "хочет ... обнять" phrase for suggestions.
func hugTypeSuggestionPhrase(hugType string) string {
	switch hugType {
	case "bear":
		return "хочет обнять тебя по-медвежьи"
	case "group":
		return "хочет обнять тебя вместе со всеми"
	case "warm":
		return "хочет тепло тебя обнять"
	case "soul":
		return "хочет обнять тебя по-душевному"
	default:
		return "хочет тебя обнять"
	}
}

// hugTypeCompletedNoun returns the hug noun for completed-hug messages.
func hugTypeCompletedNoun(hugType string) string {
	switch hugType {
	case "bear":
		return "медвежьи обнимашки"
	case "group":
		return "групповые обнимашки"
	case "warm":
		return "тёплые обнимашки"
	case "soul":
		return "душевные обнимашки"
	default:
		return "обнимашки"
	}
}

// genderVerb returns the Russian verb form by gender; nil/unknown → fallback.
func genderVerb(gender *string, male, female, fallback string) string {
	if gender == nil {
		return fallback
	}
	switch *gender {
	case "male":
		return male
	case "female":
		return female
	default:
		return fallback
	}
}

// pluralObnimani returns the correct plural of "обнимание" for n.
func pluralObnimani(n int) string {
	abs := n
	if abs < 0 {
		abs = -abs
	}
	mod10 := abs % 10
	mod100 := abs % 100
	if mod10 == 1 && mod100 != 11 {
		return "обниманя"
	}
	if mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14) {
		return "обнимани"
	}
	return "обнимань"
}
