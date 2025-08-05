package request

type CreateUserRequest struct {
	Username string `query:"username" json:"username"`
	Password string `query:"password" json:"password"`
	Email    string `query:"email" json:"email"`
}
