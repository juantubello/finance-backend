package utils

import (
	"finance-backend/config"
	"strings"
)

func LoadSubscriptionMap() map[string]string {
	env := config.GetEnv("SUBSCRIPTION_MAP")
	if env == "" {
		return map[string]string{}
	}

	result := make(map[string]string)
	pairs := strings.Split(env, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) == 2 {
			key := strings.ToLower(strings.TrimSpace(parts[0]))
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}
	return result
}
func LoadSubscriptionLogoMap() map[string]string {
	env := config.GetEnv("SUBSCRIPTION_LOGO_MAP")
	if env == "" {
		return map[string]string{}
	}

	result := make(map[string]string)
	pairs := strings.Split(env, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}
	return result
}
