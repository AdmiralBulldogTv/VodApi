package loaders

import (
	"context"
	"time"

	"github.com/AdmiralBulldogTv/VodApi/graph/loaders"
	"github.com/AdmiralBulldogTv/VodApi/graph/model"
	"github.com/AdmiralBulldogTv/VodApi/src/global"
	"github.com/AdmiralBulldogTv/VodApi/src/structures"
	"github.com/AdmiralBulldogTv/VodApi/src/svc/mongo"
	"github.com/AdmiralBulldogTv/VodApi/src/utils"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const LoadersKey = utils.Key("dataloaders")

type Loaders struct {
	VodLoader          *loaders.VodLoader
	VodsByUserIDLoader *loaders.BatchVodLoader
	UserLoader         *loaders.UserLoader
}

func New(gCtx global.Context) *Loaders {
	return &Loaders{
		VodLoader: loaders.NewVodLoader(loaders.VodLoaderConfig{
			Fetch: func(keys []primitive.ObjectID) ([]*model.Vod, []error) {
				ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
				defer cancel()
				cur, err := gCtx.Inst().Mongo.Collection(mongo.CollectionNameVods).Find(ctx, bson.M{
					"_id": bson.M{
						"$in": keys,
					},
				})
				dbVods := []structures.Vod{}
				if err == nil {
					err = cur.All(ctx, &dbVods)
				}
				vods := make([]*model.Vod, len(keys))
				errs := make([]error, len(keys))
				if err != nil {
					logrus.Error("failed to fetch vods: ", err)
					for i := range errs {
						errs[i] = err
					}
					return vods, errs
				}

				mp := map[primitive.ObjectID]structures.Vod{}
				for _, v := range dbVods {
					mp[v.ID] = v
				}

				for i, v := range keys {
					if vod, ok := mp[v]; ok {
						vods[i] = vod.ToModel()
					} else {
						errs[i] = mongo.ErrNoDocuments
					}
				}

				return vods, errs
			},
			Wait: time.Millisecond * 50,
		}),
		VodsByUserIDLoader: loaders.NewBatchVodLoader(loaders.BatchVodLoaderConfig{
			Fetch: func(keys []primitive.ObjectID) ([][]*model.Vod, []error) {
				ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
				defer cancel()
				cur, err := gCtx.Inst().Mongo.Collection(mongo.CollectionNameVods).Find(ctx, bson.M{
					"user_id": bson.M{
						"$in": keys,
					},
				})
				dbVods := []structures.Vod{}
				if err == nil {
					err = cur.All(ctx, &dbVods)
				}
				vods := make([][]*model.Vod, len(keys))
				errs := make([]error, len(keys))
				if err != nil {
					logrus.Error("failed to fetch vods: ", err)
					for i := range errs {
						errs[i] = err
					}
					return vods, errs
				}

				mp := map[primitive.ObjectID][]structures.Vod{}
				for _, v := range dbVods {
					mp[v.UserID] = append(mp[v.UserID], v)
				}

				for i, v := range keys {
					if vds, ok := mp[v]; ok {
						vs := make([]*model.Vod, len(vds))
						for i, vod := range vds {
							vs[i] = vod.ToModel()
						}
					} else {
						errs[i] = mongo.ErrNoDocuments
					}
				}

				return vods, errs
			},
			Wait: time.Millisecond * 50,
		}),
		UserLoader: loaders.NewUserLoader(loaders.UserLoaderConfig{
			Fetch: func(keys []primitive.ObjectID) ([]*model.User, []error) {
				ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
				defer cancel()
				cur, err := gCtx.Inst().Mongo.Collection(mongo.CollectionNameUsers).Find(ctx, bson.M{
					"_id": bson.M{
						"$in": keys,
					},
				})
				dbUsers := []structures.User{}
				if err == nil {
					err = cur.All(ctx, &dbUsers)
				}
				users := make([]*model.User, len(keys))
				errs := make([]error, len(keys))
				if err != nil {
					logrus.Error("failed to fetch users: ", err)
					for i := range errs {
						errs[i] = err
					}
					return users, errs
				}

				mp := map[primitive.ObjectID]structures.User{}
				for _, v := range dbUsers {
					mp[v.ID] = v
				}

				for i, v := range keys {
					if user, ok := mp[v]; ok {
						users[i] = user.ToModel()
					} else {
						errs[i] = mongo.ErrNoDocuments
					}
				}

				return users, errs
			},
			Wait: time.Millisecond * 50,
		}),
	}
}

func For(ctx context.Context) *Loaders {
	return ctx.Value(LoadersKey).(*Loaders)
}
