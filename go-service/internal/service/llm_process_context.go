package service

import (
	"strings"

	"github.com/google/uuid"

	"auraoa/go-service/internal/pkg/ai"
)

func bindLLMProcessContext(req *ai.ChatRequest, processID, processTitle string, businessLogID uuid.UUID) {
	if req == nil {
		return
	}
	req.ProcessID = strings.TrimSpace(processID)
	req.ProcessTitle = strings.TrimSpace(processTitle)
	if businessLogID != uuid.Nil {
		id := businessLogID
		req.BusinessLogID = &id
	}
}
