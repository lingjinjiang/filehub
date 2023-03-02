package common

import "errors"

const BLOCK_SIZE int64 = 50 * 1024 * 1024

var Exist = errors.New("file exist")
