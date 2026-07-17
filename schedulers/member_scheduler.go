package schedulers

import (
	"log"
	"time"

	"github.com/username/myproject/services"
)

type MemberScheduler struct {
	memberService services.MemberService
	interval      time.Duration
	stop          chan struct{}
}

func NewMemberScheduler(memberService services.MemberService, interval time.Duration) *MemberScheduler {
	return &MemberScheduler{
		memberService: memberService,
		interval:      interval,
		stop:          make(chan struct{}),
	}
}

func (s *MemberScheduler) Start() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	log.Printf("🟡 member scheduler started (interval: %s)", s.interval)

	// run once immediately on start
	s.run()

	for {
		select {
		case <-ticker.C:
			s.run()
		case <-s.stop:
			log.Println("🔴 member scheduler stopped")
			return
		}
	}
}

func (s *MemberScheduler) Stop() {
	close(s.stop)
}

func (s *MemberScheduler) run() {
	log.Println("🔄 scheduler: checking for failed members...")
	s.memberService.RetryAllFailedMembers()
}
