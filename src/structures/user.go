package structures

import (
	"github.com/AdmiralBulldogTv/VodApi/graph/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID primitive.ObjectID `json:"id" bson:"_id,omitempty"`

	Twitch UserTwitch `json:"twitch" bson:"twitch"`

	StreamKey string `json:"stream_key" bson:"stream_key"`
}

func (u User) ToModel() *model.User {
	return &model.User{
		ID:     u.ID,
		Twitch: u.Twitch.ToModel(),
	}
}

type UserTwitch struct {
	ID             string `json:"id" bson:"id"`
	Login          string `json:"login" bson:"login"`
	DisplayName    string `json:"display_name" bson:"display_name"`
	ProfilePicture string `json:"profile_picture" bson:"profile_picture"`
}

func (u UserTwitch) ToModel() *model.UserTwitch {
	return &model.UserTwitch{
		ID:             u.ID,
		Login:          u.Login,
		DisplayName:    u.DisplayName,
		ProfilePicture: u.ProfilePicture,
	}
}
