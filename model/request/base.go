/**
@Time : 2026/01/16 09:29
@Author: FangYao( 方少、)
@Description:
@Email: fy20030315@163.com
*/

package request

type BaseNullReq struct{}

type BasePageReq struct {
	PageNum  int `json:"pageNum" form:"pageNum"`
	PageSize int `json:"pageSize" form:"pageSize"`
}
