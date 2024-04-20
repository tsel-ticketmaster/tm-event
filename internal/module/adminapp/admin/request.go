package admin

type SignInRequest struct {
	Email    string `json:"email" validate:"email"`
	Password string `json:"password" validate:"required"`
}

type CreateRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"email"`
}
