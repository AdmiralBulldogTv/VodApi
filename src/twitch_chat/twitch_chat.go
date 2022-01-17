package twitch_chat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/AdmiralBulldogTv/VodApi/src/emotes"
	"github.com/AdmiralBulldogTv/VodApi/src/global"
	"github.com/AdmiralBulldogTv/VodApi/src/structures"
	"github.com/AdmiralBulldogTv/VodApi/src/svc/mongo"
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/go-redis/redis/v8"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func New(gCtx global.Context) <-chan struct{} {
	done := make(chan struct{})

	ctx, cancel := context.WithTimeout(gCtx, time.Second*5)
	cur, err := gCtx.Inst().Mongo.Collection(mongo.CollectionUsers).Find(ctx, bson.M{})
	cancel()
	users := []structures.User{}
	if err == nil {
		err = cur.All(ctx, &users)
	}
	if err != nil {
		logrus.Fatal("failed to fetch users: ", err)
	}

	cl := twitch.NewAnonymousClient()
	for _, v := range users {
		cl.Join(v.Twitch.Login)
	}

	userMp := map[string]primitive.ObjectID{}

	go func() {
		for {
			ctx, cancel := context.WithTimeout(gCtx, time.Second*5)
			cur, err := gCtx.Inst().Mongo.Collection(mongo.CollectionUsers).Find(ctx, bson.M{})
			cancel()
			users := []structures.User{}
			if err == nil {
				err = cur.All(ctx, &users)
			}
			if err != nil {
				logrus.Fatal("failed to fetch users: ", err)
			}

			nMp := map[string]primitive.ObjectID{}
			cl := twitch.NewAnonymousClient()
			for _, v := range users {
				cl.Join(v.Twitch.Login)
				nMp[v.Twitch.ID] = v.ID
			}
			userMp = nMp
			select {
			case <-gCtx.Done():
				return
			case <-time.After(time.Minute * 30):
			}
		}
	}()

	cl.OnPrivateMessage(func(message twitch.PrivateMessage) {
		uID := userMp[message.RoomID]
		if uID.IsZero() {
			return
		}

		ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
		defer cancel()
		pipe := gCtx.Inst().Redis.Pipeline(ctx)
		setNxCmd := pipe.SetNX(ctx, "twitch-chat-msg:"+message.ID, "1", time.Second*30)
		getCmd := pipe.Get(ctx, "streamer-live:"+uID.Hex())
		_, _ = pipe.Exec(ctx)
		set, err := setNxCmd.Result()
		if err != nil {
			logrus.Warn("failed to check deleted messages: ", err)
			return
		}
		if !set {
			return
		}

		get, err := getCmd.Result()
		if err != nil {
			if err == redis.Nil {
				return
			}

			logrus.Warn("failed to check live channel: ", err)
			return
		}

		vid, err := primitive.ObjectIDFromHex(get)
		if err != nil {
			logrus.Warn("bad vod id in redis: ", err)
			return
		}

		emoteMp := map[string]emotes.Emote{}

		ctx, cancel = context.WithTimeout(gCtx, time.Second*5)
		defer cancel()
		ffzEmotes, err := emotes.GetFFZ(gCtx, ctx, message.RoomID)
		if err != nil {
			logrus.Debug("failed to get ffz emotes: ", err)
		}

		for _, v := range ffzEmotes {
			emoteMp[v.Name] = v
		}

		ctx, cancel = context.WithTimeout(gCtx, time.Second*5)
		defer cancel()
		bttvEmotes, err := emotes.GetBttv(gCtx, ctx, message.RoomID)
		if err != nil {
			logrus.Debug("failed to get bttv emotes: ", err)
		}

		for _, v := range bttvEmotes {
			emoteMp[v.Name] = v
		}

		ctx, cancel = context.WithTimeout(gCtx, time.Second*5)
		defer cancel()
		seventvEmotes, err := emotes.Get7TV(gCtx, ctx, message.RoomID)
		if err != nil {
			logrus.Debug("failed to get 7tv emotes: ", err)
		}

		for _, v := range seventvEmotes {
			emoteMp[v.Name] = v
		}

		emotes := map[string]structures.ChatEmote{}

		for _, v := range message.Emotes {
			emotes[v.Name] = structures.ChatEmote{
				Name: v.Name,
				URLs: []string{
					fmt.Sprintf("https://static-cdn.jtvnw.net/emoticons/v1/%s/1.0", v.ID),
					fmt.Sprintf("https://static-cdn.jtvnw.net/emoticons/v1/%s/2.0", v.ID),
					fmt.Sprintf("https://static-cdn.jtvnw.net/emoticons/v1/%s/3.0", v.ID),
				},
			}
		}

		badges := []structures.ChatBadge{}
		if message.User.Badges["broadcaster"] != 0 {
			badges = append(badges, structures.ChatBadge{
				Name: "Broadcaster",
				URLs: []string{
					"https://static-cdn.jtvnw.net/chat-badges/broadcaster.png",
				},
			})
		}

		if message.User.Badges["mod"] != 0 {
			badges = append(badges, structures.ChatBadge{
				Name: "Moderator",
				URLs: []string{
					"https://static-cdn.jtvnw.net/badges/v1/3267646d-33f0-4b17-b3df-f923a41db1d0/1",
					"https://static-cdn.jtvnw.net/badges/v1/3267646d-33f0-4b17-b3df-f923a41db1d0/2",
					"https://static-cdn.jtvnw.net/badges/v1/3267646d-33f0-4b17-b3df-f923a41db1d0/3",
				},
			})
		}

		if message.User.Badges["vip"] != 0 {
			badges = append(badges, structures.ChatBadge{
				Name: "VIP",
				URLs: []string{
					"https://static-cdn.jtvnw.net/badges/v1/b817aba4-fad8-49e2-b88a-7cc744dfa6ec/1",
					"https://static-cdn.jtvnw.net/badges/v1/b817aba4-fad8-49e2-b88a-7cc744dfa6ec/2",
					"https://static-cdn.jtvnw.net/badges/v1/b817aba4-fad8-49e2-b88a-7cc744dfa6ec/3",
				},
			})
		}

		if message.User.Badges["staff"] != 0 {
			badges = append(badges, structures.ChatBadge{
				Name: "Staff",
				URLs: []string{},
			})
		}

		if message.User.Badges["partner"] != 0 {
			badges = append(badges, structures.ChatBadge{
				Name: "Partner",
				URLs: []string{
					"https://static-cdn.jtvnw.net/badges/v1/d12a2e27-16f6-41d0-ab77-b780518f00a3/1",
					"https://static-cdn.jtvnw.net/badges/v1/d12a2e27-16f6-41d0-ab77-b780518f00a3/2",
					"https://static-cdn.jtvnw.net/badges/v1/d12a2e27-16f6-41d0-ab77-b780518f00a3/3",
				},
			})
		}

		if message.User.Badges["subscriber"] != 0 {
			badges = append(badges, structures.ChatBadge{
				Name: fmt.Sprintf("Subscriber (%d months)", message.User.Badges["subscriber"]),
				URLs: []string{
					"https://static-cdn.jtvnw.net/badges/v1/5d9f2208-5dd8-11e7-8513-2ff4adfae661/1",
					"https://static-cdn.jtvnw.net/badges/v1/5d9f2208-5dd8-11e7-8513-2ff4adfae661/2",
					"https://static-cdn.jtvnw.net/badges/v1/5d9f2208-5dd8-11e7-8513-2ff4adfae661/3",
				},
			})
		}

		splits := strings.Split(message.Message, " ")
		for _, v := range splits {
			if e, ok := emoteMp[v]; ok {
				emotes[v] = structures.ChatEmote{
					Name: v,
					URLs: e.URLs,
				}
			}
		}

		uniqueEmotes := make([]structures.ChatEmote, len(emotes))
		i := 0
		for _, v := range emotes {
			uniqueEmotes[i] = v
			i++
		}

		if _, err := gCtx.Inst().Mongo.Collection(mongo.CollectionNameChat).InsertOne(ctx, structures.Chat{
			VodID: vid,
			Twitch: structures.ChatTwitch{
				ID:          message.ID,
				UserID:      message.User.ID,
				Login:       message.User.Name,
				DisplayName: message.User.DisplayName,
				Color:       message.User.Color,
			},
			Timestamp: message.Time,
			Content:   message.Message,
			Emotes:    uniqueEmotes,
			Badges:    badges,
		}); err != nil {
			logrus.Error("failed to insert message into chat: ", err)
		}
	})

	cl.OnClearChatMessage(func(message twitch.ClearChatMessage) {
		uID := userMp[message.RoomID]
		if uID.IsZero() || message.TargetUserID == "" {
			return
		}

		h := sha256.New()
		data, _ := json.Marshal(message)
		h.Write(data)

		ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
		defer cancel()
		pipe := gCtx.Inst().Redis.Pipeline(ctx)
		setNxCmd := pipe.SetNX(ctx, "twitch-clear-msg:"+hex.EncodeToString(h.Sum(nil)), "1", time.Second*30)
		getCmd := pipe.Get(ctx, "streamer-live:"+uID.Hex())
		_, _ = pipe.Exec(ctx)
		set, err := setNxCmd.Result()
		if err != nil {
			logrus.Warn("failed to check deleted messages: ", err)
			return
		}
		if !set {
			return
		}

		get, err := getCmd.Result()
		if err != nil {
			if err == redis.Nil {
				return
			}

			logrus.Warn("failed to check live channel: ", err)
			return
		}

		vid, err := primitive.ObjectIDFromHex(get)
		if err != nil {
			logrus.Warn("bad vod id in redis: ", err)
			return
		}

		if _, err := gCtx.Inst().Mongo.Collection(mongo.CollectionNameChat).DeleteMany(ctx, bson.M{
			"vod_id":         vid,
			"twitch.user_id": message.TargetUserID,
		}); err != nil {
			logrus.Error("failed to insert message into chat: ", err)
		}
	})

	go func() {
		defer close(done)
		if err := cl.Connect(); err != nil {
			logrus.Fatal("failed to connect to twitch: ", err)
		}
	}()

	go func() {
		<-gCtx.Done()
		_ = cl.Disconnect()
	}()

	return done
}
