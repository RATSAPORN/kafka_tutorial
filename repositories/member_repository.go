package repositories

import (
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/username/myproject/models"
)

type MemberRepository interface {
	CreateMember(member *models.Member) (*models.Member, error)
	UpdateMemberStatus(tx *sqlx.Tx, memberID string, status string) error
	GetByStatus(status string) ([]*models.Member, error)
	CreateMemberBatch(tx *sqlx.Tx, members []*models.Member) ([]*models.Member, error) // ← returns created members
	GetAllMember() ([]*models.Member, error)
	GetByStatusAndDate(status string) ([]models.Member, error)
	GetByEmail(email string) (*models.Member, error)
}

type memberRepo struct {
	db *sqlx.DB
}

func NewMemberRepository(db *sqlx.DB) MemberRepository {
	return &memberRepo{db: db}
}

func (r *memberRepo) CreateMember(member *models.Member) (*models.Member, error) {
	query := `
        WITH seq AS (
            INSERT INTO member_id_counters (date, last_seq)
            VALUES (CURRENT_DATE, 1)
            ON CONFLICT (date) DO UPDATE
                SET last_seq = member_id_counters.last_seq + 1
            RETURNING last_seq
        )
        INSERT INTO members (member_id, username, email, created_at, created_by)
        SELECT
            'MBR-' || TO_CHAR(NOW(), 'YYYYMMDD') || '-' || LPAD(seq.last_seq::TEXT, 4, '0'),
            $1, $2, $3, $4
        FROM seq
        RETURNING id, member_id, username, email, created_at, status_acc`

	var newMember models.Member
	err := r.db.QueryRowx(query,
		member.Name,
		member.Email,
		member.CreatedAt,
		member.CreatedBy,
	).StructScan(&newMember)
	if err != nil {
		return nil, err
	}

	return &newMember, nil
}
func (r *memberRepo) UpdateMemberStatus(tx *sqlx.Tx, memberID string, status string) error {
	query := `UPDATE members SET status_acc = $1 WHERE member_id = $2`

	result, err := tx.Exec(query, status, memberID)
	if err != nil {
		log.Printf("❌ repo error: %v", err)
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Printf("❌ rows affected error: %v", err)
		return err
	}

	if rows == 0 {
		log.Printf("❌ no rows updated for member_id: %s", memberID)
		return fmt.Errorf("member not found")
	}

	return nil
}

func (r *memberRepo) GetAllMember() ([]*models.Member, error) {
	query := `
    SELECT id, member_id, username, COALESCE(customer_id, '') AS customer_id, email, created_at, status_acc
    FROM members
    
    ORDER BY created_at ASC
`
	var members []*models.Member
	if err := r.db.Select(&members, query); err != nil {
		log.Printf("❌ failed to get members by status %s: ", err)
		return nil, err
	}

	return members, nil
}

func (r *memberRepo) GetByStatus(status string) ([]*models.Member, error) {
	query := `
		SELECT id, member_id, username,  COALESCE(customer_id, '') AS customer_id,email, created_at, status_acc
		FROM members
		WHERE status_acc = $1
		ORDER BY created_at ASC
	`

	var members []*models.Member
	if err := r.db.Select(&members, query, status); err != nil {
		log.Printf("❌ failed to get members by status %s: %v", status, err)
		return nil, err
	}

	return members, nil
}

// interface

// implementation
func (r *memberRepo) CreateMemberBatch(tx *sqlx.Tx, members []*models.Member) ([]*models.Member, error) {
	query := `
        WITH seq AS (
            INSERT INTO member_id_counters (date, last_seq)
            VALUES (CURRENT_DATE, 1)
            ON CONFLICT (date) DO UPDATE
                SET last_seq = member_id_counters.last_seq + 1
            RETURNING last_seq
        )
        INSERT INTO members (member_id, username, email, created_at, created_by)
        SELECT
            'MBR-' || TO_CHAR(NOW(), 'YYYYMMDD') || '-' || LPAD(seq.last_seq::TEXT, 4, '0'),
            $1, $2, $3, $4
        FROM seq
        RETURNING id, member_id, username, email, created_at, status_acc`

	var createdMembers []*models.Member
	for _, member := range members {
		var newMember models.Member
		err := tx.QueryRowx(query,
			member.Name,
			member.Email,
			member.CreatedAt,
			member.CreatedBy,
		).StructScan(&newMember)
		if err != nil {
			return nil, fmt.Errorf("failed to insert member '%s': %w", member.Email, err)
		}
		createdMembers = append(createdMembers, &newMember)
	}

	return createdMembers, nil
}

func (r *memberRepo) GetByStatusAndDate(status string) ([]models.Member, error) {
	now := time.Now()
	since := now.Add(-24 * time.Hour)

	query := `
		SELECT id, member_id, username,  COALESCE(customer_id, '') AS customer_id, email, created_at, status_acc
		FROM members
		WHERE status_acc = $1 AND created_at >= $2 AND created_at < $3
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, status, since, now)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var members []models.Member
	for rows.Next() {
		var m models.Member
		if err := rows.Scan(
			&m.ID,
			&m.MemberID,
			&m.Name,
			&m.CustomerID,
			&m.Email,
			&m.CreatedAt,
			&m.Status,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		members = append(members, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return members, nil
}
func (r *memberRepo) GetByEmail(email string) (*models.Member, error) {
	query := `
        SELECT id, member_id, username, COALESCE(customer_id, '') AS customer_id, email, created_at, status_acc, role
        FROM members
        WHERE email = $1 AND deleted = FALSE
    `

	var m models.Member
	if err := r.db.Get(&m, query, email); err != nil {
		return nil, err
	}

	return &m, nil
}
