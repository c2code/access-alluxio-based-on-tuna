package utils

import (
	uuid "github.com/satori/go.uuid"
)

// NewUUID use uuid
func NewUUID() string {
	return uuid.NewV4().String()
}
