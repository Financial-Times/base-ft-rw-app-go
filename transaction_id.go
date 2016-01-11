package baseftrwapp

import (
	"math/rand"
	"net/http"

	"golang.org/x/net/context"
)

const transactionIdHeader = "X-Request-Id"
const transactionIdKey string = "transaction_id"

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func GetTransactionIdFromContext(ctx context.Context) (string, bool) {
	// ctx.Value returns nil if ctx has no value for the key;
	// string type assertion returns ok=false for nil.
	transactionId, ok := ctx.Value(transactionIdKey).(string)
	return transactionId, ok
}

func GetTransactionIdFromRequest(req *http.Request) string {
	transactionId := req.Header.Get(transactionIdHeader)
	if transactionId == "" {
		transactionId = "tid_" + randString(10)
	}
	return transactionId
}

func TransactionAwareContext(ctx context.Context, transactionId string) context.Context {
	return context.WithValue(ctx, transactionIdKey, transactionId)
}

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
