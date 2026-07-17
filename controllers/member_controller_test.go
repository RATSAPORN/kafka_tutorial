package controllers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

	"github.com/username/myproject/dtos"
	apierrors "github.com/username/myproject/errors"
	"github.com/username/myproject/services"
)

// ---- fake service implementing services.MemberService ----

type fakeMemberService struct {
	createMemberResp     *dtos.MemberDTO
	createMemberErr      *apierrors.CommonResponse
	updateStatusErr      *apierrors.CommonResponse
	createBatchErr       error
	loginResp            *dtos.LoginResponse
	loginErr             *apierrors.CommonResponse
	receivedBatchMembers []*dtos.MemberDTO
}

func (f *fakeMemberService) CreateMember(m *dtos.MemberDTO) (*dtos.MemberDTO, *apierrors.CommonResponse) {
	return f.createMemberResp, f.createMemberErr
}

func (f *fakeMemberService) UpdateMemberStatus(memberID string, status string) *apierrors.CommonResponse {
	return f.updateStatusErr
}

func (f *fakeMemberService) CreateMemberBatch(members []*dtos.MemberDTO) error {
	f.receivedBatchMembers = members
	return f.createBatchErr
}

func (f *fakeMemberService) Login(email string) (*dtos.LoginResponse, *apierrors.CommonResponse) {
	return f.loginResp, f.loginErr
}

func (f *fakeMemberService) GetByStatusAndDate(status string) ([]dtos.MemberDTO, *apierrors.CommonResponse) {
	return nil, nil
}

func (f *fakeMemberService) RetryAllFailedMembers() {}

func (f *fakeMemberService) ExportPdf() ([]byte, *apierrors.CommonResponse) {
	return nil, nil
}

func setupRouter(svc *fakeMemberService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := NewMemberController(svc, services.MailService{}, nil)
	r.POST("/members", ctrl.CreateMember)
	r.PUT("/members/:member_id/status", ctrl.UpdateMemberStatus)
	r.POST("/members/newmemberbatch", ctrl.NewMemberBatch)
	r.POST("/members/login", ctrl.Login)
	return r
}

// ============== CreateMember ==============

func TestMemberController_CreateMember_Success(t *testing.T) {
	svc := &fakeMemberService{
		createMemberResp: &dtos.MemberDTO{Name: "Alice", Email: "alice@example.com"},
		createMemberErr:  nil,
	}
	r := setupRouter(svc)

	body := `{"name":"Alice","email":"alice@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "alice@example.com") {
		t.Errorf("expected response to contain created member email, got: %s", w.Body.String())
	}
}

func TestMemberController_CreateMember_InvalidBody(t *testing.T) {
	svc := &fakeMemberService{}
	r := setupRouter(svc)

	body := `{"name":"Alice"` // missing closing brace and required email
	req := httptest.NewRequest(http.MethodPost, "/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestMemberController_CreateMember_MissingRequiredField(t *testing.T) {
	svc := &fakeMemberService{}
	r := setupRouter(svc)

	// valid JSON, but missing required "email" field (binding:"required")
	body := `{"name":"Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing required field, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestMemberController_CreateMember_ServiceError(t *testing.T) {
	svc := &fakeMemberService{
		createMemberResp: nil,
		createMemberErr:  &apierrors.ErrorInternal,
	}
	r := setupRouter(svc)

	body := `{"name":"Alice","email":"alice@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("expected non-200 on service error, got 200, body: %s", w.Body.String())
	}
}

// ============== UpdateMemberStatus ==============

func TestMemberController_UpdateMemberStatus_Success(t *testing.T) {
	svc := &fakeMemberService{updateStatusErr: nil}
	r := setupRouter(svc)

	body := `{"status":"INACTIVE"}`
	req := httptest.NewRequest(http.MethodPut, "/members/user1244/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestMemberController_UpdateMemberStatus_InvalidBody(t *testing.T) {
	svc := &fakeMemberService{}
	r := setupRouter(svc)

	body := `{"status":` // malformed JSON
	req := httptest.NewRequest(http.MethodPut, "/members/user1244/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestMemberController_UpdateMemberStatus_MissingRequiredField(t *testing.T) {
	svc := &fakeMemberService{}
	r := setupRouter(svc)

	// valid JSON, but "status" is required and missing
	body := `{"customer_id":"cust1"}`
	req := httptest.NewRequest(http.MethodPut, "/members/user1244/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing required status, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestMemberController_UpdateMemberStatus_ServiceError(t *testing.T) {
	svc := &fakeMemberService{updateStatusErr: &apierrors.ErrorInternal}
	r := setupRouter(svc)

	body := `{"status":"INACTIVE"}`
	req := httptest.NewRequest(http.MethodPut, "/members/user1244/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("expected non-200 on service error, got 200, body: %s", w.Body.String())
	}
}

// ============== NewMemberBatch ==============

func buildExcelMultipartRequest(t *testing.T, rows [][]string, fieldName string) *http.Request {
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	for i, row := range rows {
		for j, val := range row {
			cell, err := excelize.CoordinatesToCellName(j+1, i+1)
			if err != nil {
				t.Fatalf("failed to compute cell coords: %v", err)
			}
			if err := f.SetCellValue(sheet, cell, val); err != nil {
				t.Fatalf("failed to set cell value: %v", err)
			}
		}
	}

	var fileBuf bytes.Buffer
	if err := f.Write(&fileBuf); err != nil {
		t.Fatalf("failed to write excel to buffer: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fieldName, "members.xlsx")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write(fileBuf.Bytes()); err != nil {
		t.Fatalf("failed to write file part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/members/newmemberbatch", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestMemberController_NewMemberBatch_Success(t *testing.T) {
	svc := &fakeMemberService{createBatchErr: nil}
	r := setupRouter(svc)

	rows := [][]string{
		{"Name", "Email"}, // header row, skipped by controller logic
		{"Alice", "alice@example.com"},
		{"Bob", "bob@example.com"},
	}
	req := buildExcelMultipartRequest(t, rows, "file")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	if len(svc.receivedBatchMembers) != 2 {
		t.Fatalf("expected service to receive 2 members, got %d", len(svc.receivedBatchMembers))
	}
	if svc.receivedBatchMembers[0].Email != "alice@example.com" {
		t.Errorf("expected first member email alice@example.com, got: %s", svc.receivedBatchMembers[0].Email)
	}

	var respBody map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if count, ok := respBody["count"].(float64); !ok || int(count) != 2 {
		t.Errorf("expected count=2 in response, got: %v", respBody["count"])
	}
}

func TestMemberController_NewMemberBatch_NoFile(t *testing.T) {
	svc := &fakeMemberService{}
	r := setupRouter(svc)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writer.Close() // no file part added — req.File will fail to bind

	req := httptest.NewRequest(http.MethodPost, "/members/newmemberbatch", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing file, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestMemberController_NewMemberBatch_InvalidExcel(t *testing.T) {
	svc := &fakeMemberService{}
	r := setupRouter(svc)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "not_an_excel.xlsx")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte("this is not a valid xlsx file")); err != nil {
		t.Fatalf("failed to write fake file content: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/members/newmemberbatch", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unparseable excel, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestMemberController_NewMemberBatch_EmptyAfterHeader(t *testing.T) {
	svc := &fakeMemberService{}
	r := setupRouter(svc)

	// only the header row, no data rows — should trigger the "no valid members found" guard
	rows := [][]string{
		{"Name", "Email"},
	}
	req := buildExcelMultipartRequest(t, rows, "file")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty member list, got %d, body: %s", w.Code, w.Body.String())
	}
	if len(svc.receivedBatchMembers) != 0 {
		t.Fatalf("expected service not to be called, but it received %d members", len(svc.receivedBatchMembers))
	}
}

func TestMemberController_NewMemberBatch_ServiceError(t *testing.T) {
	svc := &fakeMemberService{createBatchErr: assertionError("db write failed")}
	r := setupRouter(svc)

	rows := [][]string{
		{"Name", "Email"},
		{"Alice", "alice@example.com"},
	}
	req := buildExcelMultipartRequest(t, rows, "file")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on service error, got %d, body: %s", w.Code, w.Body.String())
	}
}

type assertionError string

func (e assertionError) Error() string { return string(e) }
