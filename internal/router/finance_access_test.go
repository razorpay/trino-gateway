package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFinanceAccessHelpers(t *testing.T) {
	assert.True(t, isFinanceRoute("finance", ""))
	assert.True(t, isFinanceRoute("trino-finance-coordinator", "adhoc"))
	assert.True(t, isFinanceRoute("trino-adhoc", "finance"))
	assert.False(t, isFinanceRoute("trino-adhoc", "apps"))

	originalAllowedUsers := financeAllowedUserEmails
	defer func() {
		financeAllowedUserEmails = originalAllowedUsers
	}()

	financeAllowedUserEmails = []string{" User.One@Razorpay.com "}

	assert.True(t, isFinanceUserAllowed("user.one@razorpay.com"))
	assert.True(t, isFinanceUserAllowed("USER.ONE@RAZORPAY.COM"))
	assert.False(t, isFinanceUserAllowed("other@razorpay.com"))
	assert.False(t, isFinanceUserAllowed(""))
}
