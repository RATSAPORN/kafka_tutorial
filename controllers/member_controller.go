package controllers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/username/myproject/services"
	"github.com/xuri/excelize/v2"

	apierrors "github.com/username/myproject/errors"

	"github.com/gin-gonic/gin"
	"github.com/username/myproject/dtos"
)

type MemberController struct {
	memberService services.MemberService
	mailService   *services.MailService
	authService   services.AuthService
}

func NewMemberController(memberService services.MemberService, mail services.MailService, auth services.AuthService) *MemberController {
	return &MemberController{memberService: memberService, mailService: &mail, authService: auth}
}

// CreateMember godoc
// @Summary      Create a new member
// @Description  Creates a member record from the request body
// @Tags         members
// @Accept       json
// @Produce      json
// @Param        request  body      dtos.CreateMemberRequest  true  "Member to create"
// @Success      200      {object}  dtos.MemberDTO
// @Failure      400      {object}  apierrors.CommonResponse
// @Failure      500      {object}  apierrors.CommonResponse
// @Router       /members [post]
func (mc *MemberController) CreateMember(c *gin.Context) {
	// Implement the logic to create a member
	// You can use mc.memberService to interact with the service layer

	var req dtos.CreateMemberRequest
	log.Printf("member Headers = %v", c.Request.Header)
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := apierrors.ErrorBadRequest
		resp.Message = "invalid request body"
		apierrors.CommonErrorResponse(c, &resp)
		return
	}
	createdMember, resp := mc.memberService.CreateMember(&dtos.MemberDTO{
		Name:  req.Name,
		Email: req.Email,
	})

	if resp != nil {
		apierrors.CommonErrorResponse(c, resp)
		return
	}
	apierrors.CommonSuccessResponse(c, createdMember)
}

// Login godoc
// @Summary      Authenticate member and issue JWT
// @Description  Logs in a member using email and returns an access token
// @Tags         members
// @Accept       json
// @Produce      json
// @Param        request  body      dtos.LoginRequest  true  "Login payload"
// @Success      200      {object}  dtos.LoginResponse
// @Failure      400      {object}  apierrors.CommonResponse
// @Failure      401      {object}  apierrors.CommonResponse
// @Failure      500      {object}  apierrors.CommonResponse
// @Router       /members/login [post]
func (mc *MemberController) Login(c *gin.Context) {
	var req dtos.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := apierrors.ErrorBadRequest
		resp.Message = "invalid request body"
		apierrors.CommonErrorResponse(c, &resp)
		return
	}

	loginResp, resp := mc.memberService.Login(req.Email)
	if resp != nil {
		apierrors.CommonErrorResponse(c, resp)
		return
	}

	apierrors.CommonSuccessResponse(c, loginResp)
}

// @Summary      Update a member's status
// @Description  Updates the status of a member by ID
// @Tags         members
// @Accept       json
// @Produce      json
// @Param        member_id  path      string                     true  "Member ID"
// @Param        request    body      dtos.UpdateStatusRequest   true  "New status"
// @Success      200        {object}  map[string]string
// @Failure      400        {object}  apierrors.CommonResponse
// @Failure      500        {object}  apierrors.CommonResponse
// @Router       /members/{member_id}/status [put]
func (h *MemberController) UpdateMemberStatus(c *gin.Context) {
	memberID := c.Param("member_id")

	var req dtos.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ bind error: %v", err)
		apierrors.CommonErrorResponse(c, &apierrors.ErrorBadRequest)
		return
	}

	err := h.memberService.UpdateMemberStatus(memberID, req.Status)
	if err != nil {
		log.Printf("❌ update status error: %v", err) // ← add this
		apierrors.CommonErrorResponse(c, err)
		return
	}
	apierrors.CommonSuccessResponse(c, "")
}

// NewMemberBatch godoc
// @Summary      Batch upload members
// @Description  Uploads an Excel file containing a list of members and creates them in bulk
// @Tags         members
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "Excel file (.xlsx) with member rows"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /members/newmemberbatch [post]
func (h *MemberController) NewMemberBatch(c *gin.Context) {
	var req dtos.MemberBatchRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to bind request: " + err.Error()})
		return
	}

	src, err := req.File.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file: " + err.Error()})
		return
	}
	defer src.Close()

	excelFile, err := excelize.OpenReader(src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse Excel: " + err.Error()})
		return
	}
	defer excelFile.Close()

	// 1. Read rows from first sheet
	rows, err := excelFile.GetRows(excelFile.GetSheetName(0))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read sheet: " + err.Error()})
		return
	}

	// 2. Parse rows into members
	var members []*dtos.MemberDTO
	for i, row := range rows {
		if i == 0 || len(row) < 2 {
			continue // skip header and incomplete rows
		}
		members = append(members, &dtos.MemberDTO{
			Name:      row[0],
			Email:     row[1],
			CreatedAt: time.Now(),
			CreatedBy: "admin",
		})
	}

	log.Printf("📋 parsed %d members from excel", len(members))

	// 3. Guard empty
	if len(members) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no valid members found in file"})
		return
	}

	// 4. Call service
	if err := h.memberService.CreateMemberBatch(members); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "batch upload successful",
		"count":   len(members),
	})
}

func (h *MemberController) ExportPdf(c *gin.Context) {
	pdfBytes, errResp := h.memberService.ExportPdf()
	if errResp != nil {
		c.JSON(errResp.StatusCode, errResp)
		return
	}

	filename := fmt.Sprintf("%s_members_%s.pdf",
		time.Now().Format("20060102"),
		time.Now().Format("150405"),
	)

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

func (mc *MemberController) SendMail(c *gin.Context) {

	var req dtos.SendMailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	subject := req.Subject
	if subject == "" {
		subject = "Message from Member API"
	}
	if err := mc.mailService.SendMail(req.To, subject, req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send email: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "email sent", "to": req.To})
}
