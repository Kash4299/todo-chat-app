package user

import (
	"net/http"
	"todo/common"
	"todo/internal/dtos/request"

	"github.com/gin-gonic/gin"
)

func (h *UserHandler) CreateUser(c *gin.Context) {
	var (
		req = new(request.CreateUserRequest)
		ctx = c.Request.Context()
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		response := common.BadRequestResponse("Invalid request body")
		c.JSON(http.StatusBadRequest, response)
		return
	}

	//TODO : validate request

	err := h.service.CreateUser(ctx, req)
	if err != nil {
		response := common.InternalServerErrorResponse("Failed to create user")
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := common.SuccessResponseWithMessage("User created successfully", "")
	c.JSON(http.StatusCreated, response)
}
