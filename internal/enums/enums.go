package enums

import (
	"fmt"
	"strings"
)

var ErrorStatuses = []string{"unresolved", "resolved", "ignored"}
var NotificationChannels = []string{"email", "webhook"}
var NotificationEvents = []string{"new_error", "regression"}

var NotificationEventAliases = map[string]string{
	"error.created":   "new_error",
	"error.reopened":  "regression",
	"error.regressed": "regression",
}

func Validate(flag, val string, valid []string) error {
	if val == "" {
		return nil
	}
	for _, v := range valid {
		if v == val {
			return nil
		}
	}
	return fmt.Errorf("--%s must be one of: %s (got: %q)", flag, strings.Join(valid, ", "), val)
}

func CanonicalNotificationEvent(event string) (string, bool) {
	value := strings.TrimSpace(event)
	if value == "" {
		return "", false
	}
	if alias, ok := NotificationEventAliases[value]; ok {
		value = alias
	}
	for _, valid := range NotificationEvents {
		if value == valid {
			return value, true
		}
	}
	return "", false
}

var InContext = map[string][]string{
	"error_status":         ErrorStatuses,
	"notification_channel": NotificationChannels,
	"notification_event":   append(append([]string{}, NotificationEvents...), "error.created", "error.reopened", "error.regressed"),
}
