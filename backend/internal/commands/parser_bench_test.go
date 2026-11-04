package commands

import (
	"strings"
	"testing"
)

// BenchmarkParseComment measures the common accepted commands, including the
// non-ASCII Vietnamese forms that take the full normalization path.
func BenchmarkParseComment(b *testing.B) {
	cases := []struct {
		name string
		raw  string
	}{
		{"XAY", "XÂY"},
		{"THU", "THỦ"},
		{"DANH", "ĐÁNH"},
		{"BUILD", "BUILD"},
		{"DEFEND", "DEFEND"},
		{"ATTACK", "ATTACK"},
		{"A", "A"},
		{"B", "B"},
		{"C", "C"},
	}
	for _, testCase := range cases {
		b.Run(testCase.name, func(b *testing.B) {
			b.ReportAllocs()
			for index := 0; index < b.N; index++ {
				if _, err := ParseComment(testCase.raw); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkParseCommentInvalid measures typical rejected sentences.
func BenchmarkParseCommentInvalid(b *testing.B) {
	cases := []struct {
		name string
		raw  string
	}{
		{"sentence", "TOI MUON XAY"},
		{"punctuation", "XAY!"},
		{"multi", "XAY THU DANH"},
		{"mixed_script", "ХАУ"},
	}
	for _, testCase := range cases {
		b.Run(testCase.name, func(b *testing.B) {
			b.ReportAllocs()
			for index := 0; index < b.N; index++ {
				if _, err := ParseComment(testCase.raw); err == nil {
					b.Fatal("invalid input was accepted")
				}
			}
		})
	}
}

// BenchmarkParseCommentNearLimit measures the 500-byte boundary: one accepted
// padded command and one rejected 500-byte non-command.
func BenchmarkParseCommentNearLimit(b *testing.B) {
	b.Run("accepted_padded", func(b *testing.B) {
		raw := strings.Repeat(" ", MaxCommentBytes-3) + "XAY"
		b.ReportAllocs()
		for index := 0; index < b.N; index++ {
			if _, err := ParseComment(raw); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("rejected_500", func(b *testing.B) {
		raw := strings.Repeat("X", MaxCommentBytes)
		b.ReportAllocs()
		for index := 0; index < b.N; index++ {
			if _, err := ParseComment(raw); err == nil {
				b.Fatal("non-command was accepted")
			}
		}
	})
	b.Run("rejected_501", func(b *testing.B) {
		raw := strings.Repeat("X", MaxCommentBytes+1)
		b.ReportAllocs()
		for index := 0; index < b.N; index++ {
			if _, err := ParseComment(raw); err == nil {
				b.Fatal("oversized comment was accepted")
			}
		}
	})
}
