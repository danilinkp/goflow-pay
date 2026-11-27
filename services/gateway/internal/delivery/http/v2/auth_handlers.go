package v2

import (
	"net/http"
	authv1 "shared/pkg/gen/go/auth/v1"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func (s *Server) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	resp, err := s.authClient.Login(c.Request.Context(), &authv1.LoginRequest{
		Email:    string(req.Email),
		Password: req.Password,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	userID, _ := uuid.Parse(resp.UserId)
	compID, _ := uuid.Parse(resp.CompanyId)
	c.JSON(http.StatusOK, AuthTokenResponse{
		Token:     resp.Token,
		TokenType: "Bearer",
		ExpiresIn: 86400,
		UserId:    userID,
		CompanyId: compID,
		Email:     openapi_types.Email(resp.Email),
		Role:      RoleEnum(resp.Role),
	})
}

func (s *Server) Logout(c *gin.Context) {
	token := ""
	bearer := c.GetHeader("Authorization")
	parts := strings.SplitN(bearer, " ", 2)
	if len(parts) == 2 && parts[0] == "Bearer" {
		token = parts[1]
	}

	if token != "" {
		_, _ = s.authClient.Logout(c.Request.Context(), &authv1.LogoutRequest{Token: token})
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) Register(c *gin.Context) {
	var req RegisterUserWithInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	resp, err := s.authClient.RegisterWithExistingCompany(c.Request.Context(), &authv1.RegisterWithExistingCompanyRequest{
		Login:             req.Login,
		Email:             string(req.Email),
		Password:          req.Password,
		CompanyInviteCode: req.InviteCode,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	userID, _ := uuid.Parse(resp.UserId)
	compID, _ := uuid.Parse(resp.CompanyId)
	c.JSON(http.StatusCreated, AuthTokenResponse{
		Token:     resp.Token,
		TokenType: "Bearer",
		ExpiresIn: 86400,
		UserId:    userID,
		CompanyId: compID,
		Email:     openapi_types.Email(resp.Email),
		Role:      RoleEnum(resp.Role),
	})
}

func (s *Server) RegisterWithNewCompany(c *gin.Context) {
	var req RegisterCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	resp, err := s.authClient.RegisterWithNewCompany(c.Request.Context(), &authv1.RegisterWithNewCompanyRequest{
		Login:       req.Login,
		Email:       string(req.Email),
		Password:    req.Password,
		CompanyName: req.CompanyName,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	userID, _ := uuid.Parse(resp.UserId)
	compID, _ := uuid.Parse(resp.CompanyId)
	c.JSON(http.StatusCreated, AuthTokenResponse{
		Token:     resp.Token,
		TokenType: "Bearer",
		ExpiresIn: 86400,
		UserId:    userID,
		CompanyId: compID,
		Email:     openapi_types.Email(resp.Email),
		Role:      RoleEnum(resp.Role),
	})
}

func (s *Server) GetCurrentUserProfile(c *gin.Context) {
	userID, companyID, role := getContext(c)

	usersResp, err := s.authClient.GetUsersByCompanyId(c.Request.Context(), &authv1.GetUsersByCompanyIdRequest{
		CompanyId: companyID.String(),
	})
	if err == nil && usersResp != nil {
		for _, u := range usersResp.Users {
			if u.UserId == userID.String() {
				var createdAt time.Time
				if u.CreatedAt != nil {
					createdAt = u.CreatedAt.AsTime()
				}
				c.JSON(http.StatusOK, UserProfileResponse{
					UserId:    userID,
					CompanyId: companyID,
					Email:     openapi_types.Email(u.Email),
					Login:     u.Login,
					Role:      RoleEnum(u.Role),
					CreatedAt: createdAt,
				})
				return
			}
		}
	}

	// Fallback to context claims if company user list didn't contain or for system admin
	c.JSON(http.StatusOK, UserProfileResponse{
		UserId:    userID,
		CompanyId: companyID,
		Email:     "",
		Login:     "",
		Role:      RoleEnum(role),
		CreatedAt: time.Now(),
	})
}
