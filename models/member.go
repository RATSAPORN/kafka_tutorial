package models

import "time"

type Member struct {
	ID         int       `db:"id"`
	Name       string    `db:"username"`
	Email      string    `db:"email"`
	MemberID   string    `db:"member_id"`
	CustomerID string    `db:"customer_id"`
	Role       string    `db:"role"`
	CreatedAt  time.Time `db:"created_at"`
	CreatedBy  string    `db:"created_by"`
	UpdatedAt  time.Time `db:"updated_at"`
	UpdatedBy  int       `db:"updated_by"`
	DeletedAt  time.Time `db:"deleted_at"`
	Deleted    bool      `db:"deleted"`
	DeletedBy  int       `db:"deleted_by"`
	Status     string    `db:"status_acc"`
}
