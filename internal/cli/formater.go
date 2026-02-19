package cli

func formatCompletionLine(content string, helper string) string {
	return content + "\t" + helper
}
