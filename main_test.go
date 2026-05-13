package main

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"squeeze newlines", "a\n\n\nb", "a\nb"},
		{"strip format chars", "a​b‌c", "abc"},
		{"single newline kept", "a\nb", "a\nb"},
		{"mixed", "a​\n\n\nb", "a\nb"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalize(tt.in)
			if got != tt.want {
				t.Errorf("normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestHtmlToText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain text",
			in:   "hello world",
			want: "hello world",
		},
		{
			name: "strip tags",
			in:   "<b>hello</b> <i>world</i>",
			want: "hello world",
		},
		{
			name: "img src preserved",
			in:   `before<img src="https://example.com/a.png">after`,
			want: "before\nhttps://example.com/a.png\nafter",
		},
		{
			name: "a href appended after text",
			in:   `<a href="https://example.com">click</a>`,
			want: "click https://example.com ",
		},
		{
			name: "br becomes newline",
			in:   "a<br>b",
			want: "a\nb",
		},
		{
			name: "block elements add newline",
			in:   "<p>a</p><div>b</div><li>c</li>",
			want: "\na\nb\nc",
		},
		{
			name: "nested tags",
			in:   `<p>see <a href="https://example.com">here</a> for more</p>`,
			want: "\nsee here https://example.com  for more",
		},
		{
			name: "img with other attrs",
			in:   `<img alt="x" src="https://example.com/a.png" width="10">`,
			want: "\nhttps://example.com/a.png\n",
		},
		{
			name: "a without href",
			in:   `<a>text</a>`,
			want: "text",
		},
		{
			name: "empty",
			in:   "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := htmlToText(tt.in)
			if got != tt.want {
				t.Errorf("htmlToText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
