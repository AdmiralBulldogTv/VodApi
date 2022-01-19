package emotes

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/AdmiralBulldogTv/VodApi/src/global"
	"github.com/go-redis/redis/v8"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type BttvEmote struct {
	ID   string `json:"id"`
	Code string `json:"code"`
}

type BttvChannel struct {
	ChannelEmotes []BttvEmote `json:"channelEmotes"`
	SharedEmotes  []BttvEmote `json:"sharedEmotes"`
}

func GetBttv(gCtx global.Context, ctx context.Context, id string) ([]Emote, error) {
	resp, err := gCtx.Inst().Redis.Get(ctx, "emotes-cached:bttv:"+id)
	if err != nil {
		if err != redis.Nil {
			logrus.Warn("failed to fetched cached bttv emotes: ", err)
		}
	} else {
		emotes := []Emote{}
		if err := json.UnmarshalFromString(resp.(string), &emotes); err != nil {
			logrus.Warn("bad cache value for bttv emotes: ", err)
		} else {
			return emotes, nil
		}
	}

	bttvGlobalEmotes := []BttvEmote{}
	{
		req, err := http.NewRequestWithContext(ctx, "GET", "https://api.betterttv.net/3/cached/emotes/global", nil)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:bttv:"+id, "[]", time.Second*30)
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:bttv:"+id, "[]", time.Second*30)
			return nil, err
		}
		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:bttv:"+id, "[]", time.Second*30)
			return nil, err
		}

		if err := json.Unmarshal(data, &bttvGlobalEmotes); err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:bttv:"+id, "[]", time.Second*30)
			return nil, err
		}
	}

	channel := BttvChannel{}
	{
		req, err := http.NewRequestWithContext(ctx, "GET", "https://api.betterttv.net/3/cached/users/twitch/"+id, nil)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:bttv:"+id, "[]", time.Second*30)
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:bttv:"+id, "[]", time.Second*30)
			return nil, err
		}
		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:bttv:"+id, "[]", time.Second*30)
			return nil, err
		}

		if err := json.Unmarshal(data, &channel); err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:bttv:"+id, "[]", time.Second*30)
			return nil, err
		}
	}

	emotes := []Emote{}
	for _, v := range bttvGlobalEmotes {
		emotes = append(emotes, Emote{
			ID:   v.ID,
			Name: v.Code,
			URLs: []string{
				fmt.Sprintf("https://cdn.betterttv.net/emote/%s/1x", v.ID),
				fmt.Sprintf("https://cdn.betterttv.net/emote/%s/2x", v.ID),
				fmt.Sprintf("https://cdn.betterttv.net/emote/%s/3x", v.ID),
			},
			Provider: EmoteProviderBTTV,
		})
	}

	for _, v := range channel.ChannelEmotes {
		emotes = append(emotes, Emote{
			ID:   v.ID,
			Name: v.Code,
			URLs: []string{
				fmt.Sprintf("https://cdn.betterttv.net/emote/%s/1x", v.ID),
				fmt.Sprintf("https://cdn.betterttv.net/emote/%s/2x", v.ID),
				fmt.Sprintf("https://cdn.betterttv.net/emote/%s/3x", v.ID),
			},
			Provider: EmoteProviderBTTV,
		})
	}

	for _, v := range channel.SharedEmotes {
		emotes = append(emotes, Emote{
			ID:   v.ID,
			Name: v.Code,
			URLs: []string{
				fmt.Sprintf("https://cdn.betterttv.net/emote/%s/1x", v.ID),
				fmt.Sprintf("https://cdn.betterttv.net/emote/%s/2x", v.ID),
				fmt.Sprintf("https://cdn.betterttv.net/emote/%s/3x", v.ID),
			},
			Provider: EmoteProviderBTTV,
		})
	}

	data, _ := json.MarshalToString(emotes)
	if err = gCtx.Inst().Redis.SetEX(ctx, "emotes-cached:bttv:"+id, data, time.Minute*30); err != nil {
		logrus.Warn("failed to set emote bttv cache: ", err)
	}

	return emotes, nil
}

type FFZRoom struct {
	Sets map[string]FFZEmoteSet `json:"sets"`
}

type FFZEmoteSet struct {
	Emoticons []FFZEmote `json:"emoticons"`
}

type FFZEmote struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func GetFFZ(gCtx global.Context, ctx context.Context, id string) ([]Emote, error) {
	resp, err := gCtx.Inst().Redis.Get(ctx, "emotes-cached:ffz:"+id)
	if err != nil {
		if err != redis.Nil {
			logrus.Warn("failed to fetched cached ffz emotes: ", err)
		}
	} else {
		emotes := []Emote{}
		if err := json.UnmarshalFromString(resp.(string), &emotes); err != nil {
			logrus.Warn("bad cache value for ffz emotes: ", err)
		} else {
			return emotes, nil
		}
	}

	ffzGlobalRoom := FFZRoom{}
	{
		req, err := http.NewRequestWithContext(ctx, "GET", "https://api.frankerfacez.com/v1/set/global", nil)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:ffz:"+id, "[]", time.Second*30)
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:ffz:"+id, "[]", time.Second*30)
			return nil, err
		}
		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:ffz:"+id, "[]", time.Second*30)
			return nil, err
		}

		if err := json.Unmarshal(data, &ffzGlobalRoom); err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:ffz:"+id, "[]", time.Second*30)
			return nil, err
		}
	}

	channelRoom := FFZRoom{}
	{
		req, err := http.NewRequestWithContext(ctx, "GET", "https://api.frankerfacez.com/v1/room/id/"+id, nil)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:ffz:"+id, "[]", time.Second*30)
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:ffz:"+id, "[]", time.Second*30)
			return nil, err
		}
		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:ffz:"+id, "[]", time.Second*30)
			return nil, err
		}

		if err := json.Unmarshal(data, &channelRoom); err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:ffz:"+id, "[]", time.Second*30)
			return nil, err
		}
	}

	emotes := []Emote{}
	for _, s := range ffzGlobalRoom.Sets {
		for _, v := range s.Emoticons {
			emotes = append(emotes, Emote{
				ID:   fmt.Sprint(v.ID),
				Name: v.Name,
				URLs: []string{
					fmt.Sprintf("https://cdn.frankerfacez.com/emote/%d/1", v.ID),
					fmt.Sprintf("https://cdn.frankerfacez.com/emote/%d/2", v.ID),
					fmt.Sprintf("https://cdn.frankerfacez.com/emote/%d/3", v.ID),
				},
				Provider: EmoteProviderFFZ,
			})
		}
	}

	for _, s := range channelRoom.Sets {
		for _, v := range s.Emoticons {
			emotes = append(emotes, Emote{
				ID:   fmt.Sprint(v.ID),
				Name: v.Name,
				URLs: []string{
					fmt.Sprintf("https://cdn.frankerfacez.com/emote/%d/1", v.ID),
					fmt.Sprintf("https://cdn.frankerfacez.com/emote/%d/2", v.ID),
					fmt.Sprintf("https://cdn.frankerfacez.com/emote/%d/3", v.ID),
				},
				Provider: EmoteProviderFFZ,
			})
		}
	}

	data, _ := json.MarshalToString(emotes)
	if err = gCtx.Inst().Redis.SetEX(ctx, "emotes-cached:ffz:"+id, data, time.Minute*30); err != nil {
		logrus.Warn("failed to set emote ffz cache: ", err)
	}

	return emotes, nil
}

type SeventvEmote struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Visibility int    `json:"visibility"`
}

func Get7TV(gCtx global.Context, ctx context.Context, id string) ([]Emote, error) {
	resp, err := gCtx.Inst().Redis.Get(ctx, "emotes-cached:7tv:"+id)
	if err != nil {
		if err != redis.Nil {
			logrus.Warn("failed to fetched cached 7tv emotes: ", err)
		}
	} else {
		emotes := []Emote{}
		if err := json.UnmarshalFromString(resp.(string), &emotes); err != nil {
			logrus.Warn("bad cache value for 7tv emotes: ", err)
		} else {
			return emotes, nil
		}
	}

	seventvGlobal := []SeventvEmote{}
	{
		req, err := http.NewRequestWithContext(ctx, "GET", "https://api.7tv.app/v2/emotes/global", nil)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:7tv:"+id, "[]", time.Second*30)
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:7tv:"+id, "[]", time.Second*30)
			return nil, err
		}
		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:7tv:"+id, "[]", time.Second*30)
			return nil, err
		}

		if err := json.Unmarshal(data, &seventvGlobal); err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:7tv:"+id, "[]", time.Second*30)
			return nil, err
		}
	}

	channel := []SeventvEmote{}
	{
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://api.7tv.app/v2/users/%s/emotes", id), nil)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:7tv:"+id, "[]", time.Second*30)
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:7tv:"+id, "[]", time.Second*30)
			return nil, err
		}
		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:7tv:"+id, "[]", time.Second*30)
			return nil, err
		}

		if err := json.Unmarshal(data, &channel); err != nil {
			_, _ = gCtx.Inst().Redis.SetNX(ctx, "emotes-cached:7tv:"+id, "[]", time.Second*30)
			return nil, err
		}
	}

	emotes := []Emote{}
	for _, v := range seventvGlobal {
		emotes = append(emotes, Emote{
			ID:   v.ID,
			Name: v.Name,
			URLs: []string{
				fmt.Sprintf("https://cdn.7tv.app/emote/%s/1x", v.ID),
				fmt.Sprintf("https://cdn.7tv.app/emote/%s/2x", v.ID),
				fmt.Sprintf("https://cdn.7tv.app/emote/%s/3x", v.ID),
				fmt.Sprintf("https://cdn.7tv.app/emote/%s/4x", v.ID),
			},
			ZeroWidth: v.Visibility&128 != 0,
			Provider:  EmoteProvider7TV,
		})
	}

	for _, v := range channel {
		emotes = append(emotes, Emote{
			ID:   v.ID,
			Name: v.Name,
			URLs: []string{
				fmt.Sprintf("https://cdn.7tv.app/emote/%s/1x", v.ID),
				fmt.Sprintf("https://cdn.7tv.app/emote/%s/2x", v.ID),
				fmt.Sprintf("https://cdn.7tv.app/emote/%s/3x", v.ID),
				fmt.Sprintf("https://cdn.7tv.app/emote/%s/4x", v.ID),
			},
			Provider:  EmoteProvider7TV,
			ZeroWidth: v.Visibility&128 != 0,
		})
	}

	data, _ := json.MarshalToString(emotes)
	if err = gCtx.Inst().Redis.SetEX(ctx, "emotes-cached:7tv:"+id, data, time.Minute*30); err != nil {
		logrus.Warn("failed to set emote ffz cache: ", err)
	}

	return emotes, nil
}

type Emote struct {
	ID        string
	Name      string
	URLs      []string
	ZeroWidth bool
	Provider  EmoteProvider
}

type EmoteProvider string

const (
	EmoteProviderTwitch EmoteProvider = "TWITCH"
	EmoteProviderFFZ    EmoteProvider = "FFZ"
	EmoteProviderBTTV   EmoteProvider = "BTTV"
	EmoteProvider7TV    EmoteProvider = "7TV"
)
