package types

type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *APIError   `json:"error,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"`
}

type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

type Meta struct {
    RequestID string `json:"request_id,omitempty"`
    Page      int    `json:"page,omitempty"`
    PageSize  int    `json:"page_size,omitempty"`
    Total     int64  `json:"total,omitempty"`
}


