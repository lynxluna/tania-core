package repository

import (
	"github.com/Tanibox/tania-server/src/growth/domain"
	"github.com/Tanibox/tania-server/src/growth/storage"
	uuid "github.com/satori/go.uuid"
)

// RepositoryResult is a struct to wrap repository result
// so its easy to use it in channel
type RepositoryResult struct {
	Result interface{}
	Error  error
}

// EventWrapper is used to wrap the event interface with its struct name,
// so it will be easier to unmarshal later
type EventWrapper struct {
	EventName string
	EventData interface{}
}

type CropEventRepository interface {
	Save(uid uuid.UUID, latestVersion int, events []interface{}) <-chan error
}

type CropReadRepository interface {
	Save(cropRead *storage.CropRead) <-chan error
}

func NewCropBatchFromHistory(events []storage.CropEvent) *domain.Crop {
	state := &domain.Crop{}
	for _, v := range events {
		state.Transition(v.Event)
		state.Version++
	}
	return state
}
