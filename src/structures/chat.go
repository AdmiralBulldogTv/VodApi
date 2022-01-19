package structures

import (
	"time"

	"github.com/AdmiralBulldogTv/VodApi/graph/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Chat struct {
	ID    primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	VodID primitive.ObjectID `json:"vod_id" bson:"vod_id"`

	Twitch ChatTwitch `json:"twitch" bson:"twitch"`

	Timestamp time.Time `json:"timestamp" bson:"timestamp"`

	Content string `json:"content" bson:"content"`

	Badges []ChatBadge `json:"badges" bson:"badges"`
	Emotes []ChatEmote `json:"emotes" bson:"chat_emote"`
}

func (c Chat) ToModel() *model.Chat {
	badges := make([]*model.ChatBadge, len(c.Badges))
	for i, v := range c.Badges {
		badges[i] = v.ToModel()
	}

	emotes := make([]*model.ChatEmote, len(c.Emotes))
	for i, v := range c.Emotes {
		emotes[i] = v.ToModel()
	}

	return &model.Chat{
		ID:        c.ID,
		VodID:     c.VodID,
		Twitch:    c.Twitch.ToModel(),
		Timestamp: c.Timestamp,
		Content:   c.Content,
		Badges:    badges,
		Emotes:    emotes,
	}
}

type ChatTwitch struct {
	ID          string `json:"id" bson:"id"`
	UserID      string `json:"user_id" bson:"user_id"`
	Login       string `json:"login" bson:"login"`
	DisplayName string `json:"display_name" bson:"display_name"`
	Color       string `json:"color" bson:"color"`
}

func (c ChatTwitch) ToModel() *model.ChatTwitch {
	return &model.ChatTwitch{
		ID:          c.ID,
		UserID:      c.UserID,
		Login:       c.Login,
		DisplayName: c.DisplayName,
		Color:       c.Color,
	}
}

type ChatBadge struct {
	Name string   `json:"name" bson:"name"`
	URLs []string `json:"urls" bson:"urls"`
}

func (c ChatBadge) ToModel() *model.ChatBadge {
	return &model.ChatBadge{
		Name: c.Name,
		Urls: c.URLs,
	}
}

type ChatEmote struct {
	Name      string   `json:"name" bson:"name"`
	ZeroWidth bool     `json:"zero_width" bson:"zero_width"`
	URLs      []string `json:"urls" bson:"urls"`
}

func (c ChatEmote) ToModel() *model.ChatEmote {
	return &model.ChatEmote{
		Name: c.Name,
		Urls: c.URLs,
	}
}
