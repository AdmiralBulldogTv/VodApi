package query

import (
	"context"
	"time"

	"github.com/AdmiralBulldogTv/VodApi/graph/generated"
	"github.com/AdmiralBulldogTv/VodApi/graph/model"
	"github.com/AdmiralBulldogTv/VodApi/src/api/helpers"
	"github.com/AdmiralBulldogTv/VodApi/src/api/loaders"
	"github.com/AdmiralBulldogTv/VodApi/src/api/types"
	"github.com/AdmiralBulldogTv/VodApi/src/structures"
	"github.com/AdmiralBulldogTv/VodApi/src/svc/mongo"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.QueryResolver {
	return &Resolver{
		Resolver: r,
	}
}

func (r *Resolver) Vod(ctx context.Context, id string) (*model.Vod, error) {
	vID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, helpers.ErrBadObjectID
	}

	vod, err := loaders.For(ctx).VodLoader.Load(vID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		return nil, err
	}

	return vod, nil
}

func (r *Resolver) Vods(ctx context.Context, userID string) ([]*model.Vod, error) {
	uID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, helpers.ErrBadObjectID
	}

	vods, err := loaders.For(ctx).VodsByUserIDLoader.Load(uID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		return nil, err
	}

	return vods, nil
}

func (r *Resolver) User(ctx context.Context, id string) (*model.User, error) {
	uID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, helpers.ErrBadObjectID
	}

	user, err := loaders.For(ctx).UserLoader.Load(uID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		return nil, err
	}

	return user, nil
}

func (r *Resolver) Messages(ctx context.Context, vodID string, limit int, page int, after time.Time, before time.Time) ([]*model.Chat, error) {
	vID, _ := primitive.ObjectIDFromHex(vodID)
	filter := bson.M{
		"vod_id": vID,
		"timestamp": bson.M{
			"$gte": after,
			"$lte": before,
		},
	}

	if limit <= 0 {
		limit = 500
	} else if limit > 2500 {
		limit = 2500
	}
	if page < 0 {
		page = 0
	}

	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameChat).Find(ctx, filter, options.Find().SetLimit(int64(limit)).SetSkip(int64(page)*int64(limit)))
	dbChat := []structures.Chat{}
	if err == nil {
		err = cur.All(ctx, &dbChat)
	}
	if err != nil {
		logrus.Error("failed to fetch vods: ", err)
		return nil, helpers.ErrInternalServerError
	}

	chats := make([]*model.Chat, len(dbChat))
	for i, chat := range dbChat {
		chats[i] = chat.ToModel()
	}

	return chats, nil
}
