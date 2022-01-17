package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/AdmiralBulldogTv/VodApi/src/global"
	"github.com/AdmiralBulldogTv/VodApi/src/structures"
	"github.com/AdmiralBulldogTv/VodApi/src/svc/mongo"
	"github.com/AdmiralBulldogTv/VodApi/src/twitch"
	"github.com/AdmiralBulldogTv/VodApi/src/utils"
	"github.com/go-redis/redis/v8"
	"github.com/nicklaw5/helix"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WebhookSubscription struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Type      string `json:"type"`
	Version   string `json:"version"`
	Cost      int    `json:"cost"`
	Condition struct {
		BroadcasterUserID string `json:"broadcaster_user_id"`
	} `json:"condition"`
	Transport struct {
		Method   string `json:"method"`
		Callback string `json:"callback"`
	} `json:"transport"`
}

type WebhookVerifyPending struct {
	Challenge    string              `json:"challenge"`
	Subscription WebhookSubscription `json:"subscription"`
}

type WebhookNotification struct {
	Subscription WebhookSubscription `json:"subscription"`
	Event        struct {
		BroadcasterUserID    string `json:"broadcaster_user_id"`
		BroadcasterUserLogin string `json:"broadcaster_user_login"`
		BroadcasterUserName  string `json:"broadcaster_user_name"`
		Title                string `json:"title"`
		Language             string `json:"language"`
		CategoryID           string `json:"category_id"`
		CategoryName         string `json:"category_name"`
		IsMature             bool   `json:"is_mature"`
	} `json:"event"`
}

func WebhookTwitchHandler(gCtx global.Context) func(ctx *fasthttp.RequestCtx) {
	go func() {
		first := true
		for {
			if !first {
				select {
				case <-time.After(time.Minute * 30):
				case <-gCtx.Done():
					return
				}
			} else {
				first = false
			}

			ctx, cancel := context.WithTimeout(gCtx, time.Second*10)
			subs, err := twitch.GetEventSubs(gCtx, ctx)
			cancel()
			if err != nil {
				logrus.Error("failed to get event subs: ", err)
				continue
			}
			ctx, cancel = context.WithTimeout(gCtx, time.Second*10)
			cur, err := gCtx.Inst().Mongo.Collection(mongo.CollectionUsers).Find(ctx, bson.M{})
			cancel()
			users := []structures.User{}
			if err == nil {
				ctx, cancel = context.WithTimeout(gCtx, time.Second*10)
				err = cur.All(ctx, &users)
				cancel()
				if err != nil {
					logrus.Error("failed to get users: ", err)
					continue
				}
			}

			userMp := map[string]bool{}
			for _, v := range users {
				userMp[v.Twitch.ID] = true
			}

			deleteIDs := []string{}
			for _, v := range subs {
				if v.Status != "enabled" || !userMp[v.Condition.BroadcasterUserID] {
					deleteIDs = append(deleteIDs, v.ID)
				} else {
					delete(userMp, v.Condition.BroadcasterUserID)
				}
			}

			for _, v := range deleteIDs {
				ctx, cancel = context.WithTimeout(gCtx, time.Second*10)
				err = twitch.DeleteEventSub(gCtx, ctx, v)
				cancel()
				if err != nil {
					logrus.Errorf("failed to delete webhook %s: %s", v, err.Error())
				}
			}

			for uid := range userMp {
				ctx, cancel = context.WithTimeout(gCtx, time.Second*10)
				_, err = twitch.CreateEventSub(gCtx, ctx, helix.EventSubSubscription{
					Type:    "channel.update",
					Version: "1",
					Condition: helix.EventSubCondition{
						BroadcasterUserID: uid,
					},
					Transport: helix.EventSubTransport{
						Method:   "webhook",
						Callback: gCtx.Config().Twitch.Webhook.CallbackURL,
						Secret:   gCtx.Config().Twitch.Webhook.Secret,
					},
				})
				cancel()
				if err != nil {
					logrus.Errorf("failed to create webhook for %s: %s", uid, err.Error())
				}
			}
		}
	}()

	return func(ctx *fasthttp.RequestCtx) {
		if !ctx.IsPost() {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			return
		}

		sig := utils.B2S(ctx.Request.Header.Peek("Twitch-Eventsub-Message-Signature"))
		if len(sig) < 7 {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}

		sigBytes, err := hex.DecodeString(sig[7:])
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}

		h := hmac.New(sha256.New, utils.S2B(gCtx.Config().Twitch.Webhook.Secret))
		h.Write(ctx.Request.Header.Peek("Twitch-Eventsub-Message-Id"))
		h.Write(ctx.Request.Header.Peek("Twitch-Eventsub-Message-Timestamp"))
		h.Write(ctx.Request.Body())
		if !hmac.Equal(h.Sum(nil), sigBytes) {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}

		set, err := gCtx.Inst().Redis.SetNX(ctx, fmt.Sprintf("twitch-webhook-events:%s", ctx.Request.Header.Peek("Twitch-Eventsub-Message-Id")), "1", time.Hour*12)
		if err != nil {
			logrus.Error("redis failed to set webhook event: ", err)
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			return
		}

		if !set {
			ctx.SetStatusCode(fasthttp.StatusOK)
			return
		}

		switch utils.B2S(ctx.Request.Header.Peek("Twitch-Eventsub-Message-Type")) {
		case "notification":
			// we need to consume the data
			body := WebhookNotification{}
			if err := json.Unmarshal(ctx.Request.Body(), &body); err != nil {
				logrus.Errorf("bad body from twitch: %s : %s", err.Error(), ctx.Request.Body())
				ctx.SetStatusCode(fasthttp.StatusBadRequest)
				return
			}

			user := structures.User{}
			res := gCtx.Inst().Mongo.Collection(mongo.CollectionNameUsers).FindOne(ctx, bson.M{
				"twitch.id": body.Subscription.Condition.BroadcasterUserID,
			})
			err = res.Err()
			if err == nil {
				err = res.Decode(&user)
			}
			if err != nil {
				if err == mongo.ErrNoDocuments {
					ctx.SetStatusCode(fasthttp.StatusNotFound)
				} else {
					logrus.Errorf("error on mongo webhook lookup: %s", err.Error())
					ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				}
				return
			}

			vodID, err := gCtx.Inst().Redis.Get(ctx, "streamer-live:"+user.ID.Hex())
			if err != nil {
				if err == redis.Nil {
					ctx.SetStatusCode(fasthttp.StatusOK)
					return
				}

				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				logrus.Error("failed to check streamer live: ", err)
				return
			}

			vID, err := primitive.ObjectIDFromHex(vodID.(string))
			if err != nil {
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				logrus.Error("bad resp from redis: ", vodID)
				return
			}

			vod := structures.Vod{}
			res = gCtx.Inst().Mongo.Collection(mongo.CollectionNameVods).FindOne(ctx, bson.M{
				"_id": vID,
			})
			err = res.Err()
			if err == nil {
				err = res.Decode(&vod)
			}
			if err != nil {
				logrus.Errorf("error on mongo webhook lookup: %s", err.Error())
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				return
			}
			update := bson.M{
				"$set": bson.M{
					"title": body.Event.Title,
				},
			}
			if len(vod.Categories) == 0 || vod.Categories[len(vod.Categories)-1].ID != body.Event.CategoryID {
				url := fmt.Sprintf("https://static-cdn.jtvnw.net/ttv-boxart/%s-144x192.jpg", body.Event.CategoryID)
				if body.Event.CategoryName == "" {
					body.Event.CategoryName = "Unknown"
					body.Event.CategoryID = "0"
					url = "https://static-cdn.jtvnw.net/ttv-static/404_boxart.jpg"
				}
				update["$push"] = bson.M{
					"categories": structures.VodCategory{
						Timestamp: time.Now(),
						Name:      body.Event.CategoryName,
						ID:        body.Event.CategoryID,
						URL:       url,
					},
				}
			}

			_, err = gCtx.Inst().Mongo.Collection(mongo.CollectionNameVods).UpdateOne(ctx, bson.M{
				"_id": vID,
			}, update)
			if err != nil {
				logrus.Error("failed to update vod: ", err)
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				return
			}

			ctx.SetStatusCode(fasthttp.StatusNoContent)
		case "webhook_callback_verification":
			// we need to verify the webhook
			body := WebhookVerifyPending{}
			if err := json.Unmarshal(ctx.Request.Body(), &body); err != nil {
				logrus.Errorf("bad body from twitch: %s : %s", err.Error(), ctx.Request.Body())
				ctx.SetStatusCode(fasthttp.StatusBadRequest)
				return
			}

			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.SetBodyString(body.Challenge)
		case "revocation":
			ctx.SetStatusCode(fasthttp.StatusNoContent)
		default:
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
		}
	}
}
