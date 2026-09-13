package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Argon2Config struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	KeyLength   uint32
	SaltLength  uint32
}

type PasswordService struct {
	cfg Argon2Config
}

func NewPasswordService(cfg Argon2Config) *PasswordService {
	return &PasswordService{cfg: cfg}
}

func (p *PasswordService) Hash(password string) (string, error) {
	salt := make([]byte, p.cfg.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	key := argon2.IDKey(
		[]byte(password),
		salt,
		p.cfg.Iterations,
		p.cfg.Memory,
		p.cfg.Parallelism,
		p.cfg.KeyLength,
	)

	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		p.cfg.Memory,
		p.cfg.Iterations,
		p.cfg.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func (p *PasswordService) Verify(password, encoded string) (bool, error) {
	params, salt, expected, err := decodeArgon2(encoded)
	if err != nil {
		return false, err
	}

	actual := argon2.IDKey(
		[]byte(password),
		salt,
		params.iterations,
		params.memory,
		params.parallelism,
		uint32(len(expected)),
	)

	if len(actual) != len(expected) {
		return false, nil
	}

	var diff byte
	for i := range actual {
		diff |= actual[i] ^ expected[i]
	}

	return diff == 0, nil
}

type argon2Params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func decodeArgon2(encoded string) (argon2Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return argon2Params{}, nil, nil, errors.New("invalid argon2id hash format")
	}

	var params argon2Params
	for _, item := range strings.Split(parts[3], ",") {
		kv := strings.SplitN(item, "=", 2)
		if len(kv) != 2 {
			return argon2Params{}, nil, nil, errors.New("invalid argon2 parameters")
		}
		switch kv[0] {
		case "m":
			n, err := strconv.ParseUint(kv[1], 10, 32)
			if err != nil {
				return argon2Params{}, nil, nil, errors.New("invalid argon2 memory")
			}
			params.memory = uint32(n)
		case "t":
			n, err := strconv.ParseUint(kv[1], 10, 32)
			if err != nil {
				return argon2Params{}, nil, nil, errors.New("invalid argon2 iterations")
			}
			params.iterations = uint32(n)
		case "p":
			n, err := strconv.ParseUint(kv[1], 10, 8)
			if err != nil {
				return argon2Params{}, nil, nil, errors.New("invalid argon2 parallelism")
			}
			params.parallelism = uint8(n)
		}
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argon2Params{}, nil, nil, errors.New("invalid argon2 salt")
	}

	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argon2Params{}, nil, nil, errors.New("invalid argon2 hash")
	}

	if params.memory == 0 || params.iterations == 0 || params.parallelism == 0 || len(salt) == 0 || len(expected) == 0 {
		return argon2Params{}, nil, nil, errors.New("invalid argon2 parameters")
	}

	return params, salt, expected, nil
}
