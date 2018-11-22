package main

import (
	"errors"
	"time"
)

var (
	gitCommit = ""
)

const (
	defaultWorkingDir = "."

	//MaxDuration is the maximum duration time.Duration can take
	MaxDuration time.Duration = (1 << 63) - 1
)

var ( //error constants
	//ErrorUnauthorized is the error reporting an authentification error
	ErrorUnauthorized = errors.New("CheckAuth: client didn't provide correct authorization")
)
