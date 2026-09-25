package service

import "errors"

var ErrSendFailed = errors.New("the provider did not accept the message")
var ErrTemplateUnusable = errors.New("the template named by this message cannot be used")
var ErrAttemptCannotBeRecorded = errors.New("the attempt could not be recorded")
