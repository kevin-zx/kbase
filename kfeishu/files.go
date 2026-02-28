package kfeishu

import (
	"context"
	"fmt"

	"github.com/kevin-zx/kbase/kfeishu/token"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"github.com/pkg/errors"
)

// FeishuFilesClient 飞书文件/文件夹操作客户端接口
type FeishuFilesClient interface {
	// CreateFolder 创建文件夹
	CreateFolder(name, folderToken string) (*CreateFolderResponse, error)
	// ListFiles 获取文件夹内的文件清单
	ListFiles(folderToken string, pageSize int, pageToken, orderBy, direction, userIdType string) (*ListFilesResponse, error)
	// DeleteFile 删除文件或文件夹
	DeleteFile(fileToken, fileType string) (*DeleteFileResponse, error)
}

// feishuFilesImpl 飞书文件/文件夹操作客户端实现
type feishuFilesImpl struct {
	feishuClient
}

// NewFeishuFilesClient 创建新的飞书文件/文件夹操作客户端
// ts: token服务
// appID: 应用ID
// appSecret: 应用密钥
func NewFeishuFilesClient(ts token.TokenService, appID, appSecret string) FeishuFilesClient {
	return &feishuFilesImpl{
		feishuClient: NewFeishuClient(ts, appID, appSecret),
	}
}

// CreateFolderResponse 创建文件夹响应
type CreateFolderResponse struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

// ListFilesResponse 获取文件清单响应
type ListFilesResponse struct {
	Files   []*FileInfo `json:"files"`
	HasMore bool        `json:"has_more"`
}

// DeleteFileResponse 删除文件响应
type DeleteFileResponse struct {
	TaskID string `json:"task_id"`
}

// FileInfo 文件信息
type FileInfo struct {
	CreatedTime  string `json:"created_time"`
	ModifiedTime string `json:"modified_time"`
	Name         string `json:"name"`
	OwnerID      string `json:"owner_id"`
	ParentToken  string `json:"parent_token"`
	Token        string `json:"token"`
	Type         string `json:"type"`
	URL          string `json:"url"`
}

// CreateFolder 创建文件夹
// name: 文件夹名称
// folderToken: 父文件夹token，如果为空则在根目录创建
// 返回创建文件夹的token和url，或者错误
func (f *feishuFilesImpl) CreateFolder(name, folderToken string) (*CreateFolderResponse, error) {
	// 构建请求体
	reqBodyBuilder := larkdrive.NewCreateFolderFileReqBodyBuilder().
		Name(name)

	// 如果提供了父文件夹token，则设置
	if folderToken != "" {
		reqBodyBuilder = reqBodyBuilder.FolderToken(folderToken)
	}

	// 构建完整请求
	req := larkdrive.NewCreateFolderFileReqBuilder().
		Body(reqBodyBuilder.Build()).
		Build()

	// 获取用户访问令牌
	ua, err := f.feishuClient.GetUserAccessToken()
	if err != nil {
		return nil, errors.Wrap(err, "获取用户访问令牌失败")
	}

	// 发起请求
	resp, err := f.client.Drive.V1.File.CreateFolder(context.Background(), req, larkcore.WithUserAccessToken(ua))
	if err != nil {
		return nil, errors.Wrap(err, "创建文件夹请求失败")
	}

	// 检查响应是否成功
	if !resp.Success() {
		return nil, fmt.Errorf("创建文件夹失败: code=%d, msg=%s, requestId=%s",
			resp.Code, resp.Msg, resp.RequestId())
	}

	// 检查响应数据
	if resp.Data == nil {
		return nil, errors.New("创建文件夹响应数据为空")
	}

	// 构建返回结果
	result := &CreateFolderResponse{}

	if resp.Data.Token != nil {
		result.Token = *resp.Data.Token
	}

	if resp.Data.Url != nil {
		result.URL = *resp.Data.Url
	}

	return result, nil
}

// ListFiles 获取文件夹内的文件清单
// folderToken: 文件夹token，如果为空则获取根目录下的清单
// pageSize: 每页显示的数据项数量，默认100，最大200
// pageToken: 分页标记，第一次请求不填
// orderBy: 排序方式，可选值："EditedTime"、"CreatedTime"，默认"EditedTime"
// direction: 排序规则，可选值："ASC"、"DESC"，默认"DESC"
// userIdType: 用户ID类型，可选值："open_id"、"union_id"、"user_id"，默认"open_id"
// 返回文件清单响应，或者错误
func (f *feishuFilesImpl) ListFiles(folderToken string, pageSize int, pageToken, orderBy, direction, userIdType string) (*ListFilesResponse, error) {
	// 构建请求
	reqBuilder := larkdrive.NewListFileReqBuilder().
		PageSize(pageSize)

	// 设置可选参数
	if folderToken != "" {
		reqBuilder = reqBuilder.FolderToken(folderToken)
	}
	if pageToken != "" {
		reqBuilder = reqBuilder.PageToken(pageToken)
	}
	if orderBy != "" {
		reqBuilder = reqBuilder.OrderBy(orderBy)
	}
	if direction != "" {
		reqBuilder = reqBuilder.Direction(direction)
	}
	if userIdType != "" {
		reqBuilder = reqBuilder.UserIdType(userIdType)
	}

	req := reqBuilder.Build()

	// 获取用户访问令牌
	ua, err := f.feishuClient.GetUserAccessToken()
	if err != nil {
		return nil, errors.Wrap(err, "获取用户访问令牌失败")
	}

	// 发起请求
	resp, err := f.client.Drive.V1.File.List(context.Background(), req, larkcore.WithUserAccessToken(ua))
	if err != nil {
		return nil, errors.Wrap(err, "获取文件清单请求失败")
	}

	// 检查响应是否成功
	if !resp.Success() {
		return nil, fmt.Errorf("获取文件清单失败: code=%d, msg=%s, requestId=%s",
			resp.Code, resp.Msg, resp.RequestId())
	}

	// 检查响应数据
	if resp.Data == nil {
		return nil, errors.New("获取文件清单响应数据为空")
	}

	// 构建返回结果
	result := &ListFilesResponse{
		HasMore: false,
		Files:   []*FileInfo{},
	}

	// 设置是否有更多数据
	if resp.Data.HasMore != nil {
		result.HasMore = *resp.Data.HasMore
	}

	// 转换文件列表
	if resp.Data.Files != nil {
		for _, file := range resp.Data.Files {
			fileInfo := &FileInfo{}

			if file.CreatedTime != nil {
				fileInfo.CreatedTime = *file.CreatedTime
			}
			if file.ModifiedTime != nil {
				fileInfo.ModifiedTime = *file.ModifiedTime
			}
			if file.Name != nil {
				fileInfo.Name = *file.Name
			}
			if file.OwnerId != nil {
				fileInfo.OwnerID = *file.OwnerId
			}
			if file.ParentToken != nil {
				fileInfo.ParentToken = *file.ParentToken
			}
			if file.Token != nil {
				fileInfo.Token = *file.Token
			}
			if file.Type != nil {
				fileInfo.Type = *file.Type
			}
			if file.Url != nil {
				fileInfo.URL = *file.Url
			}

			result.Files = append(result.Files, fileInfo)
		}
	}

	return result, nil
}

// DeleteFile 删除文件或文件夹
// fileToken: 需要删除的文件或文件夹 token
// fileType: 被删除文件的类型，可选值：file, docx, bitable, folder, doc, sheet, mindnote, shortcut, slides
// 返回删除响应，删除文件夹时返回异步任务ID，或者错误
func (f *feishuFilesImpl) DeleteFile(fileToken, fileType string) (*DeleteFileResponse, error) {
	// 构建请求
	req := larkdrive.NewDeleteFileReqBuilder().
		FileToken(fileToken).
		Type(fileType).
		Build()

	// 获取用户访问令牌
	ua, err := f.feishuClient.GetUserAccessToken()
	if err != nil {
		return nil, errors.Wrap(err, "获取用户访问令牌失败")
	}

	// 发起请求
	resp, err := f.client.Drive.V1.File.Delete(context.Background(), req, larkcore.WithUserAccessToken(ua))
	if err != nil {
		return nil, errors.Wrap(err, "删除文件请求失败")
	}

	// 检查响应是否成功
	if !resp.Success() {
		return nil, fmt.Errorf("删除文件失败: code=%d, msg=%s, requestId=%s",
			resp.Code, resp.Msg, resp.RequestId())
	}

	// 构建返回结果
	result := &DeleteFileResponse{}

	if resp.Data != nil && resp.Data.TaskId != nil {
		result.TaskID = *resp.Data.TaskId
	}

	return result, nil
}
