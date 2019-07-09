package util

import (
	"fmt"
	"strings"
	"crypto/sha256"
	"github.com/satori/go.uuid"
)

func GenerateUpperCaseUuid(seed string) (string, error) {
	hash := sha256.Sum256([]byte(seed))
	uuid, err := uuid.FromBytes(hash[0:16])
	if err != nil {
		return "", err
	}

	return strings.ToUpper(fmt.Sprintf("%s", uuid)), nil
}