package types

import appErr "github.com/iac-studio/engine/pkg/errors"

func FromAppError(err error) *APIError {
    if err == nil {
        return nil
    }
    code := string(appErr.CodeUnknown)
    if e, ok := err.(*appErr.AppError); ok {
        code = string(e.Code)
        return &APIError{Code: code, Message: e.Message}
    }
    return &APIError{Code: code, Message: err.Error()}
}


