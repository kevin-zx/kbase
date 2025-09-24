package kfeishu

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kevin-zx/kbase/kfeishu/token"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"github.com/pkg/errors"
)

type FeishuDriverClient interface {
	DownloadFile(fileToken string, dstDir, exraInfo string, useFileTokenName bool) (string, error)
	UploadFile(srcPath string, parentType, parentNode string, extraInfo string) (string, error)
}

type feishuDriverImpl struct {
	feishuClient
}

func NewFeishuDriverClient(ts token.TokenService, appID, appSecret string) FeishuDriverClient {
	return &feishuDriverImpl{
		feishuClient: NewFeishuClient(ts, appID, appSecret),
	}
}

// DownloadFile downloads a file from Feishu using the file token.
// fileToken: the token of the file to download
// dstDir: the directory to save the downloaded file
// extrainfo: extra information for the download request, can be empty
// returns: the path of the downloaded file or an error if the download fails
func (f *feishuDriverImpl) DownloadFile(fileToken string, dstDir, extrainfo string, useFileTokenName bool) (string, error) {
	// 构造下载文件的请求
	req := larkdrive.NewDownloadMediaReqBuilder().
		FileToken(fileToken).
		Extra(extrainfo).
		Build()

	// 获取用户访问令牌
	ua, err := f.feishuClient.GetUserAccessToken()
	if err != nil {
		return "", err
	}

	// 发起请求
	resp, err := f.client.Drive.V1.Media.Download(context.Background(), req, larkcore.WithUserAccessToken(ua))
	if err != nil {
		return "", err
	}

	// 检查返回码
	if resp.Code != 0 {
		return "", errors.Wrapf(fmt.Errorf("download file error: %s", resp.Msg), "code: %d", resp.Code)
	}

	// 确保目标目录存在
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return "", errors.Wrap(err, "create directory failed")
	}

	// 拼接完整文件路径
	filePath := filepath.Join(dstDir, resp.FileName)
	if useFileTokenName {
		ext := filepath.Ext(resp.FileName)
		filePath = filepath.Join(dstDir, fileToken+ext)
	}
	// 创建目标文件
	outFile, err := os.Create(filePath)
	if err != nil {
		return "", errors.Wrap(err, "create file failed")
	}
	defer outFile.Close()

	// 将reader内容写入文件
	if _, err := io.Copy(outFile, resp.File); err != nil {
		return "", errors.Wrap(err, "write file failed")
	}

	return filePath, nil
}

// UploadFile uploads a file to Feishu drive
// srcPath: path of the file to upload
// parentType: type of parent node (docx_image/docx_file etc)
// parentNode: token of parent node
// extraInfo: extra information for upload request
// returns: file token or error if upload fails
func (f *feishuDriverImpl) UploadFile(srcPath string, parentType, parentNode string, extraInfo string) (string, error) {
	// Open source file
	file, err := os.Open(srcPath)
	if err != nil {
		return "", errors.Wrap(err, "open source file failed")
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return "", errors.Wrap(err, "get file info failed")
	}

	// Get user access token
	ua, err := f.feishuClient.GetUserAccessToken()
	if err != nil {
		return "", err
	}

	// Build upload request
	req := larkdrive.NewUploadAllMediaReqBuilder().
		Body(larkdrive.NewUploadAllMediaReqBodyBuilder().
			FileName(filepath.Base(srcPath)).
			ParentType(parentType).
			ParentNode(parentNode).
			Size(int(fileInfo.Size())).
			Extra(extraInfo).
			File(file).
			Build()).
		Build()

	// Execute request
	resp, err := f.client.Drive.V1.Media.UploadAll(context.Background(), req, larkcore.WithUserAccessToken(ua))
	if err != nil {
		return "", err
	}

	// Check response code
	if !resp.Success() {
		return "", errors.Wrapf(fmt.Errorf("upload file error: %s", larkcore.Prettify(resp.CodeError)),
			"requestId: %s", resp.RequestId())
	}

	// Return file token
	if resp.Data.FileToken == nil {
		return "", errors.New("empty file token in response")
	}
	return *resp.Data.FileToken, nil
}
