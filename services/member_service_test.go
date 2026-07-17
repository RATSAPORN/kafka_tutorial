package services

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/username/myproject/dtos"
	"github.com/username/myproject/models"
)

// --- fake producer (needs the MemberPublisher interface refactor above) ---

type fakeMemberRepository struct {
	members       []*models.Member
	total         int
	member        *models.Member
	createdMember *models.Member
	updatedMember *models.Member
	err           error
	createArg     *models.Member
}

func (f *fakeMemberRepository) CreateMember(m *models.Member) (*models.Member, error) {
	f.createArg = m
	if f.err != nil {
		return nil, f.err
	}
	return f.createArg, nil
}

func (f *fakeMemberRepository) UpdateMemberStatus(tx *sqlx.Tx, memberID string, status string) error {
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeMemberRepository) GetByStatus(status string) ([]*models.Member, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.members, nil
}

func (f *fakeMemberRepository) GetByEmail(email string) (*models.Member, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, m := range f.members {
		if m.Email == email {
			return m, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (f *fakeMemberRepository) GetAllMember() ([]*models.Member, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.members, nil
}

func (f *fakeMemberRepository) GetByStatusAndDate(status string) ([]models.Member, error) {
	if f.err != nil {
		return nil, f.err
	}

	var result []models.Member
	for _, m := range f.members {
		result = append(result, *m)
	}
	return result, nil
}

func (f *fakeMemberRepository) CreateMemberBatch(tx *sqlx.Tx, members []*models.Member) ([]*models.Member, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.members, nil
}

type fakeMemberProducer struct {
	err       error
	published []*dtos.MemberEvent
}

func (f *fakeMemberProducer) PublishMemberCreated(event *dtos.MemberEvent) error {
	f.published = append(f.published, event)
	return f.err
}

// ============== UpdateMemberStatus ==============

func TestUpdateMemberStatus_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectBegin()
	mock.ExpectCommit()

	repo := &fakeMemberRepository{}
	service := NewMemberService(repo, nil, sqlxDB)

	resp := service.UpdateMemberStatus("user1244", "INACTIVE")
	if resp != nil {
		t.Fatalf("expected nil response, got: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestUpdateMemberStatus_RepoError_RollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectBegin()
	mock.ExpectRollback()

	repo := &fakeMemberRepository{err: errors.New("update failed")}
	service := NewMemberService(repo, nil, sqlxDB)

	resp := service.UpdateMemberStatus("user1244", "INACTIVE")
	if resp == nil {
		t.Fatalf("expected error response, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestUpdateMemberStatus_BeginError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectBegin().WillReturnError(errors.New("connection refused"))

	repo := &fakeMemberRepository{}
	service := NewMemberService(repo, nil, sqlxDB)

	resp := service.UpdateMemberStatus("user1244", "INACTIVE")
	if resp == nil {
		t.Fatalf("expected error response, got nil")
	}
}

func TestUpdateMemberStatus_CommitError_RollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
	mock.ExpectRollback()

	repo := &fakeMemberRepository{}
	service := NewMemberService(repo, nil, sqlxDB)

	resp := service.UpdateMemberStatus("user1244", "INACTIVE")
	if resp == nil {
		t.Fatalf("expected error response, got nil")
	}
}

// ============== CreateMemberBatch ==============
// Assumes the fixed version above (real Commit/Rollback, nil-guard on producer)

func TestCreateMemberBatch_Success_PublishesEachMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectBegin()
	mock.ExpectCommit()

	repo := &fakeMemberRepository{
		members: []*models.Member{
			{MemberID: "user1", Name: "Alice", Email: "alice@example.com"},
			{MemberID: "user2", Name: "Bob", Email: "bob@example.com"},
		},
	}
	producer := &fakeMemberProducer{}
	service := NewMemberService(repo, producer, sqlxDB)

	err = service.CreateMemberBatch([]*dtos.MemberDTO{
		{Name: "Alice", Email: "alice@example.com"},
		{Name: "Bob", Email: "bob@example.com"},
	})
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(producer.published) != 2 {
		t.Fatalf("expected 2 published events, got %d", len(producer.published))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestCreateMemberBatch_RepoError_RollsBackAndSkipsPublish(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectBegin()
	mock.ExpectRollback()

	repo := &fakeMemberRepository{err: errors.New("batch insert failed")}
	producer := &fakeMemberProducer{}
	service := NewMemberService(repo, producer, sqlxDB)

	err = service.CreateMemberBatch([]*dtos.MemberDTO{
		{Name: "Alice", Email: "alice@example.com"},
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if len(producer.published) != 0 {
		t.Fatalf("expected no publishes on repo error, got %d", len(producer.published))
	}
}

func TestCreateMemberBatch_NilProducer_DoesNotPanic(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	mock.ExpectBegin()
	mock.ExpectCommit()

	repo := &fakeMemberRepository{
		members: []*models.Member{
			{MemberID: "user1", Name: "Alice", Email: "alice@example.com"},
		},
	}
	service := NewMemberService(repo, nil, sqlxDB)

	err = service.CreateMemberBatch([]*dtos.MemberDTO{
		{Name: "Alice", Email: "alice@example.com"},
	})
	if err != nil {
		t.Fatalf("expected nil error even with nil producer, got: %v", err)
	}
}

func TestCreateMember(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	repo := &fakeMemberRepository{
		members: []*models.Member{
			{
				ID: 1, Name: "mm", Email: "ff", MemberID: "user1244", CustomerID: "ffdd12", CreatedAt: time.Now(), Status: "ACTIVE",
			},
		},
	}
	producer := &fakeMemberProducer{}
	service := NewMemberService(repo, producer, sqlxDB)

	// Duplicate email — should fail before ever reaching the producer
	newMember, resp := service.CreateMember(&dtos.MemberDTO{Name: "hh", Email: "ff"})
	if newMember != nil {
		t.Fatalf("expected already have this email got:%v", newMember)
	}
	if resp == nil {
		t.Fatalf("expect duplicate error got: %v", resp)
	}
	if len(producer.published) != 0 {
		t.Fatalf("expected no publish on duplicate email, got %d", len(producer.published))
	}

	// New email — should succeed and publish exactly once
	newMember2, resp2 := service.CreateMember(&dtos.MemberDTO{Name: "www", Email: "jj"})
	if newMember2 == nil {
		t.Fatalf("expected user created got: %v", newMember2)
	}
	if resp2 != nil {
		t.Fatalf("expected resp to be nil got: %v", resp2)
	}
	if len(producer.published) != 1 {
		t.Fatalf("expected exactly 1 publish, got %d", len(producer.published))
	}
	event := producer.published[0]
	if event.Email != "jj" || event.EventType != "member.created" {
		t.Errorf("unexpected published event: %+v", event)
	}
}

func TestCreateMember_PublishError_DoesNotFailRequest(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	repo := &fakeMemberRepository{} // no existing members, so no duplicate match
	producer := &fakeMemberProducer{err: errors.New("kafka broker unreachable")}
	service := NewMemberService(repo, producer, sqlxDB)

	newMember, resp := service.CreateMember(&dtos.MemberDTO{Name: "Alice", Email: "alice@example.com"})

	// Kafka publish failures are logged, not propagated — request should still succeed
	if newMember == nil {
		t.Fatalf("expected member to be created despite publish error, got nil")
	}
	if resp != nil {
		t.Fatalf("expected nil resp despite publish error, got: %v", resp)
	}
	if len(producer.published) != 1 {
		t.Fatalf("expected publish to be attempted once, got %d", len(producer.published))
	}
}

func TestCreateMember_NilProducer_SkipsPublishWithoutPanic(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	repo := &fakeMemberRepository{}
	service := NewMemberService(repo, nil, sqlxDB) // nil producer — was the original panic case

	newMember, resp := service.CreateMember(&dtos.MemberDTO{Name: "Alice", Email: "alice@example.com"})

	if newMember == nil {
		t.Fatalf("expected member to be created even with nil producer, got nil")
	}
	if resp != nil {
		t.Fatalf("expected nil resp with nil producer, got: %v", resp)
	}
}
