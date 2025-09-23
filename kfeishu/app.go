package kfeishu

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kevin-zx/kbase/kfeishu/token"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
)

type FeishuAppTableClient struct {
	feishuClient
	AppToken   string
	TableID    string
	MaxRetries int           // 最大重试次数
	RetryDelay time.Duration // 重试间隔
}

func (f *FeishuAppTableClient) shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "code: 1254607") {
		return true
	}
	return false
}

func (f *FeishuAppTableClient) withRetry(fn func() error) error {
	var lastErr error
	for i := 0; i < f.MaxRetries; i++ {
		lastErr = fn()
		if lastErr == nil || !f.shouldRetry(lastErr) {
			return lastErr
		}

		log.Printf("Retry attempt %d/%d after error: %v", i+1, f.MaxRetries, lastErr)
		if i < f.MaxRetries-1 {
			time.Sleep(f.RetryDelay)
		}
	}
	return lastErr
}

// new feishu app table client
// appId: app id
// appSecret: app secret
// appToken: app token 多维表的app token
// tableId: table id 多维表的table id
func NewFeishuAppClient(ts token.TokenService, appId, appSecret, appToken, tableId string) *FeishuAppTableClient {
	return NewFeishuAppClientWithRetry(ts, appId, appSecret, appToken, tableId, 5, time.Minute)
}

// new feishu app table client with retry config
// appId: app id
// appSecret: app secret
// appToken: app token 多维表的app token
// tableId: table id 多维表的table id
// maxRetries: 最大重试次数
// retryDelay: 重试间隔
func NewFeishuAppClientWithRetry(ts token.TokenService, appID, appSecret, appToken, tableID string, maxRetries int, retryDelay time.Duration) *FeishuAppTableClient {
	return &FeishuAppTableClient{
		feishuClient: NewFeishuClient(ts, appID, appSecret),
		AppToken:     appToken,
		TableID:      tableID,
		MaxRetries:   maxRetries,
		RetryDelay:   retryDelay,
	}
}

func (f *FeishuAppTableClient) Close() {
	f.ts.Close()
}

// list all records in the table
func (f *FeishuAppTableClient) ListAllRecords() ([]*larkbitable.AppTableRecord, error) {
	pageSize := 500
	pageToken := ""
	records := make([]*larkbitable.AppTableRecord, 0)
	for {
		pageData, err := f.ListRecordsPage(pageSize, pageToken)
		if err != nil {
			return nil, err
		}
		records = append(records, pageData.Records...)
		if !pageData.HasMore {
			break
		}
		pageToken = pageData.NextPageToken
	}
	return records, nil
}

type PageData struct {
	Records       []*larkbitable.AppTableRecord
	NextPageToken string
	HasMore       bool
}

// update records in the table
func (f *FeishuAppTableClient) UpdateRecords(records map[string]map[string]any) error {
	return f.withRetry(func() error {
		fields, err := f.GetTableFields()
		if err != nil {
			return err
		}

		recordDatas := make([]*larkbitable.AppTableRecord, 0)
		for rid, record := range records {
			err = f.CheckFields(fields, record)
			if err != nil {
				log.Printf("check fields error, data: %+v", record)
				return err
			}
			recordData := larkbitable.NewAppTableRecordBuilder().
				Fields(record).
				RecordId(rid).
				Build()
			recordDatas = append(recordDatas, recordData)
		}

		req := larkbitable.NewBatchUpdateAppTableRecordReqBuilder().
			Body(larkbitable.NewBatchUpdateAppTableRecordReqBodyBuilder().
				Records(recordDatas).
				Build()).
			AppToken(f.AppToken).
			TableId(f.TableID).
			Build()
		uat, err := f.GetUserAccessToken()
		if err != nil {
			fmt.Printf("%+v", records)
			return err
		}
		// 发起请求
		resp, err := f.client.Bitable.AppTableRecord.BatchUpdate(context.Background(), req, larkcore.WithUserAccessToken(uat))
		if err != nil {
			fmt.Printf("raw body: %s\n", string(resp.ApiResp.RawBody))
			fmt.Printf("%+v", records)
			return err
		}

		if !resp.Success() {
			fmt.Printf("raw body: %s\n", string(resp.ApiResp.RawBody))
			return fmt.Errorf("code: %d, msg: %s, requestId: %s", resp.Code, resp.Msg, resp.RequestId())
		}
		return nil
	})
}

// delete all records in the table
func (f *FeishuAppTableClient) DeleteAllRecords() error {
	return f.withRetry(func() error {
		allRecords, err := f.ListAllRecords()
		if err != nil {
			return err
		}
		records := make([]string, 0)
		if len(allRecords) == 0 {
			return nil
		}
		batch := 100
		for _, record := range allRecords {
			records = append(records, *record.RecordId)
			if len(records) == batch {
				err = f.DeleteRecords(records)
				if err != nil {
					return err
				}
				records = make([]string, 0)
			}
		}

		if len(records) > 0 {
			err = f.DeleteRecords(records)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (f *FeishuAppTableClient) DeleteRecords(records []string) error {
	return f.withRetry(func() error {
		req := larkbitable.NewBatchDeleteAppTableRecordReqBuilder().
			AppToken(f.AppToken).
			TableId(f.TableID).
			Body(
				larkbitable.NewBatchDeleteAppTableRecordReqBodyBuilder().
					Records(records).
					Build()).
			Build()
		// 发起请求
		uat, err := f.GetUserAccessToken()
		if err != nil {
			return err
		}
		resp, err := f.client.Bitable.AppTableRecord.BatchDelete(
			context.Background(),
			req,
			larkcore.WithUserAccessToken(uat),
		)
		if err != nil {
			return err
		}

		if !resp.Success() {
			return fmt.Errorf("code: %d, msg: %s, requestId: %s", resp.Code, resp.Msg, resp.RequestId())
		}
		return nil
	})
}

// list one page of records in the table
func (f *FeishuAppTableClient) ListRecordsPage(pageSize int, pageToken string) (*PageData, error) {
	var result *PageData
	err := f.withRetry(func() error {
		var innerErr error
		result, innerErr = f.listRecordsPage(pageSize, pageToken)
		return innerErr
	})
	return result, err
}

func (f *FeishuAppTableClient) listRecordsPage(pageSize int, pageToken string) (*PageData, error) {
	req := larkbitable.NewListAppTableRecordReqBuilder().
		AppToken(f.AppToken).
		TableId(f.TableID).
		PageSize(pageSize).
		PageToken(pageToken).
		Build()

	uat, err := f.GetUserAccessToken()
	if err != nil {
		return nil, err
	}
	resp, err := f.client.Bitable.AppTableRecord.List(
		context.Background(),
		req,
		larkcore.WithUserAccessToken(uat),
	)
	if err != nil {
		return nil, err
	}

	if !resp.Success() {
		log.Printf("raw body: %s", string(resp.ApiResp.RawBody))
		return nil, fmt.Errorf("code: %d, msg: %s, requestId: %s", resp.Code, resp.Msg, resp.RequestId())
	}

	pageToken = ""
	if resp.Data.PageToken != nil {
		pageToken = *resp.Data.PageToken
	}
	return &PageData{
		Records:       resp.Data.Items,
		NextPageToken: pageToken,
		HasMore:       *resp.Data.HasMore,
	}, nil

}

// insert records to the table
func (f *FeishuAppTableClient) InsertRecords(records []map[string]interface{}) error {
	return f.withRetry(func() error {

		fields, err := f.GetTableFields()
		if err != nil {
			return err
		}

		recordDatas := make([]*larkbitable.AppTableRecord, 0)
		for _, record := range records {
			err = f.CheckFields(fields, record)
			if err != nil {
				log.Printf("check fields error, data: %+v", record)
				return err
			}
			recordData := larkbitable.NewAppTableRecordBuilder().
				Fields(record).
				Build()
			recordDatas = append(recordDatas, recordData)
		}

		req := larkbitable.NewBatchCreateAppTableRecordReqBuilder().
			Body(larkbitable.NewBatchCreateAppTableRecordReqBodyBuilder().
				Records(recordDatas).
				Build()).
			AppToken(f.AppToken).
			TableId(f.TableID).
			Build()
		userAccessToken, err := f.GetUserAccessToken()
		if err != nil {
			return err
		}
		resp, err := f.client.Bitable.AppTableRecord.BatchCreate(context.Background(), req, larkcore.WithUserAccessToken(userAccessToken))
		if err != nil {
			if resp != nil && resp.ApiResp != nil {
				log.Printf("raw body: %s", string(resp.ApiResp.RawBody))
				log.Printf("tableId: %s, appToken: %s", f.TableID, f.AppToken)
			}
			return err
		}

		if !resp.Success() {
			for _, record := range records {
				log.Printf("record: %+v", record)
			}
			log.Printf("raw body: %s", string(resp.ApiResp.RawBody))
			log.Printf("tableId: %s, appToken: %s", f.TableID, f.AppToken)
			return fmt.Errorf("code: %d, msg: %s, requestId: %s", resp.Code, resp.Msg, resp.RequestId())
		}

		log.Printf("Successfully inserted records")
		return nil
	})
}

// check fileds is valid
func (f *FeishuAppTableClient) CheckFields(fields []*larkbitable.AppTableFieldForList, record map[string]interface{}) error {

	for key := range record {
		found := false
		for _, field := range fields {
			if key == *field.FieldName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("field %s not found", key)
		}
	}
	return nil
}

func (f *FeishuAppTableClient) ListTables() ([]*larkbitable.AppTable, error) {
	req := larkbitable.NewListAppTableReqBuilder().
		AppToken(f.AppToken).
		PageSize(100).
		Build()

	uat, err := f.GetUserAccessToken()
	if err != nil {
		return nil, err
	}
	// 发起请求
	resp, err := f.client.Bitable.AppTable.List(context.Background(), req, larkcore.WithUserAccessToken(uat))
	if err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("code: %d, msg: %s", resp.Code, resp.Msg)
	}

	return resp.Data.Items, nil
}

// get fileds of the table
func (f *FeishuAppTableClient) GetTableFields() ([]*larkbitable.AppTableFieldForList, error) {

	req := larkbitable.NewListAppTableFieldReqBuilder().
		AppToken(f.AppToken).
		TableId(f.TableID).
		PageSize(200).
		Build()
	// 发起请求
	uat, err := f.GetUserAccessToken()
	if err != nil {
		return nil, err
	}
	resp, err := f.client.Bitable.AppTableField.List(context.Background(), req, larkcore.WithUserAccessToken(uat))
	if err != nil {
		return nil, err
	}

	if !resp.Success() {
		return nil, fmt.Errorf("code: %d, msg: %s, requestId: %s", resp.Code, resp.Msg, resp.RequestId())
	}
	return resp.Data.Items, nil
}

// add filed to table
func (f *FeishuAppTableClient) AddField(field *SimpleField) (*larkbitable.AppTableField, error) {

	appField := field.Build()

	uat, err := f.GetUserAccessToken()
	if err != nil {
		return nil, err
	}
	req := larkbitable.NewCreateAppTableFieldReqBuilder().
		AppTableField(appField).Build()
	response, err := f.client.Bitable.AppTableField.Create(context.Background(), req, larkcore.WithUserAccessToken(uat))
	if err != nil {
		return nil, err
	}

	if !response.Success() {
		return nil, fmt.Errorf("code: %d, msg: %s, requestId: %s", response.Code, response.Msg, response.RequestId())
	}
	return response.Data.Field, nil
}

// query
// filter: filter info
// pageSize: page size
// pageToken: page token
// returns: records, next page token, error
func (f *FeishuAppTableClient) Query(filter larkbitable.FilterInfo, pageSize int, pageToken string) ([]*larkbitable.AppTableRecord, string, error) {
	req := larkbitable.NewSearchAppTableRecordReqBuilder().
		AppToken(f.AppToken).
		TableId(f.TableID).
		PageSize(pageSize).
		Body(
			larkbitable.NewSearchAppTableRecordReqBodyBuilder().
				Filter(&filter).
				AutomaticFields(false).
				Build()).
		Build()

	uat, err := f.GetUserAccessToken()
	if err != nil {
		return nil, "", err
	}
	// 发起请求
	resp, err := f.client.Bitable.AppTableRecord.Search(context.Background(), req, larkcore.WithUserAccessToken(uat))
	if err != nil {
		return nil, "", err
	}

	if !resp.Success() {
		return nil, "", fmt.Errorf("code: %d, msg: %s, requestId: %s", resp.Code, resp.Msg, resp.RequestId())
	}

	return resp.Data.Items, *resp.Data.PageToken, nil
}

// qeury with max page size
func (f *FeishuAppTableClient) QueryAll(filter larkbitable.FilterInfo, pageSize, maxPageSize int) ([]*larkbitable.AppTableRecord, error) {

	pt := ""
	records := make([]*larkbitable.AppTableRecord, 0)
	for page := 1; page <= maxPageSize; page++ {

		records, nextPageToken, err := f.Query(filter, pageSize, pt)
		if err != nil {
			return nil, err
		}

		if len(records) == 0 {
			break
		}

		pt = nextPageToken
		for _, record := range records {
			records = append(records, record)
		}
		if pt == "" {
			break
		}
		if len(records) < pageSize {
			break
		}
	}

	return records, nil

}
