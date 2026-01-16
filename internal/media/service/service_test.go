package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/romariotrain/media-platform/internal/media/models"
)

func TestGetMedia_InvalidID(t *testing.T) {
	ctx := context.Background()
	st := new(StoreMock)
	svc := New(st)

	// Invalid input should be rejected before calling the repository.
	got, err := svc.GetMedia(ctx, uuid.Nil)
	require.ErrorIs(t, err, models.ErrInvalidArgument)
	require.Nil(t, got)
	st.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestGetMedia_Found(t *testing.T) {
	ctx := context.Background()
	st := new(StoreMock)
	svc := New(st)

	id := uuid.New()
	want := &models.Media{
		ID:     id,
		Status: models.UploadedStatus,
	}

	// Service should delegate to the repository and return its result.
	st.On("GetByID", mock.Anything, id).Return(want, nil).Once()

	got, err := svc.GetMedia(ctx, id)
	require.NoError(t, err)
	require.Equal(t, want, got)
	st.AssertExpectations(t)
}

func TestCreateMedia_InvalidArguments(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name      string
		mediaType models.MediaType
		source    string
	}{
		{name: "empty type", mediaType: "", source: "src"},
		{name: "empty source", mediaType: models.Video, source: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := new(StoreMock)
			svc := New(st)

			// Invalid arguments should short-circuit without persisting anything.
			got, err := svc.CreateMedia(ctx, tc.mediaType, tc.source)
			require.ErrorIs(t, err, models.ErrInvalidArgument)
			require.Nil(t, got)
			st.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		})
	}
}

func TestCreateMedia_SetsFieldsAndPersists(t *testing.T) {
	ctx := context.Background()
	st := new(StoreMock)
	svc := New(st)

	fixedID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	fixedTime := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	svc.idGen = func() uuid.UUID { return fixedID }
	svc.clock = func() time.Time { return fixedTime }

	var persisted *models.Media
	st.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			persisted = args.Get(1).(*models.Media)
		}).
		Return(nil).
		Once()

	// Service should set invariants before persisting.
	got, err := svc.CreateMedia(ctx, models.Video, "s3://bucket/file.mp4")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, persisted, got)

	require.Equal(t, fixedID, got.ID)
	require.Equal(t, models.UploadedStatus, got.Status)
	require.Equal(t, models.Video, got.Type)
	require.Equal(t, "s3://bucket/file.mp4", got.Source)
	require.Equal(t, fixedTime, got.CreatedAt)
	require.Equal(t, fixedTime, got.UpdatedAt)
	st.AssertExpectations(t)
}

func TestCreateMedia_RepoErrorPropagated(t *testing.T) {
	ctx := context.Background()
	st := new(StoreMock)
	svc := New(st)

	// Service should pass through repository errors to the caller.
	st.On("Create", mock.Anything, mock.Anything).Return(models.ErrConflict).Once()

	got, err := svc.CreateMedia(ctx, models.Video, "src")
	require.ErrorIs(t, err, models.ErrConflict)
	require.Nil(t, got)
	st.AssertExpectations(t)
}
