package name

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func SafeConcatNameWithSeparatorAndLength(length int, sep string, name ...string) string {
	fullPath := strings.Join(name, sep)
	if len(fullPath) < length {
		return fullPath
	}
	digest := sha256.Sum256([]byte(fullPath))
	// since we cut the string in the middle, the last char may not be compatible with what is expected in k8s
	// we are checking and if necessary removing the last char
	c := fullPath[length-8]
	if 'a' <= c && c <= 'z' || '0' <= c && c <= '9' {
		return fullPath[0:length-7] + sep + hex.EncodeToString(digest[0:])[0:5]
	}

	return fullPath[0:length-8] + sep + hex.EncodeToString(digest[0:])[0:6]
}

func SafeConcatName(name ...string) string {
	return SafeConcatNameWithSeparatorAndLength(64, "-", name...)
}

func SafeHashConcatName(name ...string) string {
	d := sha256.New()
	for _, part := range name {
		d.Write([]byte(part))
		d.Write([]byte{'\x00'})
	}
	hash := d.Sum(nil)
	suffix := hex.EncodeToString(hash[:])[:8]
	return SafeConcatName(append(name, suffix)...)
}
