package v2

import (
	"net/http"
	authv1 "shared/pkg/gen/go/auth/v1"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func (s *Server) ListCompanies(c *gin.Context, params ListCompaniesParams) {
	_, _, role := getContext(c)
	if role != "admin" {
		sendError(c, http.StatusForbidden, "FORBIDDEN", "only admin can list companies")
		return
	}

	limit := 20
	if params.Limit != nil {
		limit = *params.Limit
	}
	offset := 0
	if params.Offset != nil {
		offset = *params.Offset
	}

	resp, err := s.authClient.ListCompanies(c.Request.Context(), &authv1.ListCompaniesRequest{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	items := make([]CompanyResponse, len(resp.Companies))
	for i, comp := range resp.Companies {
		compID, _ := uuid.Parse(comp.CompanyId)
		var createdAt time.Time
		if comp.CreatedAt != nil {
			createdAt = comp.CreatedAt.AsTime()
		}
		items[i] = CompanyResponse{
			CompanyId: compID,
			Name:      comp.Name,
			CreatedAt: createdAt,
		}
	}

	c.JSON(http.StatusOK, CompanyListResponse{
		Items: items,
		Total: int(resp.Total),
	})
}

func (s *Server) GetCompanyById(c *gin.Context, companyId CompanyIdPath) {
	_, userCompanyID, role := getContext(c)
	if role != "admin" && userCompanyID != companyId {
		sendError(c, http.StatusForbidden, "FORBIDDEN", "access denied to this company")
		return
	}

	resp, err := s.authClient.GetCompanyById(c.Request.Context(), &authv1.GetCompanyByIdRequest{
		CompanyId: companyId.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	var createdAt time.Time
	if resp.Company.CreatedAt != nil {
		createdAt = resp.Company.CreatedAt.AsTime()
	}

	c.JSON(http.StatusOK, CompanyResponse{
		CompanyId: companyId,
		Name:      resp.Company.Name,
		CreatedAt: createdAt,
	})
}

func (s *Server) GetCompanyInviteCode(c *gin.Context, companyId CompanyIdPath) {
	_, userCompanyID, role := getContext(c)
	if role != "admin" && (role != "company_admin" || userCompanyID != companyId) {
		sendError(c, http.StatusForbidden, "FORBIDDEN", "access denied to company invite code")
		return
	}

	resp, err := s.authClient.GetInviteCodeByCompanyId(c.Request.Context(), &authv1.GetInviteCodeByCompanyIdRequest{
		CompanyId: companyId.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, InviteCodeResponse{
		CompanyId:  companyId,
		InviteCode: resp.InviteCode,
	})
}

func (s *Server) ListCompanyUsers(c *gin.Context, companyId CompanyIdPath) {
	_, userCompanyID, role := getContext(c)
	if role != "admin" && userCompanyID != companyId {
		sendError(c, http.StatusForbidden, "FORBIDDEN", "access denied to company users")
		return
	}

	resp, err := s.authClient.GetUsersByCompanyId(c.Request.Context(), &authv1.GetUsersByCompanyIdRequest{
		CompanyId: companyId.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	users := make([]UserResponse, len(resp.Users))
	for i, u := range resp.Users {
		uID, _ := uuid.Parse(u.UserId)
		var createdAt time.Time
		if u.CreatedAt != nil {
			createdAt = u.CreatedAt.AsTime()
		}
		users[i] = UserResponse{
			UserId:    uID,
			CompanyId: companyId,
			Email:     openapi_types.Email(u.Email),
			Login:     u.Login,
			Role:      RoleEnum(u.Role),
			CreatedAt: createdAt,
		}
	}

	c.JSON(http.StatusOK, users)
}

func (s *Server) RegisterUserWithInviteCode(c *gin.Context, companyId CompanyIdPath) {
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

func (s *Server) RemoveCompanyUser(c *gin.Context, companyId CompanyIdPath, userId UserIdPath) {
	currentUserID, userCompanyID, role := getContext(c)
	if role != "admin" && (role != "company_admin" || userCompanyID != companyId) {
		sendError(c, http.StatusForbidden, "FORBIDDEN", "access denied to remove user")
		return
	}

	if currentUserID == userId {
		sendError(c, http.StatusBadRequest, "BAD_REQUEST", "cannot delete own account")
		return
	}

	_, err := s.authClient.RemoveCompanyUser(c.Request.Context(), &authv1.RemoveCompanyUserRequest{
		CompanyId: companyId.String(),
		UserId:    userId.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) GetCompanyUser(c *gin.Context, companyId CompanyIdPath, userId UserIdPath) {
	_, userCompanyID, role := getContext(c)
	if role != "admin" && userCompanyID != companyId {
		sendError(c, http.StatusForbidden, "FORBIDDEN", "access denied to company user")
		return
	}

	resp, err := s.authClient.GetUsersByCompanyId(c.Request.Context(), &authv1.GetUsersByCompanyIdRequest{
		CompanyId: companyId.String(),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	for _, u := range resp.Users {
		if u.UserId == userId.String() {
			var createdAt time.Time
			if u.CreatedAt != nil {
				createdAt = u.CreatedAt.AsTime()
			}
			c.JSON(http.StatusOK, UserResponse{
				UserId:    userId,
				CompanyId: companyId,
				Email:     openapi_types.Email(u.Email),
				Login:     u.Login,
				Role:      RoleEnum(u.Role),
				CreatedAt: createdAt,
			})
			return
		}
	}

	sendError(c, http.StatusNotFound, "NOT_FOUND", "user not found in company")
}
