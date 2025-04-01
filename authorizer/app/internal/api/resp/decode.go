package resp

import (
	"apikey/internal/errormap"
	"apikey/pkg/errorx"
	"encoding/json"
	"strconv"
)

func DecodeBody(data string, v interface{}) error {
	err := json.Unmarshal([]byte(data), v)
	if err != nil {
		return errorx.WrapErrorf(err, errormap.CodeInvalidArgument, "invalid request body")
	}
	return nil
}

func DecodePath(str string) (int64, error) {
	id, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, errorx.WrapErrorf(err, errormap.CodeInvalidArgument, "invalid path")
	}
	return id, nil
}
