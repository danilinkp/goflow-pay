package request

type RegisterEmployeeRequest struct {
	Login             string
	Email             string
	Password          string
	CompanyInviteCode string
}
