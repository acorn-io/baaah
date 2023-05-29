package tester

import (
	"encoding/json"
	"math/rand"
)

const (
	characters  = "bcdfghjklmnpqrstvwxz2456789"
	tokenLength = 54
)

func generate(obj any) (string, error) {
	d, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	r := rand.New(rand.NewSource(int64(len(d))))
	token := make([]byte, tokenLength)
	for i := range token {
		token[i] = characters[r.Int63n(int64(len(characters)))]
	}
	return string(token), nil
}
