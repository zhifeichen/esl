// Copyright 2020 chen zhifei
// Please check out LICENSE file for more information about what you CAN and what you CANNOT do!
// Basically in short this is a free software for you to do whatever you want to do BUT copyright must be included!
// I didn't write all of this code so you could say it's yours.
// MIT License

package esl

import "github.com/zhifeichen/log"

// StringInSlice - Will check if string in list. This is equivalent to python if x in []
// @TODO - What the fuck Nevio...
func StringInSlice(str string, list []string) bool {
	for _, value := range list {
		if value == str {
			return true
		}
	}
	return false
}

func Init() {
	logger.Discard()
}

func EnableLog(fileName, level string) {
	logger = log.New(log.NewOptions(
		log.Filename(fileName),
		log.Level(level),
	))
}
