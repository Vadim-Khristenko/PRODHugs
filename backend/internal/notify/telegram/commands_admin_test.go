package telegram

import (
	"strings"
	"testing"
)

func TestGrantAmountArg(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		want   int32
		wantOK bool
	}{
		{name: "username and amount", text: "/grant @anna 100", want: 100, wantOK: true},
		{name: "reply mode amount only", text: "/grant 250", want: 250, wantOK: true},
		{name: "zero is valid", text: "/grant @anna 0", want: 0, wantOK: true},
		{name: "bot-suffixed command", text: "/grant@bot @anna 42", want: 42, wantOK: true},
		{name: "no amount", text: "/grant @anna", want: 0, wantOK: false},
		{name: "bare command", text: "/grant", want: 0, wantOK: false},
		{name: "negative rejected", text: "/grant @anna -5", want: 0, wantOK: false},
		{name: "non-numeric rejected", text: "/grant @anna lots", want: 0, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := grantAmountArg(strings.Fields(tt.text))
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("grantAmountArg(%q) = (%d, %v), want (%d, %v)", tt.text, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
