package signer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type FileKeyPayload struct {
	PhotoUuid uuid.UUID		  `json:"p"`
	OwnerUuid uuid.UUID		  `json:"o"`
	PhotoRawFile string 	  `json:"pr"`
	PhotoMediumFile string 	`json:"pm"`
	PhotoSmallFile string 	`json:"ps"`
	Expires int64					  `json:"exp"`
}

type KeySigner struct {
	secret []byte
	ttl time.Duration
}

func NewKeySigner(secret string, ttl time.Duration) *KeySigner {
	return &KeySigner{secret: []byte(secret), ttl: ttl}
}

func (ks *KeySigner) Sign(photoUuid, ownerUuid uuid.UUID, photoRaw, photoMedium, photoSmall string) (string, error) {
	payload := &FileKeyPayload{
		PhotoUuid: photoUuid,
		OwnerUuid: ownerUuid,
		PhotoRawFile: photoRaw,
		PhotoMediumFile: photoMedium,
		PhotoSmallFile: photoSmall,
	  Expires: time.Now().Add(ks.ttl).Unix(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	encoded := base64.RawURLEncoding.EncodeToString(data)
	signature := ks.computeHMAC(encoded)

	return fmt.Sprintf("%s.%s", encoded, signature), nil
}

func (ks *KeySigner) Validate(token string, expectedPhotoUuid uuid.UUID) (*FileKeyPayload, error) {

	var encoded, signature string
	if _, err := fmt.Sscanf(token, "%s.%s", &encoded, &signature); err != nil {
		parts := splitToken(token)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid token format")
		}
		encoded, signature = parts[0], parts[1]
	}

	expectedHMAC := ks.computeHMAC(encoded)
	if !hmac.Equal([]byte(expectedHMAC), []byte(signature)) {
		return nil, fmt.Errorf("invalid signature")
	}

	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding")
	}

	var payload FileKeyPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("invalid payload")
	}

	if payload.Expires < time.Now().Unix() {
		return nil, fmt.Errorf("token expired")
	}

	if payload.PhotoUuid != expectedPhotoUuid {
		return nil, fmt.Errorf("photo uuid missmatch")
	}

	return &payload, nil
}

func (ks *KeySigner) computeHMAC(data string) string {
	hash := hmac.New(sha256.New, ks.secret)
	hash.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}

func splitToken(token string) []string {
	for i := len(token) - 1; i >= 0; i-- {
		if token[i] == '.' {
			return []string{token[:i], token[i+1:]}
		}
	}
	return nil
}
