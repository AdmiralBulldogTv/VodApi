package auth

import (
	"context"

	"github.com/AdmiralBulldogTv/VodApi/src/api/helpers"
	"github.com/AdmiralBulldogTv/VodApi/src/structures"
)

func For(ctx context.Context) *structures.User {
	raw, _ := ctx.Value(helpers.UserKey).(*structures.User)
	return raw
}
