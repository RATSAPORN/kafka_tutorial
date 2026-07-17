package repositories

import (
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"

	"github.com/username/myproject/models"
)

func newRepoWithMock(t *testing.T) (MemberRepository, sqlmock.Sqlmock, *sqlx.DB) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := NewMemberRepository(sqlxDB)
	return repo, mock, sqlxDB
}

// ============== CreateMember ==============

func TestMemberRepo_CreateMember_Success(t *testing.T) {
	repo, mock, sqlxDB := newRepoWithMock(t)
	defer sqlxDB.Close()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "member_id", "username", "email", "created_at", "status_acc"}).
		AddRow(1, "MBR-20260625-0001", "Alice", "alice@example.com", now, "ACTIVE")

	mock.ExpectQuery("INSERT INTO members").
		WithArgs("Alice", "alice@example.com", now, "admin").
		WillReturnRows(rows)

	member := &models.Member{
		Name:      "Alice",
		Email:     "alice@example.com",
		CreatedAt: now,
		CreatedBy: "admin",
	}

	result, err := repo.CreateMember(member)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatalf("expected non-nil member")
	}
	if result.MemberID != "MBR-20260625-0001" {
		t.Errorf("expected MemberID 'MBR-20260625-0001', got: %s", result.MemberID)
	}
	if result.Email != "alice@example.com" {
		t.Errorf("expected Email 'alice@example.com', got: %s", result.Email)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestMemberRepo_CreateMember_QueryError(t *testing.T) {
	repo, mock, sqlxDB := newRepoWithMock(t)
	defer sqlxDB.Close()

	now := time.Now()
	mock.ExpectQuery("INSERT INTO members").
		WithArgs("Alice", "alice@example.com", now, "admin").
		WillReturnError(fmtErr("unique constraint violation"))

	member := &models.Member{
		Name:      "Alice",
		Email:     "alice@example.com",
		CreatedAt: now,
		CreatedBy: "admin",
	}

	result, err := repo.CreateMember(member)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if result != nil {
		t.Fatalf("expected nil member on error, got: %+v", result)
	}
}

// ============== UpdateMemberStatus ==============

func TestMemberRepo_UpdateMemberStatus_Success(t *testing.T) {
	repo, mock, sqlxDB := newRepoWithMock(t)
	defer sqlxDB.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE members SET status_acc").
		WithArgs("INACTIVE", "MBR-20260625-0001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback() // just to close the tx cleanly in this test

	tx, err := sqlxDB.Beginx()
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	err = repo.UpdateMemberStatus(tx, "MBR-20260625-0001", "INACTIVE")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	tx.Rollback() // satisfies the ExpectRollback() above; not asserting commit behavior here

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestMemberRepo_UpdateMemberStatus_NoRowsAffected(t *testing.T) {
	repo, mock, sqlxDB := newRepoWithMock(t)
	defer sqlxDB.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE members SET status_acc").
		WithArgs("INACTIVE", "MBR-NONEXISTENT").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	tx, err := sqlxDB.Beginx()
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	err = repo.UpdateMemberStatus(tx, "MBR-NONEXISTENT", "INACTIVE")
	if err == nil {
		t.Fatalf("expected 'member not found' error, got nil")
	}

	tx.Rollback()
}

func TestMemberRepo_UpdateMemberStatus_ExecError(t *testing.T) {
	repo, mock, sqlxDB := newRepoWithMock(t)
	defer sqlxDB.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE members SET status_acc").
		WithArgs("INACTIVE", "MBR-20260625-0001").
		WillReturnError(fmtErr("connection lost"))
	mock.ExpectRollback()

	tx, err := sqlxDB.Beginx()
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	err = repo.UpdateMemberStatus(tx, "MBR-20260625-0001", "INACTIVE")
	if err == nil {
		t.Fatalf("expected exec error, got nil")
	}

	tx.Rollback()
}

// ============== GetByStatus ==============

func TestMemberRepo_GetByStatus_Success(t *testing.T) {
	repo, mock, sqlxDB := newRepoWithMock(t)
	defer sqlxDB.Close()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "member_id", "username", "email", "created_at", "status_acc"}).
		AddRow(1, "MBR-20260625-0001", "Alice", "alice@example.com", now, "ACTIVE").
		AddRow(2, "MBR-20260625-0002", "Bob", "bob@example.com", now, "ACTIVE")

	mock.ExpectQuery("SELECT (.+) FROM members").
		WithArgs("ACTIVE").
		WillReturnRows(rows)

	members, err := repo.GetByStatus("ACTIVE")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if members[0].MemberID != "MBR-20260625-0001" {
		t.Errorf("unexpected first member: %+v", members[0])
	}
}

func TestMemberRepo_GetByStatus_QueryError(t *testing.T) {
	repo, mock, sqlxDB := newRepoWithMock(t)
	defer sqlxDB.Close()

	mock.ExpectQuery("SELECT (.+) FROM members").
		WithArgs("ACTIVE").
		WillReturnError(fmtErr("query timeout"))

	members, err := repo.GetByStatus("ACTIVE")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if members != nil {
		t.Fatalf("expected nil members on error, got: %+v", members)
	}
}

// ============== CreateMemberBatch ==============

func TestMemberRepo_CreateMemberBatch_Success(t *testing.T) {
	repo, mock, sqlxDB := newRepoWithMock(t)
	defer sqlxDB.Close()

	now := time.Now()

	mock.ExpectBegin()

	row1 := sqlmock.NewRows([]string{"id", "member_id", "username", "email", "created_at", "status_acc"}).
		AddRow(1, "MBR-20260625-0001", "Alice", "alice@example.com", now, "ACTIVE")
	mock.ExpectQuery("INSERT INTO members").
		WithArgs("Alice", "alice@example.com", now, "admin").
		WillReturnRows(row1)

	row2 := sqlmock.NewRows([]string{"id", "member_id", "username", "email", "created_at", "status_acc"}).
		AddRow(2, "MBR-20260625-0002", "Bob", "bob@example.com", now, "ACTIVE")
	mock.ExpectQuery("INSERT INTO members").
		WithArgs("Bob", "bob@example.com", now, "admin").
		WillReturnRows(row2)

	mock.ExpectRollback() // closing tx in this test; not asserting commit here

	tx, err := sqlxDB.Beginx()
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	members := []*models.Member{
		{Name: "Alice", Email: "alice@example.com", CreatedAt: now, CreatedBy: "admin"},
		{Name: "Bob", Email: "bob@example.com", CreatedAt: now, CreatedBy: "admin"},
	}

	created, err := repo.CreateMemberBatch(tx, members)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("expected 2 created members, got %d", len(created))
	}

	tx.Rollback()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestMemberRepo_CreateMemberBatch_PartialFailure(t *testing.T) {
	repo, mock, sqlxDB := newRepoWithMock(t)
	defer sqlxDB.Close()

	now := time.Now()

	mock.ExpectBegin()

	row1 := sqlmock.NewRows([]string{"id", "member_id", "username", "email", "created_at", "status_acc"}).
		AddRow(1, "MBR-20260625-0001", "Alice", "alice@example.com", now, "ACTIVE")
	mock.ExpectQuery("INSERT INTO members").
		WithArgs("Alice", "alice@example.com", now, "admin").
		WillReturnRows(row1)

	// Second insert fails (e.g. duplicate email)
	mock.ExpectQuery("INSERT INTO members").
		WithArgs("Bob", "bob@example.com", now, "admin").
		WillReturnError(fmtErr("duplicate key value violates unique constraint"))

	mock.ExpectRollback()

	tx, err := sqlxDB.Beginx()
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	members := []*models.Member{
		{Name: "Alice", Email: "alice@example.com", CreatedAt: now, CreatedBy: "admin"},
		{Name: "Bob", Email: "bob@example.com", CreatedAt: now, CreatedBy: "admin"},
	}

	created, err := repo.CreateMemberBatch(tx, members)
	if err == nil {
		t.Fatalf("expected error on second insert failure, got nil")
	}
	if created != nil {
		t.Fatalf("expected nil created members on failure, got: %+v", created)
	}

	tx.Rollback()
}

// small helper since errors.New collides with "errors" import naming in some setups
func fmtErr(msg string) error {
	return &simpleError{msg}
}

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }
