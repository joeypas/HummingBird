package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

func issueToken(uid uuid.UUID) (string, error) {
	exp := time.Now().Add(72 * time.Hour).Unix()
	payload := uid.String() + ":" + strconv.FormatInt(exp, 10)
	h := hmac.New(sha256.New, jwtSecret)
	h.Write([]byte(payload))
	sig := hex.EncodeToString(h.Sum(nil))
	token := payload + "." + sig
	return base64.RawURLEncoding.EncodeToString([]byte(token)), nil
}

func parseToken(tok string) (uuid.UUID, error) {
	data, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		return uuid.Nil, err
	}
	parts := strings.Split(string(data), ".")
	if len(parts) != 2 {
		return uuid.Nil, errors.New("invalid token")
	}
	payload, sig := parts[0], parts[1]
	h := hmac.New(sha256.New, jwtSecret)
	h.Write([]byte(payload))
	expected := hex.EncodeToString(h.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return uuid.Nil, errors.New("invalid token")
	}
	fields := strings.Split(payload, ":")
	if len(fields) != 2 {
		return uuid.Nil, errors.New("invalid token")
	}
	uidStr, expStr := fields[0], fields[1]
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return uuid.Nil, errors.New("invalid token")
	}
	if time.Now().Unix() > exp {
		return uuid.Nil, errors.New("expired token")
	}
	return uuid.FromString(uidStr)
}
