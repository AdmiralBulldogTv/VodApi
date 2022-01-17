package twitch

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AdmiralBulldogTv/VodApi/src/global"
	"github.com/go-redis/redis/v8"
	jsoniter "github.com/json-iterator/go"
	"github.com/nicklaw5/helix"
	"github.com/sirupsen/logrus"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func GetEventSubs(gCtx global.Context, ctx context.Context) ([]helix.EventSubSubscription, error) {
	subs := []helix.EventSubSubscription{}
	after := ""

start:
	url := "https://api.twitch.tv/helix/eventsub/subscriptions"
	if after != "" {
		url = url + "?after=" + after
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Client-Id", gCtx.Config().Twitch.ClientID)
	tkn, err := GetAuth(gCtx, ctx)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tkn)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	webhookResp := helix.ManyEventSubSubscriptions{}
	if err := json.Unmarshal(data, &webhookResp); err != nil {
		return nil, err
	}

	subs = append(subs, webhookResp.EventSubSubscriptions...)
	if webhookResp.Pagination.Cursor != "" {
		after = webhookResp.Pagination.Cursor
		goto start
	}

	return subs, nil
}

func CreateEventSub(gCtx global.Context, ctx context.Context, hook helix.EventSubSubscription) (helix.EventSubSubscription, error) {
	data, _ := json.Marshal(hook)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.twitch.tv/helix/eventsub/subscriptions", bytes.NewReader(data))
	if err != nil {
		return helix.EventSubSubscription{}, err
	}

	auth, err := GetAuth(gCtx, ctx)
	if err != nil {
		return helix.EventSubSubscription{}, err
	}
	req.Header.Set("Authorization", "Bearer "+auth)
	req.Header.Set("Client-ID", gCtx.Config().Twitch.ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return helix.EventSubSubscription{}, err
	}

	defer resp.Body.Close()
	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return helix.EventSubSubscription{}, err
	}

	err = json.Unmarshal(data, &hook)
	return hook, err
}

func DeleteEventSub(gCtx global.Context, ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", "https://api.twitch.tv/helix/eventsub/subscriptions?id="+id, nil)
	if err != nil {
		return err
	}

	auth, err := GetAuth(gCtx, ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+auth)
	req.Header.Set("Client-ID", gCtx.Config().Twitch.ClientID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		return fmt.Errorf("bad status resp: %d", resp.StatusCode)
	}

	return nil
}

func GetAuth(gCtx global.Context, ctx context.Context) (string, error) {
	token, err := gCtx.Inst().Redis.Get(ctx, "twitch:app-token")
	if err == nil {
		return token.(string), nil
	}

	if err != redis.Nil {
		logrus.Error("failed to query redis: ", err)
	}

	v := url.Values{}

	v.Set("client_id", gCtx.Config().Twitch.ClientID)
	v.Set("client_secret", gCtx.Config().Twitch.ClientSecret)
	v.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://id.twitch.tv/oauth2/token", strings.NewReader(v.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(v.Encode())))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	tokenResp := helix.AccessCredentials{}
	if err := json.Unmarshal(data, &tokenResp); err != nil {
		return "", err
	}

	if err := gCtx.Inst().Redis.SetEX(ctx, "twitch:app-token", tokenResp.AccessToken, time.Second*time.Duration(tokenResp.ExpiresIn)); err != nil {
		logrus.Error("failed to store token in redis: ", err)
	}

	return tokenResp.AccessToken, nil
}
