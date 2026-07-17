package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/username/myproject/controllers"
)

func PermissionRoutes(router *gin.RouterGroup, permissionController *controllers.PermissionController) {
	// Initialize controllers, services, and repositories here
	permissionGroup := router.Group("/permissions")
	permissionGroup.Use()
	{
		permissionGroup.POST("/", permissionController.CheckPermission)

	}

}
