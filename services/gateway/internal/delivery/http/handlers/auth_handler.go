package handlers

import (
	"gateway/internal/clients/grpc"
	"gateway/internal/dto"
	"net/http"
	authv1 "shared/pkg/gen/go/auth/v1"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	client *grpc.AuthGRPCClient
}

func NewAuthHandler(client *grpc.AuthGRPCClient) *AuthHandler {
	return &AuthHandler{client: client}
}

func (h *AuthHandler) RegisterWithNewCompany(c *gin.Context) {
	var req dto.RegisterWithNewCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.RegisterWithNewCompany(c.Request.Context(), &authv1.RegisterWithNewCompanyRequest{
		Login:       req.Login,
		Email:       req.Email,
		Password:    req.Password,
		CompanyName: req.CompanyName,
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

func (h *AuthHandler) RegisterWithExistingCompany(c *gin.Context) {
	var req dto.RegisterWithExistingCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.RegisterWithExistingCompany(c.Request.Context(), &authv1.RegisterWithExistingCompanyRequest{
		Login:             req.Login,
		Email:             req.Email,
		Password:          req.Password,
		CompanyInviteCode: req.InviteCode,
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

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.Login(c.Request.Context(), &authv1.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.AuthResponse{
		UserId:    mustParseUUID(resp.UserId),
		CompanyId: mustParseUUID(resp.CompanyId),
		Email:     resp.Email,
		Role:      resp.Role,
		Token:     resp.Token,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	_, err := h.client.Logout(c.Request.Context(), &authv1.LogoutRequest{
		Token: token,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h *AuthHandler) GetUsersByCompanyId(c *gin.Context) {
	companyId := c.Param("company_id")

	resp, err := h.client.GetUsersByCompanyId(c.Request.Context(), &authv1.GetUsersByCompanyIdRequest{
		CompanyId: companyId,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	users := make([]dto.UserResponse, 0, len(resp.Users))
	for i, u := range resp.Users {
		users[i] = dto.UserResponse{
			UserId:    mustParseUUID(u.UserId),
			CompanyId: mustParseUUID(u.CompanyId),
			Login:     u.Login,
			Email:     u.Email,
			Role:      u.Role,
		}
	}

	c.JSON(http.StatusOK, users)
}

func (h *AuthHandler) GetInviteCode(c *gin.Context) {
	companyId := c.Param("company_id")

	resp, err := h.client.GetInviteCodeByCompanyId(c.Request.Context(), &authv1.GetInviteCodeByCompanyIdRequest{
		CompanyId: companyId,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"invite_code": resp.InviteCode})
}
