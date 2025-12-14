package requestctx

import "context"

type uidKey struct{}

var uidKeyInstance uidKey

func WithUID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, uidKeyInstance, uid)
}

func UID(ctx context.Context) string {
	uid, _ := ctx.Value(uidKeyInstance).(string)
	return uid
}
