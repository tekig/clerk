package recorder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tekig/clerk/internal/repository/mock"
	"go.uber.org/mock/gomock"
)

func TestRecorder_dont_write_empty_block(t *testing.T) {
	ctrl := gomock.NewController(t)

	storage := mock.NewMockStorage(ctrl)

	recorder, err := NewRecorder(
		storage,
		nil,
	)
	assert.NoError(t, err)

	assert.NoError(t, recorder.Shutdown())
}
