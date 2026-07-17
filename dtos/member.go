package dtos

import (
	"mime/multipart"
	"time"

	"github.com/username/myproject/models"
)

type MemberDTO struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	Email      string     `json:"email"`
	MemberID   string     `json:"member_id"`
	CustomerID string     `json:"customer_id"`
	Role       string     `json:"role,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	CreatedBy  string     `json:"created_by,omitempty"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
	UpdatedBy  *int       `json:"updated_by,omitempty"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
	Deleted    bool       `json:"deleted,omitempty"`
	DeletedBy  *int       `json:"deleted_by,omitempty"`
	Status     string     `json:"status_acc"`
}

type CreateMemberRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
}

type MemberEvent struct {
	EventType string `json:"event_type"` // e.g. "member.created"
	MemberID  string `json:"member_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

type UpdateStatusRequest struct {
	CustomerID string `json:"customer_id"`
	Status     string `json:"status" binding:"required"`
}

type LoginRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	UserID      string `json:"user_id,omitempty"`
	Role        string `json:"role,omitempty"`
}

type MemberBatchRequest struct {
	File       *multipart.FileHeader `form:"file"`
	Department string                `form:"department"`
	BatchName  string                `form:"batch_name"`
}

func ToMemberDTO(m *models.Member) *MemberDTO {
	dto := &MemberDTO{
		ID:        m.ID,
		Name:      m.Name,
		Email:     m.Email,
		MemberID:  m.MemberID,
		CreatedAt: m.CreatedAt,
		CreatedBy: m.CreatedBy,
		Status:    m.Status,
	}

	if !m.UpdatedAt.IsZero() {
		dto.UpdatedAt = &m.UpdatedAt
	}
	if m.UpdatedBy != 0 {
		dto.UpdatedBy = &m.UpdatedBy
	}
	if !m.DeletedAt.IsZero() {
		dto.DeletedAt = &m.DeletedAt
	}
	if m.DeletedBy != 0 {
		dto.DeletedBy = &m.DeletedBy
	}

	return dto
}

type PermissionRequest struct {
	Method        string `json:"method"`
	Path          string `json:"path"`
	Authorization string `json:"authorization"`
}

type SendMailRequest struct {
	To      string `json:"to" binding:"required,email"`
	Subject string `json:"subject"`
	Message string `json:"message" binding:"required"`
}
