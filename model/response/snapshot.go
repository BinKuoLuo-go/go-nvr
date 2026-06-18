/**
@Time : 2026/03/10 14:46
@Author: FangYao( 方少、)
@Description:
@Email: fy20030315@163.com
*/

package response

import "go-nvr/model"

// SnapshotListRsp 黑名单列表
type SnapshotListRsp struct {
	Total int64            `json:"total"`
	List  []model.Snapshot `json:"list"`
}
