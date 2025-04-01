package errormap

import (
	"apikey/pkg/errorx"
)

var (
	ErrInvalidParams     = errorx.NewErrorf(CodeInvalidArgument, "invalid params")
	ErrAuthentication    = errorx.NewErrorf(CodeUnauthorized, "authentication failed")
	ErrUnauthorized      = errorx.NewErrorf(CodeUnauthorized, "invalid access token")
	ErrInvalidCommerceID = errorx.NewErrorf(CodeInvalidArgument, "invalid  id")
	ErrCommerceNotFound  = errorx.NewErrorf(CodeNotFound, "company not found")
	ErrCommerceExist     = errorx.NewErrorf(CodePrecondition, "company already exists")
	ErrInactiveCommerce  = errorx.NewErrorf(CodeInvalidArgument, "inactive company")
	ErrNoRows            = errorx.NewErrorf(CodeNoRows, "API KEY inválida o inexistente")
	ErrDuplicateKey      = errorx.NewErrorf(DuplicateKey, " id already exists")
)

const (
	CodeUnknown errorx.ErrorCode = iota
	CodeInvalidArgument
	CodeInvalidToken
	CodeNotFound
	CodeNoRows
	CodePrecondition
	CodeDecode
	CodeUnauthorized
	DuplicateKey
)
