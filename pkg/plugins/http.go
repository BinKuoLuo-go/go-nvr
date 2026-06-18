package plugins

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

const (
	SystemErr    = 500
	MySqlErr     = 501
	ValidatorErr = 412
)

type RepError struct {
	code int
	err  error
}

func (re *RepError) Error() string {
	return re.err.Error()
}

func (re *RepError) Code() int {
	return re.code
}

// NewReqError
func NewRepError(code int, err error) *RepError {
	return &RepError{code, err}
}

// NewMySqlError
func NewMySqlError(err error) *RepError {
	return &RepError{MySqlErr, err}
}

// NewValidatorError
func NewValidatorError(err error) *RepError {
	return &RepError{ValidatorErr, err}
}

func ReloadErr(err interface{}) *RepError {
	repErr, ok := err.(*RepError)
	if !ok {
		repError, ok := err.(error)
		if ok {
			return &RepError{
				code: SystemErr,
				//err:  fmt.Errorf("unknow error"),
				err: repError,
			}
		}
		return &RepError{
			code: SystemErr,
			err:  repError,
		}
	}
	return repErr
}

// http 成功
func HttpSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "Success",
		"data": data,
	})
}

// http 失败
func HttpErr(c *gin.Context, err *RepError, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": err.Code(),
		"msg":  err.Error(),
		"data": data,
	})
}

// 返回前端
func HttpResponse(c *gin.Context, httpStatus int, code int, data gin.H, msg string) {
	c.JSON(httpStatus, gin.H{
		"code": code,
		"msg":  msg,
		"data": data,
	})
}

// 返回前端-成功
func Success(c *gin.Context, data gin.H, message string) {
	HttpResponse(c, http.StatusOK, 200, data, message)
}

// 返回前端-失败
func Fail(c *gin.Context, data gin.H, message string) {
	HttpResponse(c, http.StatusBadRequest, 400, data, message)
}
