package wechat

import (
	"regexp"
	"strings"
)

func FormatMarkdown(content string) string {
	content = regexp.MustCompile("```([\\s\\S]*?)```").ReplaceAllString(
		content,
		"<pre>$1</pre>",
	)

	content = regexp.MustCompile("`([^`]+)`").ReplaceAllString(
		content,
		"<code>$1</code>",
	)

	content = regexp.MustCompile("\\*\\*([^\\*]+)\\*\\*").ReplaceAllString(
		content,
		"<b>$1</b>",
	)

	content = strings.ReplaceAll(content, "<pre>", "\n━━━━━━━━━━━━━━━━\n")
	content = strings.ReplaceAll(content, "</pre>", "\n━━━━━━━━━━━━━━━━\n")

	return content
}
