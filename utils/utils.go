package utils

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Status  int         `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func (data *Response) ValidResponse(w http.ResponseWriter) {
	w.WriteHeader(data.Status)
	response := Response{
		Status:  data.Status,
		Message: data.Message,
		Data:    data.Data,
		Error:   data.Error,
	}
	json.NewEncoder(w).Encode(response)
}
