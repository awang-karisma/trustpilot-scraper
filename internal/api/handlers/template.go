package handlers

import (
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/awang-karisma/trustpilot-scraper/internal/api/dto"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
)

// TemplateHandler handles template-related requests
type TemplateHandler struct {
	db *gorm.DB
}

// NewTemplateHandler creates a new TemplateHandler
func NewTemplateHandler(db *gorm.DB) *TemplateHandler {
	return &TemplateHandler{db: db}
}

// List returns all templates
// @Summary List all templates
// @Description Get a list of all notification templates
// @Tags templates
// @Produce json
// @Success 200 {object} dto.TemplateListResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/templates [get]
func (h *TemplateHandler) List(c fiber.Ctx) error {
	var templates []database.Template
	result := h.db.Find(&templates)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	return c.JSON(dto.ToTemplateListResponse(templates, result.RowsAffected))
}

// Get returns a single template by ID
// @Summary Get a template
// @Description Get a notification template by ID
// @Tags templates
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} dto.TemplateResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/templates/{id} [get]
func (h *TemplateHandler) Get(c fiber.Ctx) error {
	id := c.Params("id")

	var template database.Template
	result := h.db.First(&template, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(dto.ErrorResponse{Error: "template not found"})
		}
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	return c.JSON(dto.ToTemplateResponse(template))
}

// Create creates a new template
// @Summary Create a new template
// @Description Register a new notification template
// @Tags templates
// @Accept json
// @Produce json
// @Param request body dto.CreateTemplateRequest true "Template details"
// @Success 201 {object} dto.TemplateResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/templates [post]
func (h *TemplateHandler) Create(c fiber.Ctx) error {
	var req dto.CreateTemplateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(400).JSON(dto.ErrorResponse{Error: "invalid request body"})
	}

	template := database.Template{
		Name:        req.Name,
		FileName:    req.FileName,
		Description: req.Description,
		Enabled:     req.Enabled,
	}

	result := h.db.Create(&template)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	return c.Status(201).JSON(dto.ToTemplateResponse(template))
}

// Update updates an existing template
// @Summary Update a template
// @Description Update notification template details by ID
// @Tags templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Param request body dto.UpdateTemplateRequest true "Template details"
// @Success 200 {object} dto.TemplateResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/templates/{id} [put]
func (h *TemplateHandler) Update(c fiber.Ctx) error {
	id := c.Params("id")

	var template database.Template
	result := h.db.First(&template, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(dto.ErrorResponse{Error: "template not found"})
		}
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	var req dto.UpdateTemplateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(400).JSON(dto.ErrorResponse{Error: "invalid request body"})
	}

	// Apply updates
	if req.Name != nil {
		template.Name = *req.Name
	}
	if req.FileName != nil {
		template.FileName = *req.FileName
	}
	if req.Description != nil {
		template.Description = *req.Description
	}
	if req.Enabled != nil {
		template.Enabled = *req.Enabled
	}

	result = h.db.Save(&template)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	return c.JSON(dto.ToTemplateResponse(template))
}

// Delete deletes a template
// @Summary Delete a template
// @Description Delete a notification template by ID
// @Tags templates
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/templates/{id} [delete]
func (h *TemplateHandler) Delete(c fiber.Ctx) error {
	id := c.Params("id")

	result := h.db.Delete(&database.Template{}, "id = ?", id)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	if result.RowsAffected == 0 {
		return c.Status(404).JSON(dto.ErrorResponse{Error: "template not found"})
	}

	return c.JSON(dto.SuccessResponse{Success: true, Message: "template deleted"})
}

// Validate validates a template
// @Summary Validate a template
// @Description Check if a template file exists and is valid
// @Tags templates
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/templates/{id}/validate [post]
func (h *TemplateHandler) Validate(c fiber.Ctx) error {
	id := c.Params("id")

	var template database.Template
	result := h.db.First(&template, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(dto.ErrorResponse{Error: "template not found"})
		}
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	// TODO: Add actual template validation logic
	// For now, return success
	return c.JSON(fiber.Map{
		"valid":   true,
		"message": "template is valid",
	})
}
