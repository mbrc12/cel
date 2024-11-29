package main

func SubtractSlice(a, b []string) []string {
	set := make(map[string]bool)

	for _, item := range b {
		set[item] = true
	}

	var result []string

	for _, item := range a {
		if !set[item] {
			result = append(result, item)
		}
	}

	return result
}
