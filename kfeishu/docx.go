package kfeishu

import (
	"context"
	"fmt"
	"log"

	"github.com/kevin-zx/kbase/kfeishu/token"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdocx "github.com/larksuite/oapi-sdk-go/v3/service/docx/v1"
)

type FeishuDocxClient struct {
	feishuClient
}

// NewFeishuDocxClient creates a new Feishu document client
// ts: token service
// appId: app id
// appSecret: app secret
func NewFeishuDocxClient(ts token.TokenService, appId, appSecret string) *FeishuDocxClient {
	return &FeishuDocxClient{
		feishuClient: NewFeishuClient(ts, appId, appSecret),
	}
}

func (f *FeishuDocxClient) Close() {
	f.ts.Close()
}

type CreateDocxResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// CreateDocx creates a new docx file in the Feishu app
func (f *FeishuDocxClient) CreateDocx(title string, folderToken string) (*larkdocx.Document, error) {
	req := larkdocx.NewCreateDocumentReqBuilder().
		Body(larkdocx.NewCreateDocumentReqBodyBuilder().
			FolderToken(folderToken).
			Title(title).
			Build()).
		Build()

	uat, err := f.GetUserAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get user access token failed: %w", err)
	}
	resp, err := f.client.Docx.V1.Document.Create(context.Background(), req, larkcore.WithUserAccessToken(uat))
	if err != nil {
		return nil, fmt.Errorf("create docx failed: %w", err)
	}
	if resp != nil {
		if resp.Code != 0 {
			log.Printf("raw body : %s", string(resp.RawBody))
			return nil, fmt.Errorf("create docx failed: %s", resp.Msg)
		}
	}
	return resp.Data.Document, nil
}

// ConvertMarkdownToDocx converts markdown content to a docx file
func (f *FeishuDocxClient) ConvertMarkdownToDocx(markdownContent string) (*larkdocx.ConvertDocumentRespData, error) {
	req := larkdocx.NewConvertDocumentReqBuilder().
		Body(larkdocx.NewConvertDocumentReqBodyBuilder().
			ContentType(`markdown`).
			Content(markdownContent).
			Build()).
		Build()
	uat, err := f.GetUserAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get user access token failed: %w", err)
	}
	resp, err := f.client.Docx.V1.Document.Convert(context.Background(), req, larkcore.WithUserAccessToken(uat))
	if err != nil {
		return nil, fmt.Errorf("convert markdown to docx failed: %w", err)
	}
	if resp != nil {
		if resp.Code != 0 {
			log.Printf("raw body : %s", string(resp.RawBody))
			return nil, fmt.Errorf("convert markdown to docx failed: %s", resp.Msg)
		}

	}
	return resp.Data, nil
}

// CreateDocumentBlockDescendantReq creates a new block in the document
type CreateDocumentBlockDescendantReq struct {
	DocumentID  string            `json:"document_id"`
	BlockID     string            `json:"block_id"`
	ChildrenID  []string          `json:"children_id"`
	Index       *int              `json:"index"`
	Descendants []*larkdocx.Block `json:"descendants"`

	// DocumentRevisionID string `json:"document_revision_id"`
}

// CreateDocumentBlockDescendant creates a new block in the document
func (f *FeishuDocxClient) CreateDocumentBlockDescendant(r *CreateDocumentBlockDescendantReq) (*larkdocx.CreateDocumentBlockDescendantRespData, error) {
	idx := -1
	if r.Index != nil {
		idx = *r.Index
	}
	for i, b := range r.Descendants {
		if b.Table != nil && b.Table.Property != nil && b.Table.Property.MergeInfo != nil {
			// Clear merge info to avoid errors
			r.Descendants[i].Table.Property.MergeInfo = nil
		}
	}
	req := larkdocx.NewCreateDocumentBlockDescendantReqBuilder().
		DocumentId(r.DocumentID).
		BlockId(r.BlockID).
		DocumentRevisionId(-1).
		Body(larkdocx.NewCreateDocumentBlockDescendantReqBodyBuilder().
			ChildrenId(r.ChildrenID).
			Index(idx).
			Descendants(r.Descendants).
			Build()).Build()

	uat, err := f.GetUserAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get user access token failed: %w", err)
	}
	esp, err := f.client.Docx.V1.DocumentBlockDescendant.Create(context.Background(), req, larkcore.WithUserAccessToken(uat))
	if err != nil {
		return nil, fmt.Errorf("create document block descendant failed: %w", err)
	}
	if esp != nil {
		if esp.Code != 0 {
			log.Printf("raw body : %s", string(esp.RawBody))
			return nil, fmt.Errorf("create document block descendant failed: %s", esp.Msg)
		}
	}
	if esp.Data != nil {
		return esp.Data, nil
	}
	return nil, nil
}

// ListDocumentBlocksResponse 列出文档块的响应
type ListDocumentBlocksResponse struct {
	Code int                     `json:"code"`
	Msg  string                  `json:"msg"`
	Data *ListDocumentBlocksData `json:"data"`
}

// ListDocumentBlocksData 文档块数据
type ListDocumentBlocksData struct {
	HasMore   bool                 `json:"has_more"`
	Items     []*DocumentBlockItem `json:"items"`
	PageToken *string              `json:"page_token,omitempty"`
}

// DocumentBlockItem 文档块项
type DocumentBlockItem struct {
	BlockID   string     `json:"block_id"`
	BlockType int        `json:"block_type"`
	Children  []string   `json:"children,omitempty"`
	ParentID  string     `json:"parent_id,omitempty"`
	Page      *PageBlock `json:"page,omitempty"`
	Text      *TextBlock `json:"text,omitempty"`
}

// PageBlock 页面块
type PageBlock struct {
	Elements []*TextElement         `json:"elements"`
	Style    map[string]interface{} `json:"style"`
}

// TextBlock 文本块
type TextBlock struct {
	Elements []*TextElement         `json:"elements"`
	Style    map[string]interface{} `json:"style"`
}

// TextElement 文本元素
type TextElement struct {
	TextRun *TextRun `json:"text_run"`
}

// TextRun 文本运行
type TextRun struct {
	Content          string            `json:"content"`
	TextElementStyle *TextElementStyle `json:"text_element_style,omitempty"`
}

// TextElementStyle 文本元素样式
type TextElementStyle struct {
	BackgroundColor *int       `json:"background_color,omitempty"`
	TextColor       *int       `json:"text_color,omitempty"`
	Bold            bool       `json:"bold,omitempty"`
	Link            *LinkStyle `json:"link,omitempty"`
}

// LinkStyle 链接样式
type LinkStyle struct {
	URL string `json:"url"`
}

// RawContentResponse 获取文档纯文本内容的响应
type RawContentResponse struct {
	Code      int             `json:"code"`
	Msg       string          `json:"msg"`
	Data      *RawContentData `json:"data"`
	RequestId string          `json:"request_id,omitempty"`
}

// RawContentData 纯文本内容数据
type RawContentData struct {
	Content string `json:"content"`
}

// ListDocumentBlocksPage 获取单页文档块
// documentID: 文档ID
// pageSize: 每页大小，最大500
// pageToken: 分页token，第一次请求为空
// documentRevisionID: 文档版本ID，-1表示最新版本
func (f *FeishuDocxClient) ListDocumentBlocksPage(documentID string, pageSize int, pageToken string, documentRevisionID int) (*ListDocumentBlocksData, error) {
	req := larkdocx.NewListDocumentBlockReqBuilder().
		DocumentId(documentID).
		PageSize(pageSize).
		PageToken(pageToken).
		DocumentRevisionId(documentRevisionID).
		Build()

	uat, err := f.GetUserAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get user access token failed: %w", err)
	}

	resp, err := f.client.Docx.V1.DocumentBlock.List(context.Background(), req, larkcore.WithUserAccessToken(uat))
	if err != nil {
		return nil, fmt.Errorf("list document blocks failed: %w", err)
	}

	if !resp.Success() {
		log.Printf("raw body: %s", string(resp.RawBody))
		return nil, fmt.Errorf("list document blocks failed: code=%d, msg=%s, requestId=%s",
			resp.Code, resp.Msg, resp.RequestId())
	}

	// 直接使用 SDK 提供的响应数据
	if resp.Data == nil {
		return &ListDocumentBlocksData{
			HasMore:   false,
			Items:     nil,
			PageToken: nil,
		}, nil
	}

	// 转换 SDK 数据到我们的结构体
	items := make([]*DocumentBlockItem, 0, len(resp.Data.Items))
	for _, item := range resp.Data.Items {
		blockItem := &DocumentBlockItem{
			BlockID:   *item.BlockId,
			BlockType: *item.BlockType,
			Children:  make([]string, 0),
			ParentID:  "",
		}

		if item.Children != nil {
			for _, child := range item.Children {
				blockItem.Children = append(blockItem.Children, child)
			}
		}

		if item.ParentId != nil {
			blockItem.ParentID = *item.ParentId
		}

		// 这里可以添加 Page 和 Text 字段的转换，如果需要的话
		// 目前先保留为空，根据实际需求添加

		items = append(items, blockItem)
	}

	pageTokenPtr := resp.Data.PageToken
	if pageTokenPtr != nil && *pageTokenPtr == "" {
		pageTokenPtr = nil
	}

	return &ListDocumentBlocksData{
		HasMore:   *resp.Data.HasMore,
		Items:     items,
		PageToken: pageTokenPtr,
	}, nil
}

// ListAllDocumentBlocks 获取所有文档块（自动处理分页）
// documentID: 文档ID
// pageSize: 每页大小，最大500，建议使用500
func (f *FeishuDocxClient) ListAllDocumentBlocks(documentID string, pageSize int) ([]*DocumentBlockItem, error) {
	pageToken := ""
	allItems := make([]*DocumentBlockItem, 0)
	documentRevisionID := -1 // 使用最新版本

	for {
		data, err := f.ListDocumentBlocksPage(documentID, pageSize, pageToken, documentRevisionID)
		if err != nil {
			return nil, err
		}

		if data.Items != nil {
			allItems = append(allItems, data.Items...)
		}

		if !data.HasMore || data.PageToken == nil || *data.PageToken == "" {
			break
		}

		pageToken = *data.PageToken
	}

	return allItems, nil
}

// GetRawContent 获取文档的纯文本内容
// documentID: 文档ID
// lang: 语言代码（0表示中文，其他值参考飞书文档）
func (f *FeishuDocxClient) GetRawContent(documentID string, lang int) (*RawContentResponse, error) {
	req := larkdocx.NewRawContentDocumentReqBuilder().
		DocumentId(documentID).
		Lang(lang).
		Build()

	uat, err := f.GetUserAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get user access token failed: %w", err)
	}

	resp, err := f.client.Docx.V1.Document.RawContent(context.Background(), req, larkcore.WithUserAccessToken(uat))
	if err != nil {
		return nil, fmt.Errorf("get raw content failed: %w", err)
	}

	if !resp.Success() {
		log.Printf("raw body: %s", string(resp.RawBody))
		return nil, fmt.Errorf("get raw content failed: code=%d, msg=%s, requestId=%s",
			resp.Code, resp.Msg, resp.RequestId())
	}

	// 构建响应结构
	rawContentResp := &RawContentResponse{
		Code:      resp.Code,
		Msg:       resp.Msg,
		RequestId: resp.RequestId(),
	}

	// 检查是否有数据
	if resp.Data != nil && resp.Data.Content != nil {
		rawContentResp.Data = &RawContentData{
			Content: *resp.Data.Content,
		}
	}

	return rawContentResp, nil
}
