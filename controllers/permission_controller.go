package controllers

import (
	"log"
	"net/http"
	"strings"

	"github.com/username/myproject/dtos"
	"github.com/username/myproject/services"

	"github.com/gin-gonic/gin"
)

type PermissionController struct {
	authService services.AuthService       // validates tokens, returns user identity/claims
	permService services.PermissionService // your original service from earlier
}

func NewPermissionController(
	authService services.AuthService,
	permService services.PermissionService,
) *PermissionController {
	return &PermissionController{
		authService: authService,
		permService: permService,
	}
}

// CheckPermission godoc
// @Summary      Check user permission
// @Description  Validates the bearer token and checks if the user has permission to access the given method/path
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Param        request  body      dtos.PermissionRequest  true  "Token, method, and path to check"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      403      {object}  map[string]interface{}
// @Failure      500      {object}  map[string]string
// @Router       /permissions [post]
func (pc *PermissionController) CheckPermission(c *gin.Context) {
	var req dtos.PermissionRequest
	log.Printf("Permission Headers = %v", c.Request.Header)
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid request body",
		})
		return
	}
	log.Printf("Permission Request = method=%s path=%s", req.Method, req.Path)
	if req.Authorization == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "missing authorization",
		})
		return
	}
	log.Printf("Permission Request = method=%s path=%s", req.Method, req.Path)
	token := strings.TrimPrefix(req.Authorization, "Bearer ")

	log.Printf("TOKEN = %s", token)

	claims, err := pc.authService.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": err.Error(),
		})
		return
	}

	allowed, err := pc.permService.HasPermission(
		c.Request.Context(),
		claims.UserID,
		req.Method,
		req.Path,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "permission check failed",
		})
		return
	}

	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{
			"allowed": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"allowed": true,
		"user_id": claims.UserID,
		"role":    claims.Role,
	})
}
