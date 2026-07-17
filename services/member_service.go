package services

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/jmoiron/sqlx"
	"github.com/username/myproject/assets/fonts"
	"github.com/username/myproject/configs"
	"github.com/username/myproject/dtos"
	apierrors "github.com/username/myproject/errors"
	"github.com/username/myproject/models"
	"github.com/username/myproject/repositories"
)

type MemberService interface {
	CreateMember(member *dtos.MemberDTO) (*dtos.MemberDTO, *apierrors.CommonResponse)
	UpdateMemberStatus(memberID string, status string) *apierrors.CommonResponse // ← add this
	ExportPdf() ([]byte, *apierrors.CommonResponse)
	CreateMemberBatch(members []*dtos.MemberDTO) error
	GetByStatusAndDate(status string) ([]dtos.MemberDTO, *apierrors.CommonResponse)
	Login(email string) (*dtos.LoginResponse, *apierrors.CommonResponse)
	RetryAllFailedMembers()
}

type memberService struct {
	// You can add dependencies like repositories here
	memberRepo     repositories.MemberRepository
	memberProducer configs.MemberProducer
	db             *sqlx.DB
	signingKey     []byte
	issuer         string
	tokenTTL       time.Duration
}

func NewMemberService(memberRepo repositories.MemberRepository, memberProducer configs.MemberProducer, db *sqlx.DB) MemberService {
	return &memberService{memberRepo: memberRepo, memberProducer: memberProducer, db: db, signingKey: nil, issuer: "", tokenTTL: 24 * time.Hour}
}

func NewMemberServiceWithAuth(memberRepo repositories.MemberRepository, memberProducer configs.MemberProducer, db *sqlx.DB, signingKey []byte, issuer string, tokenTTL time.Duration) MemberService {
	return &memberService{memberRepo: memberRepo, memberProducer: memberProducer, db: db, signingKey: signingKey, issuer: issuer, tokenTTL: tokenTTL}
}
func (s *memberService) CreateMember(memberDTO *dtos.MemberDTO) (*dtos.MemberDTO, *apierrors.CommonResponse) {
	// TEMPORARY TEST CODE — remove after pool testing
	existing, err := s.memberRepo.GetByStatus("ACTIVE") // or a dedicated GetByEmail method
	if err != nil {
		log.Printf("❌ repo error: %v", err)
		return nil, &apierrors.ErrorInternal
	}
	for _, m := range existing {
		if m.Email == memberDTO.Email {
			return nil, &apierrors.ErrorInternal // whatever your error type is
		}
	}

	newMember, err := s.memberRepo.CreateMember(&models.Member{
		Name:      memberDTO.Name,
		Email:     memberDTO.Email,
		CreatedAt: time.Now(),
		CreatedBy: "coco",
	})

	if err != nil {
		log.Printf("❌ repo error: %v", err)
		return nil, &apierrors.ErrorInternal
	}
	if newMember == nil {
		log.Printf("❌ newMember is nil")
		return nil, &apierrors.ErrorInternal
	}

	log.Printf("✅ member created: %+v", newMember)

	if s.memberProducer != nil {
		if err := s.memberProducer.PublishMemberCreated(&dtos.MemberEvent{
			EventType: "member.created",
			MemberID:  newMember.MemberID,
			Name:      newMember.Name,
			Email:     newMember.Email,
		}); err != nil {
			log.Printf("❌ kafka publish error: %v", err)
		}
	} else {
		log.Printf("⚠️ member producer not configured, skipping kafka publish")
	}

	dtos := dtos.ToMemberDTO(newMember)

	return dtos, nil
}

// option B — retry all failed members (for a scheduler)
func (s *memberService) Login(email string) (*dtos.LoginResponse, *apierrors.CommonResponse) {
	member, err := s.memberRepo.GetByEmail(email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			resp := apierrors.ErrorUnauthorized
			resp.Message = "invalid credentials"
			return nil, &resp
		}

		log.Printf("❌ repo error while looking up email %s: %v", email, err)
		return nil, &apierrors.ErrorInternal
	}

	token, err := NewToken(s.signingKey, s.issuer, member.MemberID, member.Role, s.tokenTTL)
	if err != nil {
		log.Printf("❌ token issuance failed: %v", err)
		return nil, &apierrors.ErrorInternal
	}

	return &dtos.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.tokenTTL.Seconds()),
		UserID:      member.MemberID,
		Role:        member.Role,
	}, nil
}

func (s *memberService) RetryAllFailedMembers() {
	members, err := s.memberRepo.GetByStatus("FAILED")
	if err != nil {
		log.Printf("❌ failed to fetch failed members: %v", err)
		return
	}
	if len(members) == 0 {
		log.Println("✅ no failed members to retry")
		return
	}

	for _, member := range members {
		if err := s.retryMember(member); err != nil {
			log.Printf("❌ failed to retry member %s: %v", member.MemberID, err)
			continue
		}
		log.Printf("✅ re-queued member %s", member.MemberID)
	}
}
func (s *memberService) retryMember(member *models.Member) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("❌ panic recovered, transaction rolled back for member %s: %v", member.MemberID, r)
			panic(r)
		}
	}()

	if err := s.memberRepo.UpdateMemberStatus(tx, member.MemberID, "PENDING"); err != nil {
		tx.Rollback()
		return fmt.Errorf("update status: %w", err)
	}

	if err := s.memberProducer.PublishMemberCreated(&dtos.MemberEvent{
		EventType: "member.created",
		MemberID:  member.MemberID,
		Name:      member.Name,
		Email:     member.Email,
	}); err != nil {
		tx.Rollback()
		return fmt.Errorf("publish event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// app1/services/member_service.go
func (s *memberService) UpdateMemberStatus(memberID string, status string) *apierrors.CommonResponse {
	tx, err := s.db.Beginx()
	if err != nil {
		log.Printf("❌ failed to begin transaction: %v", err)
		return &apierrors.ErrorInternal
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("❌ panic recovered, transaction rolled back: %v", r)
			panic(r) // re-panic after cleanup
		}
	}()

	if err := s.memberRepo.UpdateMemberStatus(tx, memberID, status); err != nil {
		tx.Rollback()
		log.Printf("❌ repo error, transaction rolled back: %v", err)
		return &apierrors.ErrorInternal
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		log.Printf("❌ failed to commit transaction: %v", err)
		return &apierrors.ErrorInternal
	}

	log.Printf("✅ member status updated: id=%s status=%s", memberID, status)
	return nil
}
func (s *memberService) CreateMemberBatch(members []*dtos.MemberDTO) error {
	tx, err := s.db.Beginx()

	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Convert DTOs → models (same as CreateMember does)
	var memberModels []*models.Member
	for _, m := range members {
		memberModels = append(memberModels, &models.Member{
			Name:      m.Name,
			Email:     m.Email,
			CreatedAt: time.Now(),
			CreatedBy: "admin",
		})
	}

	createdMembers, err := s.memberRepo.CreateMemberBatch(tx, memberModels)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	if s.memberProducer != nil {
		for _, member := range createdMembers {
			if err := s.memberProducer.PublishMemberCreated(&dtos.MemberEvent{
				EventType: "member.created",
				MemberID:  member.MemberID,
				Name:      member.Name,
				Email:     member.Email,
			}); err != nil {
				log.Printf("❌ kafka publish error for member %s: %v", member.MemberID, err)
			} else {
				log.Printf("✅ member created and published: %s", member.MemberID)
			}
		}
	} else {
		log.Printf("⚠️ member producer not configured, skipping kafka publish")
	}

	return nil
}

func (s *memberService) ExportPdf() ([]byte, *apierrors.CommonResponse) {
	members, err_1 := s.memberRepo.GetByStatusAndDate("FAILED")
	members_2, err_2 := s.memberRepo.GetByStatusAndDate("PENDING")
	members = append(members, members_2...)

	if err_1 != nil {
		log.Printf("❌ failed to get member: %v", err_1)
		return nil, &apierrors.CommonResponse{StatusCode: http.StatusInternalServerError}
	}

	if err_2 != nil {
		log.Printf("❌ failed to get member: %v", err_2)
		return nil, &apierrors.CommonResponse{StatusCode: http.StatusInternalServerError}
	}
	if len(members) == 0 {
		log.Printf("there are no failed members")
		return nil, &apierrors.CommonResponse{
			StatusCode: http.StatusNotFound,
			Message:    "no active members found",
		}
	}
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AliasNbPages("")

	pdf.AddUTF8FontFromBytes("Sarabun", "", fonts.SarabunRegular)
	pdf.AddUTF8FontFromBytes("Sarabun", "B", fonts.SarabunBold)

	headers := []string{"ชื่อผู้ใช้งาน", "อีเมล", "สถานะ"}
	colWidths := []float64{60.0, 80.0, 50.0}

	pdf.SetHeaderFunc(func() {
		pdf.SetFont("Sarabun", "B", 12)
		pdf.SetFillColor(230, 230, 230)
		for i, h := range headers {
			pdf.CellFormat(colWidths[i], 10, h, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)
		pdf.SetFont("Sarabun", "", 12)
	})

	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Sarabun", "", 8)
		pdf.SetX(-50)
		pdf.CellFormat(40, 10, fmt.Sprintf("Page %d / {nb}", pdf.PageNo()), "", 0, "R", false, 0, "")
	})

	pdf.AddPage() // triggers header automatically

	for _, m := range members {

		pdf.CellFormat(colWidths[0], 10, m.Name, "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[1], 10, m.Email, "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[2], 10, m.Status, "1", 0, "C", false, 0, "")
		pdf.Ln(-1)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		log.Printf("❌ failed to generate pdf: %v", err)
		return nil, &apierrors.CommonResponse{StatusCode: http.StatusInternalServerError}
	}

	return buf.Bytes(), nil
}

func drawTableHeader(pdf *fpdf.Fpdf, headers []string, colWidths []float64) {
	pdf.SetFont("Sarabun", "B", 16)
	pdf.SetFillColor(230, 230, 230)
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 10, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Sarabun", "", 14) // reset to body font for the rows that follow
}

func (s *memberService) GetByStatusAndDate(status string) ([]dtos.MemberDTO, *apierrors.CommonResponse) {
	members, err := s.memberRepo.GetByStatusAndDate(status)
	if err != nil {
		return nil, &apierrors.CommonResponse{StatusCode: http.StatusInternalServerError}
	}

	dtoList := make([]dtos.MemberDTO, 0, len(members))
	for _, m := range members {
		dtoList = append(dtoList, dtos.MemberDTO{
			ID:         m.ID,
			MemberID:   m.MemberID,
			Name:       m.Name,
			CustomerID: m.CustomerID,
			Email:      m.Email,
			CreatedAt:  m.CreatedAt,
			Status:     m.Status,
		})
	}

	return dtoList, nil
}
