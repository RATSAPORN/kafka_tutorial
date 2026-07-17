package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/username/myproject/controllers"
)

func MemberRegisterRoutes(router *gin.RouterGroup, memberController *controllers.MemberController) {
	// Initialize controllers, services, and repositories here
	memberGroup := router.Group("/members")
	memberGroup.Use()
	{
		memberGroup.POST("/", memberController.CreateMember)
		memberGroup.GET("/", memberController.SendMail)
		memberGroup.PUT("/:member_id/status", memberController.UpdateMemberStatus)
		memberGroup.POST("/newmemberbatch", memberController.NewMemberBatch)
		memberGroup.GET("/export-pdf", memberController.ExportPdf)
		memberGroup.GET("/email", memberController.SendMail)
		memberGroup.POST("/login", memberController.Login)
	}

}
