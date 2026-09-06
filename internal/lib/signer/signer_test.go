package signer_test

import (
	"photo-viewer-server/internal/lib/signer"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testPhotoMetadata struct {
	photoUuid uuid.UUID
	ownerUuid uuid.UUID
	rawFile string
	mediumFile string
	smallFile string
}

func newTestSigner(ttl time.Duration) *signer.KeySigner{
	return signer.NewKeySigner("1234567890123456789012345678901234", ttl)
}

func newTestPhotoMetadata() *testPhotoMetadata {
	return &testPhotoMetadata{
		photoUuid: uuid.New(),
		ownerUuid: uuid.New(),
		rawFile: uuid.New().String(),
		mediumFile: uuid.New().String(),
		smallFile: uuid.New().String(),
	}
}

func TestSign_ReturnsNonEmptyToken(t *testing.T) {
	s := newTestSigner(time.Hour)
	m := newTestPhotoMetadata()

	key, err := s.Sign(m.photoUuid, m.ownerUuid, m.rawFile, m.mediumFile, m.smallFile)

	require.NoError(t, err)
	assert.NotEmpty(t, key)

	assert.Equal(t, 1, strings.Count(key, "."), "token should have format payload.signature")
}

func TestValidation_ValidateRightSignedData(t *testing.T) {
	s := newTestSigner(time.Hour)
	m := newTestPhotoMetadata()

	key, err := s.Sign(m.photoUuid, m.ownerUuid, m.rawFile, m.mediumFile, m.smallFile)

	require.NoError(t, err)

	payload, err := s.Validate(key, m.photoUuid)
	
	require.NoError(t, err)
	assert.Equal(t, m.photoUuid, payload.PhotoUuid)
	assert.Equal(t, m.ownerUuid, payload.OwnerUuid)
	assert.Equal(t, m.rawFile, payload.PhotoRawFile)
	assert.Equal(t, m.mediumFile, payload.PhotoMediumFile)
	assert.Equal(t, m.smallFile, payload.PhotoSmallFile)
}
