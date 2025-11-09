package contextutils

import "context"

type uidKey struct{}

var uidKeyInstance uidKey

func WithUID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, uidKeyInstance, uid)
}

func UID(ctx context.Context) string {
	if v := ctx.Value(uidKeyInstance); v != nil {
		if uid, ok := v.(string); ok {
			return uid
		}
	}
	return ""
}
