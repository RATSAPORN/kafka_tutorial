package schedulers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/username/myproject/services"
)

type ReportScheduler struct {
	memberService services.MemberService
	mailService   services.MailService
	db            *sqlx.DB
	hour          int // hour of day to run, e.g. 8 = 08:00
	minute        int

	stop chan struct{}
}

func NewReportScheduler(memberService services.MemberService, mailService services.MailService, db *sqlx.DB, hour, minute int) *ReportScheduler {
	return &ReportScheduler{
		memberService: memberService,
		mailService:   mailService,
		db:            db,
		hour:          hour,
		minute:        minute,
		stop:          make(chan struct{}),
	}
}

func (s *ReportScheduler) Start() {
	log.Printf("🟡 member scheduler started (daily at %02d:%02d)", s.hour, s.minute)

	for {
		next := s.nextRunTime()
		wait := time.Until(next)
		log.Printf("⏳ next report scheduled at %s (in %s)", next.Format("2006-01-02 15:04:05"), wait)

		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
			s.run()
		case <-s.stop:
			timer.Stop()
			log.Println("🔴 member scheduler stopped")
			return
		}
	}
}

func (s *ReportScheduler) nextRunTime() time.Time {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		log.Printf("warning: could not load Asia/Bangkok timezone, falling back to UTC: %v", err)
		loc = time.UTC
	}

	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), s.hour, s.minute, 0, 0, loc)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
func (s *ReportScheduler) Stop() {
	close(s.stop)
}
func (s *ReportScheduler) run() {
	const reportLockKey = 918273645

	conn, err := s.db.Conn(context.Background())
	if err != nil {
		log.Printf("error getting db conn: %v", err)
		return
	}
	defer conn.Close()

	var acquired bool
	err = conn.QueryRowContext(context.Background(),
		"SELECT pg_try_advisory_lock($1)", reportLockKey).Scan(&acquired)
	if err != nil {
		log.Printf("error acquiring advisory lock: %v", err)
		return
	}

	if !acquired {
		log.Println("⏭️ another instance is already running the report, skipping")
		return
	}
	defer func() {
		if _, err := conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", reportLockKey); err != nil {
			log.Printf("warning: failed to release advisory lock: %v", err)
		}
	}()

	log.Println("🔄 scheduler: checking for failed members...")

	pdf, resp := s.memberService.ExportPdf()
	if resp != nil {
		log.Printf("error exporting pdf: %v", err)
		return
	}

	filename := fmt.Sprintf("%s_members_%s.pdf",
		time.Now().Format("20060102"),
		time.Now().Format("150405"),
	)

	successMembers, apiErr := s.memberService.GetByStatusAndDate("ACTIVE")
	if apiErr != nil {
		log.Printf("error getting active members: %v", apiErr)
		return
	}
	pendingMembers, apiErr := s.memberService.GetByStatusAndDate("PENDING")
	if apiErr != nil {
		log.Printf("error getting pending members: %v", apiErr)
		return
	}
	failedMembers, apiErr := s.memberService.GetByStatusAndDate("FAILED")
	if apiErr != nil {
		log.Printf("error getting failed members: %v", apiErr)
		return
	}

	body := fmt.Sprintf(
		"Daily member registration summary for %s\n\n"+
			"✅ Active:  %d\n"+
			"⏳ Pending: %d\n"+
			"❌ Failed:  %d\n"+
			"-------------------\n"+
			"Total:      %d\n\n"+
			"Full details are attached as PDF.",
		time.Now().Format("2006-01-02"),
		len(successMembers),
		len(pendingMembers),
		len(failedMembers),
		len(successMembers)+len(pendingMembers)+len(failedMembers),
	)

	if err := s.mailService.SendMail(
		"6630290321@student.chula.ac.th",
		fmt.Sprintf("%s summary of member registration", time.Now().Format("20060102")),
		body,
		services.Attachment{Filename: filename, Data: pdf},
	); err != nil {
		log.Printf("error sending mail: %v", err)
		return
	}

	log.Println("✅ scheduler: report sent successfully")
}
