package telegram

import "testing"

func TestExtractHugComment(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		usedReply bool
		want      string
	}{
		{
			name:      "explicit username with comment",
			text:      "/hug @anna привет",
			usedReply: false,
			want:      "привет",
		},
		{
			name:      "bot-suffixed command, username only, no comment",
			text:      "/hug@bot @anna",
			usedReply: false,
			want:      "",
		},
		{
			name:      "reply mode with comment",
			text:      "/hug обними",
			usedReply: true,
			want:      "обними",
		},
		{
			name:      "bare command",
			text:      "/hug",
			usedReply: false,
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHugComment(tt.text, tt.usedReply)
			if got != tt.want {
				t.Errorf("extractHugComment(%q, %v) = %q, want %q", tt.text, tt.usedReply, got, tt.want)
			}
		})
	}
}
