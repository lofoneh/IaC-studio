package handlers

import "net/http"

type GraphsHandler struct{}

func NewGraphsHandler() *GraphsHandler { return &GraphsHandler{} }

func (h *GraphsHandler) Save(w http.ResponseWriter, r *http.Request)  { w.WriteHeader(201) }
func (h *GraphsHandler) Load(w http.ResponseWriter, r *http.Request)  { w.WriteHeader(200) }


