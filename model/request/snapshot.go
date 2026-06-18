/**
@Time : 2026/03/10 14:35
@Author: FangYao( 方少、)
@Description:
@Email: fy20030315@163.com
*/

package request

// SnapshotReq 快照分页查询结构体
type SnapshotReq struct {
	StreamName string `json:"stream_name"`
	Label      string `json:"label"`
	StartMs    int64  `json:"start_ms"` // 查询开始时间
	EndMs      int64  `json:"end_ms"`   // 查询结束时间
	PageNum    int    `json:"pageNum" form:"pageNum"`
	PageSize   int    `json:"pageSize" form:"pageSize"`
}
