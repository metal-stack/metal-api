package service

import "go.uber.org/zap"

var testlogger = zap.NewNop()
var emptyResult = map[string]interface{}{}
