package copilot

func copySovrintStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
}

func intersectSovrintStrings(left []string, right []string) []string {
	allowed := make(map[string]struct{}, len(right))
	for _, value := range right {
		allowed[value] = struct{}{}
	}
	result := make([]string, 0)
	for _, value := range left {
		if _, ok := allowed[value]; ok {
			result = append(result, value)
		}
	}
	return result
}

func mergeSovrintStrings(left []string, right []string) []string {
	seen := make(map[string]struct{}, len(left)+len(right))
	result := make([]string, 0, len(left)+len(right))
	for _, value := range append(copySovrintStrings(left), right...) {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
