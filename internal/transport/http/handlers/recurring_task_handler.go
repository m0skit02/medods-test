package handlers

import (
	"errors"
	"net/http"
	"time"

	recurringtaskdomain "example.com/taskservice/internal/domain/recurringtask"
	recurringtaskusecase "example.com/taskservice/internal/usecase/recurringtask"
)

type RecurringTaskHandler struct {
	usecase recurringtaskusecase.Usecase
}

func NewRecurringTaskHandler(usecase recurringtaskusecase.Usecase) *RecurringTaskHandler {
	return &RecurringTaskHandler{usecase: usecase}
}

func (h *RecurringTaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req recurringTaskMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	created, err := h.usecase.Create(r.Context(), recurringtaskusecase.CreateInput{
		Title:       req.Title,
		Description: req.Description,
		Active:      req.Active,
		Recurrence:  toRecurringRecurrenceInput(req.Recurrence),
	})
	if err != nil {
		writeRecurringTaskUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newRecurringTaskDTO(created))
}

func (h *RecurringTaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	template, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		writeRecurringTaskUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newRecurringTaskDTO(template))
}

func (h *RecurringTaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req recurringTaskMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	updated, err := h.usecase.Update(r.Context(), id, recurringtaskusecase.UpdateInput{
		Title:       req.Title,
		Description: req.Description,
		Active:      req.Active,
		Recurrence:  toRecurringRecurrenceInput(req.Recurrence),
	})
	if err != nil {
		writeRecurringTaskUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newRecurringTaskDTO(updated))
}

func (h *RecurringTaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.usecase.Delete(r.Context(), id); err != nil {
		writeRecurringTaskUsecaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *RecurringTaskHandler) List(w http.ResponseWriter, r *http.Request) {
	templates, err := h.usecase.List(r.Context())
	if err != nil {
		writeRecurringTaskUsecaseError(w, err)
		return
	}

	response := make([]recurringTaskDTO, 0, len(templates))
	for i := range templates {
		response = append(response, newRecurringTaskDTO(&templates[i]))
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *RecurringTaskHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var req recurringTaskGenerateDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	tasks, err := h.usecase.Generate(r.Context(), recurringtaskusecase.GenerateInput{
		FromDate: req.FromDate,
		ToDate:   req.ToDate,
	})
	if err != nil {
		writeRecurringTaskUsecaseError(w, err)
		return
	}

	responseTasks := make([]taskDTO, 0, len(tasks))
	for i := range tasks {
		responseTasks = append(responseTasks, newTaskDTO(&tasks[i]))
	}

	writeJSON(w, http.StatusOK, recurringTaskGenerateResponseDTO{
		FromDate:       req.FromDate.UTC(),
		ToDate:         req.ToDate.UTC(),
		GeneratedCount: len(responseTasks),
		Tasks:          responseTasks,
	})
}

func writeRecurringTaskUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, recurringtaskdomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, recurringtaskusecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}

func toRecurringRecurrenceInput(dto recurringRecurrenceDTO) recurringtaskusecase.RecurrenceInput {
	return recurringtaskusecase.RecurrenceInput{
		Type:         dto.Type,
		StartDate:    dto.StartDate,
		EndDate:      dto.EndDate,
		IntervalDays: dto.IntervalDays,
		DayOfMonth:   dto.DayOfMonth,
		Dates:        append([]time.Time(nil), dto.Dates...),
	}
}
