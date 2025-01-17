package controllers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/vfa-nhanbt/todo-api/app/db/repositories"
	"github.com/vfa-nhanbt/todo-api/app/models"

	// "github.com/vfa-nhanbt/todo-api/pkg/constants"
	pkgRepo "github.com/vfa-nhanbt/todo-api/pkg/repositories"

	"github.com/vfa-nhanbt/todo-api/pkg/helpers"
)

type BookController struct {
	Repository *repositories.BookRepository
}

type CreateBookRequest struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description" validate:"required"`
	Price       int    `json:"price"`
}

func (controller *BookController) AddBookHandler(c *fiber.Ctx) error {
	createBookRequest := &CreateBookRequest{}

	/// Validate request body
	err := helpers.ValidateRequestBody(createBookRequest, c)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}

	bookModel := &models.BookModel{
		ID:          uuid.New(),
		Title:       createBookRequest.Title,
		Description: createBookRequest.Description,
		Price:       createBookRequest.Price,
	}

	/// Set author for this book
	userId, err := helpers.GetUserIdFromToken(c)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}
	userUUID, err := uuid.Parse(userId)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}
	bookModel.AuthorID = userUUID
	bookModel.Author = models.AuthorModel{
		UserID: userUUID,
		UserModel: &models.UserModel{
			ID: userUUID,
		},
	}

	/// Get DB context for audit log
	ctx, err := helpers.GetDBContext(c)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}

	/// Insert book to DB
	err = controller.Repository.InsertBook(bookModel, ctx)
	if err != nil {
		res := pkgRepo.BaseResponse{
			Code:      "e-book-001",
			IsSuccess: false,
			Data:      "Cannot insert book record to table with error: " + err.Error(),
		}
		return c.Status(fiber.StatusBadRequest).JSON(res.ToMap())
	}

	helpers.DeleteCachedWithKey("search:books*")

	/// Insert success, return status OK
	res := pkgRepo.BaseResponse{
		Code:      "s-book-001",
		IsSuccess: true,
		Data:      bookModel,
	}
	return c.Status(fiber.StatusOK).JSON(res.ToMap())
}

func (controller *BookController) DeleteBookWithID(c *fiber.Ctx) error {
	bookId := c.Params("id")

	/// Get current user id from token
	userId, err := helpers.GetUserIdFromToken(c)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}

	/// Find book from DB
	bookFromDB, err := controller.Repository.GetBookByID(c.Params("id"))
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}
	/// Check if user is author of this book
	if bookFromDB.AuthorID.String() != userId {
		res := pkgRepo.BaseResponse{
			Code:      "e-book-001",
			IsSuccess: false,
			Data:      "user is not owner of this book",
		}
		return c.Status(fiber.StatusBadRequest).JSON(res.ToMap())
	}

	/// Get DB context for audit log
	ctx, err := helpers.GetDBContext(c)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}

	/// Delete book from DB
	err = controller.Repository.DeleteBookWithID(bookId, ctx)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}

	helpers.DeleteCachedWithKey("search:books*")

	/// Delete success, return status OK
	res := pkgRepo.BaseResponse{
		Code:      "s-book-001",
		IsSuccess: true,
		Data:      "Delete book successfully",
	}
	return c.Status(fiber.StatusOK).JSON(res.ToMap())
}

func (controller *BookController) UpdateBook(c *fiber.Ctx) error {
	// updateBookRequest := &CreateBookRequest{}
	var updateBookRequest struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Price       *int    `json:"price" validate:"updatePrice"`
		helpers.StructHelper
	}
	/// Validate request body
	err := helpers.ValidateRequestBody(&updateBookRequest, c)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}

	/// Get current user id from token
	userId, err := helpers.GetUserIdFromToken(c)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}

	/// Find book from DB
	bookFromDB, err := controller.Repository.GetBookByID(c.Params("id"))
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}
	/// Check if user is author of this book
	if bookFromDB.AuthorID.String() != userId {
		res := pkgRepo.BaseResponse{
			Code:      "e-book-001",
			IsSuccess: false,
			Data:      "user is not owner of this book",
		}
		return c.Status(fiber.StatusBadRequest).JSON(res.ToMap())
	}

	/// Get DB context for audit log
	ctx, err := helpers.GetDBContext(c)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}

	/// Update book
	err = controller.Repository.UpdateBook(bookFromDB, updateBookRequest.StructToUnNilMap(updateBookRequest), ctx)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}

	helpers.DeleteCachedWithKey("search:books*")

	/// Update success, return status OK
	res := pkgRepo.BaseResponse{
		Code:      "s-book-001",
		IsSuccess: true,
		Data:      "update book successfully",
	}
	return c.Status(fiber.StatusOK).JSON(res.ToMap())
}

func (controller *BookController) GetAllBooks(c *fiber.Ctx) error {
	books, err := controller.Repository.GetAllBooks()
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}
	/// Get all books success, return status OK
	res := pkgRepo.BaseResponse{
		Code:      "s-book-001",
		IsSuccess: true,
		Data:      books,
	}
	return c.Status(fiber.StatusOK).JSON(res.ToMap())
}

func (controller *BookController) GetBookByID(c *fiber.Ctx) error {
	bookID := c.Params("id")

	books, err := controller.Repository.GetBookByID(bookID)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}
	/// Get all books success, return status OK
	res := pkgRepo.BaseResponse{
		Code:      "s-book-001",
		IsSuccess: true,
		Data:      books,
	}
	return c.Status(fiber.StatusOK).JSON(res.ToMap())
}

func (controller *BookController) GetBooksByPage(c *fiber.Ctx) error {
	// limit := constants.DefaultPaginationDataLimit
	queryParam := c.Queries()
	pageQuery := queryParam["page"]
	limitQuery := queryParam["limit"]
	page, err := strconv.Atoi(pageQuery)
	limit, err := strconv.Atoi(limitQuery)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}
	books, err := controller.Repository.GetBookByPage(page, limit)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}
	/// Get all books success, return status OK
	res := pkgRepo.BaseResponse{
		Code:      "s-book-001",
		IsSuccess: true,
		Data:      books,
	}
	return c.Status(fiber.StatusOK).JSON(res.ToMap())
}

func (controller *BookController) SearchBooks(c *fiber.Ctx) error {
	queryParam := c.Queries()
	pageQuery := queryParam["page"]
	limitQuery := queryParam["limit"]
	query := queryParam["q"]
	page, err := strconv.Atoi(pageQuery)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}
	limit, err := strconv.Atoi(limitQuery)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}

	var books []*models.BookModel

	/// Check books from redis
	cachedKey := fmt.Sprintf("search:books:%s:page:%s:limit:%s", query, pageQuery, limitQuery)
	cachedResult, err := helpers.GetRedisClient().Get(helpers.GetContextFromFiber(c), cachedKey).Result()
	if err == nil && cachedResult != "" {
		err = json.Unmarshal([]byte(cachedResult), &books)
		if err == nil {
			res := pkgRepo.BaseResponse{
				Code:      "s-book-001",
				IsSuccess: true,
				Data:      books,
			}
			return c.Status(fiber.StatusOK).JSON(res.ToMap())
		}
	}

	books, err = controller.Repository.SearchBooks(page, limit, query)
	if err != nil {
		return pkgRepo.BaseErrorResponse(c, err)
	}

	/// Cache books to redis
	data, _ := json.Marshal(books)
	err = helpers.GetRedisClient().Set(helpers.GetContextFromFiber(c), cachedKey, data, 5*time.Minute).Err()
	if err != nil {
		fmt.Printf("Error caching books to redis: %v", err)
	}

	res := pkgRepo.BaseResponse{
		Code:      "s-book-001",
		IsSuccess: true,
		Data:      books,
	}
	return c.Status(fiber.StatusOK).JSON(res.ToMap())
}
