package utils

import (
	"errors"
	"fmt"
)

func Wrap(err error, message string) error {
	return errors.New(message + " > " + err.Error())
}

func Error(prefix, message string) error {
	return errors.New(fmt.Sprint("[", prefix, "] ", message))
}
