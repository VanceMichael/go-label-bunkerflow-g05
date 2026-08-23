package worker

import "context"

func detachedPublishContext(context.Context) context.Context {
	return context.Background()
}
