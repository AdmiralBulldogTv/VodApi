package structures

import (
	"time"

	"github.com/AdmiralBulldogTv/VodApi/graph/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Vod struct {
	ID     primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID primitive.ObjectID `json:"user_id" bson:"user_id"`

	Title string `json:"title" bson:"title"`

	Categories []VodCategory `json:"categories" bson:"categories"`

	State      VodState      `json:"vod_state" bson:"vod_state"`
	Visibility VodVisibility `json:"vod_visibility" bson:"vod_visibility"`

	Variants []VodVariant `json:"variants" bson:"variants"`

	Thumbnail struct {
		Static   string `json:"static" bson:"static"`
		Animated string `json:"animated" bson:"animated"`
	} `json:"thumbnail" bson:"thumbnail"`

	StartedAt time.Time `json:"started_at" bson:"started_at"`
	EndedAt   time.Time `json:"ended_at" bson:"ended_at"`
}

func (v Vod) ToModel() *model.Vod {
	categories := make([]*model.VodCategory, len(v.Categories))
	for i, v := range v.Categories {
		categories[i] = v.ToModel()
	}
	variants := make([]*model.VodVariant, len(v.Variants))
	for i, v := range v.Variants {
		variants[i] = v.ToModel()
	}
	var endedAt *time.Time
	if !v.EndedAt.IsZero() {
		endedAt = &v.EndedAt
	}
	return &model.Vod{
		ID:         v.ID,
		UserID:     v.UserID,
		Title:      v.Title,
		Categories: categories,
		Variants:   variants,
		Thumbnails: &model.VodThumbnails{
			Static:   v.Thumbnail.Static,
			Animated: v.Thumbnail.Animated,
		},
		State:      v.State.ToModel(),
		Visibility: v.Visibility.ToModel(),
		StartedAt:  v.StartedAt,
		EndedAt:    endedAt,
	}
}

type VodCategory struct {
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
	Name      string    `json:"name" bson:"name"`
	ID        string    `json:"id" bson:"id"`
	URL       string    `json:"url" bson:"url"`
}

func (v VodCategory) ToModel() *model.VodCategory {
	return &model.VodCategory{
		Timestamp: v.Timestamp,
		Name:      v.Name,
		ID:        v.ID,
		URL:       v.URL,
	}
}

type VodState int32

const (
	VodStateLive VodState = iota
	VodStateQueued
	VodStateProcessing
	VodStateReady
	VodStateStorage
	VodStateFailed
	VodStateCanceled
)

func (v VodState) ToModel() model.VodState {
	switch v {
	case VodStateLive:
		return model.VodStateLive
	case VodStateQueued:
		return model.VodStateQueued
	case VodStateProcessing:
		return model.VodStateProcessing
	case VodStateReady:
		return model.VodStateReady
	case VodStateStorage:
		return model.VodStateStorage
	case VodStateFailed:
		return model.VodStateFailed
	case VodStateCanceled:
		return model.VodStateCanceled
	}

	return "unknown"
}

type VodVisibility int32

const (
	VodVisibilityPublic VodVisibility = iota
	VodVisibilityDeleted
)

func (v VodVisibility) ToModel() model.VodVisibility {
	switch v {
	case VodVisibilityPublic:
		return model.VodVisibilityPublic
	case VodVisibilityDeleted:
		return model.VodVisibilityDeleted
	}

	return "unknown"
}

type VodVariant struct {
	Name    string `json:"name" bson:"name"`
	Width   int    `json:"width" bson:"width"`
	Height  int    `json:"height" bson:"height"`
	FPS     int    `json:"fps" bson:"fps"`
	Bitrate int    `json:"bitrate" bson:"bitrate"`
	Ready   bool   `json:"ready" bson:"ready"`
}

func (v VodVariant) ToModel() *model.VodVariant {
	return &model.VodVariant{
		Name:    v.Name,
		Width:   v.Width,
		Height:  v.Height,
		Fps:     v.FPS,
		Bitrate: v.Bitrate,
		Ready:   v.Ready,
	}
}
