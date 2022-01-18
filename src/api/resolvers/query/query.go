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

func (r *Resolver) Vod(ctx context.Context, vID primitive.ObjectID) (*model.Vod, error) {
	vod, err := loaders.For(ctx).VodLoader.Load(vID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		return nil, err
	}

	return vod, nil
}

func (r *Resolver) User(ctx context.Context, uID primitive.ObjectID) (*model.User, error) {
	user, err := loaders.For(ctx).UserLoader.Load(uID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		return nil, err
	}

	return user, nil
}

func (r *Resolver) Messages(ctx context.Context, vID primitive.ObjectID, limit int, page int, after time.Time, before time.Time) ([]*model.Chat, error) {
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

func (r *Resolver) Vods(ctx context.Context, userID primitive.ObjectID, limit int, page int, search *string, after *time.Time, before *time.Time) ([]*model.Vod, error) {
	filter := bson.M{
		"user_id": userID,
	}
	if limit <= 0 {
		limit = 50
	} else if limit > 100 {
		limit = 100
	}
	if page < 0 {
		page = 0
	}

	if search != nil {
		filter["$text"] = bson.M{
			"$search": *search,
		}
	}

	timeFilter := bson.M{}

	if after != nil {
		timeFilter["$gte"] = *after
	}

	if before != nil {
		timeFilter["$lte"] = *before
	}

	if len(timeFilter) != 0 {
		filter["started_at"] = timeFilter
	}

	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameVods).Find(ctx, filter, options.Find().SetLimit(int64(limit)).SetSkip(int64(page)*int64(limit)))
	dbVods := []structures.Vod{}
	if err == nil {
		err = cur.All(ctx, &dbVods)
	}
	if err != nil {
		logrus.Error("failed to fetch vods: ", err)
		return nil, helpers.ErrInternalServerError
	}

	vods := make([]*model.Vod, len(dbVods))
	for i, vod := range dbVods {
		vods[i] = vod.ToModel()
	}

	return vods, nil
}
