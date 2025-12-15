package hide

func WithMaskPhoneAndCardRule(pattern []string) Rule {
	return &rule{
		pattern:   pattern,
		transform: maskCardAndPhone,
	}
}

func WithMaskNameRule(pattern []string) Rule {
	return &rule{
		pattern:   pattern,
		transform: maskName,
	}
}

func WithMaskEmailRule(pattern []string) Rule {
	return &rule{
		pattern:   pattern,
		transform: maskEmail,
	}
}

func WithFullExcludeRule(pattern []string) Rule {
	return &rule{
		pattern:   pattern,
		transform: fullExclude,
	}
}

func WithMaskURLRule(pattern []string) Rule {
	return &rule{
		pattern:   pattern,
		transform: maskURL,
	}
}
