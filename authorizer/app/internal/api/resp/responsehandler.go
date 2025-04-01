package resp

import (
	"apikey/internal/model"
	"net/http"
	"time"
)

const TimeLayout = "2006-01-02 15:04:05"

type TemplateResponse struct {
	model.BaseResponse
	Data interface{} `json:"data"`
}

func NewTemplateResponse(data interface{}) TemplateResponse {
	return TemplateResponse{
		BaseResponse: model.BaseResponse{
			Status:    "success",
			Code:      http.StatusOK,
			Datetime:  time.Now().Format(TimeLayout),
			Timestamp: time.Now().Unix(),
		},
		Data: data,
	}
}
