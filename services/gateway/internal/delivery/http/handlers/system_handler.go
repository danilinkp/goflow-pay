package handlers

import (
	"gateway/internal/clients/grpc"
	"gateway/internal/delivery/http/middleware"
	"gateway/internal/dto"
	"net/http"
	authv1 "shared/pkg/gen/go/auth/v1"

	"github.com/gin-gonic/gin"
)

type SystemHandler struct {
	authClient     *grpc.AuthGRPCClient
	bootstrapToken string
}

func NewSystemHandler(authClient *grpc.AuthGRPCClient, bootstrapToken string) *SystemHandler {
	return &SystemHandler{
		authClient:     authClient,
		bootstrapToken: bootstrapToken,
	}
}

func (h *SystemHandler) Init(c *gin.Context) {
	var req struct {
		BootstrapToken string `json:"bootstrap_token"`
		Login          string `json:"login"`
		Email          string `json:"email"`
		Password       string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.BootstrapToken != h.bootstrapToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid bootstrap token"})
		return
	}

	resp, err := h.authClient.InitSystem(c.Request.Context(), &authv1.InitSystemRequest{
		Login:    req.Login,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.AuthResponse{
		UserId:    mustParseUUID(resp.UserId),
		CompanyId: mustParseUUID(resp.CompanyId),
		Email:     resp.Email,
		Role:      resp.Role,
		Token:     resp.Token,
	})
}

func (h *SystemHandler) AddAdmin(c *gin.Context) {
	var req struct {
		Login    string `json:"login"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authClient.AddAdmin(middleware.GRPCContext(c), &authv1.AddAdminRequest{
		Login:    req.Login,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.AuthResponse{
		UserId:    mustParseUUID(resp.UserId),
		CompanyId: mustParseUUID(resp.CompanyId),
		Email:     resp.Email,
		Role:      resp.Role,
		Token:     resp.Token,
	})
}
