package router

import (
	"context"
	"fmt"
	"strings"

	"github.com/razorpay/trino-gateway/internal/provider"
)

const financeRoutingIdentifier = "finance"

var financeAllowedUserEmails = []string{
	// Add finance backend users here, for example: "user@razorpay.com".
}

func authorizeFinanceBackendAccess(ctx *context.Context, username string, backendId string, groupId string) error {
	if !isFinanceRoute(backendId, groupId) {
		return nil
	}

	if isFinanceUserAllowed(username) {
		return nil
	}

	provider.Logger(*ctx).Infow(
		fmt.Sprint(LOG_TAG, "User is not authorized for finance backend"),
		map[string]interface{}{
			"backendId": backendId,
			"groupId":   groupId,
			"username":  username,
		},
	)
	return fmt.Errorf("user %s is not authorized for finance backend", username)
}

func isFinanceRoute(backendId string, groupId string) bool {
	return containsNormalized(backendId, financeRoutingIdentifier) ||
		containsNormalized(groupId, financeRoutingIdentifier)
}

func isFinanceUserAllowed(username string) bool {
	normalizedUsername := normalizeEmail(username)
	if normalizedUsername == "" {
		return false
	}
	for _, email := range financeAllowedUserEmails {
		if normalizedUsername == normalizeEmail(email) {
			return true
		}
	}
	return false
}

func containsNormalized(value string, needle string) bool {
	return strings.Contains(normalizeEmail(value), normalizeEmail(needle))
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
