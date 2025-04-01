package model

type AppSyncEvent struct {
	Operation      string            `json:"operation"`
	Input          string            `json:"input"`
	PathParameters map[string]string `json:"pathParameters"`
}

type GetByCompany struct {
	Id int64 `json:"id"`
}
