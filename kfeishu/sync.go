package kfeishu

import (
	"context"
	"fmt"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
)

// syncConfig 同步配置
type syncConfig struct {
	syncFieldName   string // 跨表关联字段名
	includeFields   []string // 仅同步的字段列表
	excludeFields   []string // 排除的字段列表
	deleteUnmatched bool // 是否删除目标表中多余记录
	batchSize       int // 批量操作大小
	sortField       string // 回流排序字段
	sortDesc        bool // 是否倒序
	limit           int // 回流记录数上限
}

func defaultSyncConfig() *syncConfig {
	return &syncConfig{
		syncFieldName: "SyncID",
		batchSize:     500,
	}
}

// SyncOption 同步配置选项
type SyncOption func(*syncConfig)

// WithSyncFieldName 设置跨表关联字段名，默认为 "SyncID"
func WithSyncFieldName(name string) SyncOption {
	return func(c *syncConfig) { c.syncFieldName = name }
}

// WithIncludeFields 仅同步指定字段，默认同步全部共有字段
func WithIncludeFields(fields []string) SyncOption {
	return func(c *syncConfig) { c.includeFields = fields }
}

// WithExcludeFields 排除指定字段不同步
func WithExcludeFields(fields []string) SyncOption {
	return func(c *syncConfig) { c.excludeFields = fields }
}

// WithDeleteUnmatched 同步时删除目标表中不存在于源表的记录
func WithDeleteUnmatched(v bool) SyncOption {
	return func(c *syncConfig) { c.deleteUnmatched = v }
}

// WithBatchSize 设置批量插入/更新/删除操作的大小，默认 500
func WithBatchSize(size int) SyncOption {
	return func(c *syncConfig) { c.batchSize = size }
}

// WithSortField 设置回流的排序字段和顺序（仅 SyncTableReverse 使用）
func WithSortField(field string, desc bool) SyncOption {
	return func(c *syncConfig) { c.sortField = field; c.sortDesc = desc }
}

// WithLimit 设置回流记录数上限（仅 SyncTableReverse 使用），0 表示全部
func WithLimit(limit int) SyncOption {
	return func(c *syncConfig) { c.limit = limit }
}

// SyncRecords 将外部数据批量同步到飞书多维表。
// 以 idField 为匹配字段，已有记录更新，新记录插入，重复记录保留最后一条。
func SyncRecords(client *FeishuAppTableClient, idField string, records []map[string]any, opts ...SyncOption) error {
	cfg := defaultSyncConfig()
	for _, o := range opts {
		o(cfg)
	}
	if len(records) == 0 {
		return nil
	}

	fields, err := client.GetTableFields()
	if err != nil {
		return fmt.Errorf("get table fields: %w", err)
	}
	fieldTypes := fieldsTypeMap(fields)
	if _, ok := fieldTypes[idField]; !ok {
		return fmt.Errorf("id field %q not found in table", idField)
	}

	existingRecords, err := client.ListAllRecords()
	if err != nil {
		return fmt.Errorf("list existing records: %w", err)
	}

	// 构建 ID → RecordID 映射
	idToRecordID := make(map[string]string)
	for _, r := range existingRecords {
		if r.Fields == nil || r.RecordId == nil {
			continue
		}
		if rid := ConvertData2String(r.Fields[idField]); rid != "" {
			idToRecordID[rid] = *r.RecordId
		}
	}

	deduped := dedupRecordsByID(records, idField)

	var updates map[string]map[string]any
	var adds []map[string]any

	for _, r := range deduped {
		rid := ConvertData2String(r[idField])
		conv := convertRecordValues(r, fieldTypes)
		if recordID, ok := idToRecordID[rid]; ok {
			if updates == nil {
				updates = make(map[string]map[string]any)
			}
			updates[recordID] = conv
		} else {
			adds = append(adds, conv)
		}
	}

	if len(updates) > 0 {
		if err := batchUpdateRecords(client, updates, cfg.batchSize); err != nil {
			return fmt.Errorf("update: %w", err)
		}
	}
	if len(adds) > 0 {
		if err := batchInsertRecords(client, adds, cfg.batchSize); err != nil {
			return fmt.Errorf("insert: %w", err)
		}
	}
	return nil
}

// SyncTableForward 从源表同步到目标表。
// 通过 syncFieldName（默认 "SyncID"）关联，该字段在目标表中存储源表 RecordID。
// 仅同步两张表共有的字段。
func SyncTableForward(src, dst *FeishuAppTableClient, opts ...SyncOption) error {
	cfg := defaultSyncConfig()
	for _, o := range opts {
		o(cfg)
	}

	srcFields, err := src.GetTableFields()
	if err != nil {
		return fmt.Errorf("get source fields: %w", err)
	}
	srcFieldTypes := fieldsTypeMap(srcFields)

	dstFields, err := dst.GetTableFields()
	if err != nil {
		return fmt.Errorf("get destination fields: %w", err)
	}
	dstFieldTypes := fieldsTypeMap(dstFields)

	if _, ok := dstFieldTypes[cfg.syncFieldName]; !ok {
		return fmt.Errorf("destination table is missing sync field %q", cfg.syncFieldName)
	}

	syncFields := resolveSyncFields(cfg.includeFields, cfg.excludeFields, srcFieldTypes, dstFieldTypes, cfg.syncFieldName)

	srcRecords, err := src.ListAllRecords()
	if err != nil {
		return fmt.Errorf("list source records: %w", err)
	}

	dstRecords, err := dst.ListAllRecords()
	if err != nil {
		return fmt.Errorf("list destination records: %w", err)
	}

	syncData := buildSyncDataMap(srcRecords, syncFields, cfg.syncFieldName)

	var updates map[string]map[string]any
	var adds []map[string]any
	var deletes []string

	for _, r := range dstRecords {
		if r.Fields == nil || r.RecordId == nil {
			continue
		}
		sid := ConvertData2String(r.Fields[cfg.syncFieldName])
		if sid == "" {
			continue
		}
		if data, ok := syncData[sid]; ok {
			if updates == nil {
				updates = make(map[string]map[string]any)
			}
			updates[*r.RecordId] = data
			delete(syncData, sid)
		} else if cfg.deleteUnmatched {
			deletes = append(deletes, *r.RecordId)
		}
	}

	for _, data := range syncData {
		adds = append(adds, data)
	}

	if len(updates) > 0 {
		if err := batchUpdateRecords(dst, updates, cfg.batchSize); err != nil {
			return fmt.Errorf("update: %w", err)
		}
	}
	if len(adds) > 0 {
		if err := batchInsertRecords(dst, adds, cfg.batchSize); err != nil {
			return fmt.Errorf("insert: %w", err)
		}
	}
	if len(deletes) > 0 {
		if err := batchDeleteRecords(dst, deletes, cfg.batchSize); err != nil {
			return fmt.Errorf("delete: %w", err)
		}
	}
	return nil
}

// SyncTableReverse 从子表回流字段到主表。
// 读取子表记录，通过 syncFieldName（存储主表 RecordID）匹配主表记录并更新指定字段。
//
// 配合 WithSortField + WithLimit 可只回流最近更新的记录而非全部。
func SyncTableReverse(main, sub *FeishuAppTableClient, syncFields []string, opts ...SyncOption) error {
	if len(syncFields) == 0 {
		return fmt.Errorf("syncFields is required")
	}
	cfg := defaultSyncConfig()
	for _, o := range opts {
		o(cfg)
	}

	subFieldTypes, err := sub.GetTableFields()
	if err != nil {
		return fmt.Errorf("get sub table fields: %w", err)
	}
	fieldTypeMap := fieldsTypeMap(subFieldTypes)

	var subRecords []*larkbitable.AppTableRecord
	if cfg.sortField != "" {
		subRecords, err = sub.SearchRecords(nil, cfg.limitOrDefault(500), []SortArg{
			{FieldName: cfg.sortField, Desc: cfg.sortDesc},
		}, nil, 1)
	} else if cfg.limit > 0 {
		subRecords, err = sub.ListAllRecords()
		if err == nil && len(subRecords) > cfg.limit {
			subRecords = subRecords[:cfg.limit]
		}
	} else {
		subRecords, err = sub.ListAllRecords()
	}
	if err != nil {
		return fmt.Errorf("list sub records: %w", err)
	}

	updateData := make(map[string]map[string]any)
	for _, r := range subRecords {
		if r.Fields == nil || r.RecordId == nil {
			continue
		}
		syncID := ConvertData2String(r.Fields[cfg.syncFieldName])
		if syncID == "" {
			continue
		}
		data := make(map[string]any)
		for _, field := range syncFields {
			v, ok := r.Fields[field]
			if !ok {
				continue
			}
			ft, exists := fieldTypeMap[field]
			if !exists {
				continue
			}
			nv, err := ConvertInterface2TypeValue(v, ft)
			if err != nil {
				data[field] = v
			} else {
				data[field] = nv
			}
		}
		if len(data) > 0 {
			updateData[syncID] = data
		}
	}
	if len(updateData) == 0 {
		return nil
	}

	mainRecords, err := main.ListAllRecords()
	if err != nil {
		return fmt.Errorf("list main records: %w", err)
	}

	matched := make(map[string]map[string]any)
	for _, r := range mainRecords {
		if r.RecordId == nil {
			continue
		}
		if data, ok := updateData[*r.RecordId]; ok {
			matched[*r.RecordId] = data
		}
	}
	if len(matched) == 0 {
		return nil
	}
	return batchUpdateRecords(main, matched, cfg.batchSize)
}

func (c *syncConfig) limitOrDefault(def int) int {
	if c.limit > 0 {
		return c.limit
	}
	return def
}

// SearchRecords 按指定条件搜索记录，支持排序、字段筛选和分页。
// maxPages 为 0 时获取全部页。
func (f *FeishuAppTableClient) SearchRecords(filter *larkbitable.FilterInfo, pageSize int, sortFields []SortArg, fieldNames []string, maxPages int) ([]*larkbitable.AppTableRecord, error) {
	var allRecords []*larkbitable.AppTableRecord
	pageToken := ""

	var sorts []*larkbitable.Sort
	for _, s := range sortFields {
		sorts = append(sorts, larkbitable.NewSortBuilder().
			FieldName(s.FieldName).
			Desc(s.Desc).
			Build())
	}

	for page := 0; page < maxPages || maxPages == 0; page++ {
		bodyBuilder := larkbitable.NewSearchAppTableRecordReqBodyBuilder().
			Filter(filter).FieldNames(fieldNames).Sort(sorts).
			AutomaticFields(false)

		req := larkbitable.NewSearchAppTableRecordReqBuilder().
			AppToken(f.AppToken).TableId(f.TableID).
			PageSize(pageSize).PageToken(pageToken).
			Body(bodyBuilder.Build()).Build()

		uat, err := f.GetUserAccessToken()
		if err != nil {
			return nil, err
		}
		resp, err := f.client.Bitable.AppTableRecord.Search(context.Background(), req, larkcore.WithUserAccessToken(uat))
		if err != nil {
			return nil, err
		}
		if !resp.Success() {
			return nil, fmt.Errorf("search records failed: code=%d, msg=%s, requestId=%s", resp.Code, resp.Msg, resp.RequestId())
		}
		if resp.Data == nil || resp.Data.Items == nil {
			break
		}
		allRecords = append(allRecords, resp.Data.Items...)
		if resp.Data.PageToken != nil && *resp.Data.PageToken != "" {
			pageToken = *resp.Data.PageToken
		} else {
			break
		}
		if !*resp.Data.HasMore {
			break
		}
	}
	return allRecords, nil
}

// SortArg 搜索排序条件
type SortArg struct {
	FieldName string // 排序字段名
	Desc      bool   // 是否倒序
}

func fieldsTypeMap(fields []*larkbitable.AppTableFieldForList) map[string]int {
	m := make(map[string]int, len(fields))
	for _, f := range fields {
		if f.FieldName != nil && f.Type != nil {
			m[*f.FieldName] = *f.Type
		}
	}
	return m
}

// dedupRecordsByID 按 idField 去重，保留最后一条
func dedupRecordsByID(records []map[string]any, idField string) []map[string]any {
	seen := make(map[string]map[string]any)
	var result []map[string]any
	for _, r := range records {
		rid := ConvertData2String(r[idField])
		if rid == "" {
			result = append(result, r)
			continue
		}
		seen[rid] = r
	}
	for _, r := range seen {
		result = append(result, r)
	}
	return result
}

// convertRecordValues 按表字段类型转换记录值，转换失败保留原值
func convertRecordValues(record map[string]any, fieldTypes map[string]int) map[string]any {
	out := make(map[string]any, len(record))
	for k, v := range record {
		ft, ok := fieldTypes[k]
		if !ok {
			continue
		}
		nv, err := ConvertInterface2TypeValue(v, ft)
		if err != nil {
			out[k] = v
		} else {
			out[k] = nv
		}
	}
	return out
}

// resolveSyncFields 根据 include/exclude、源目标共有、排除关联字段后确定需要同步的字段
func resolveSyncFields(include, exclude []string, srcTypes, dstTypes map[string]int, syncFieldName string) map[string]int {
	includeSet := make(map[string]struct{})
	if len(include) > 0 {
		for _, f := range include {
			includeSet[f] = struct{}{}
		}
	} else {
		for f := range srcTypes {
			includeSet[f] = struct{}{}
		}
	}

	for _, f := range exclude {
		delete(includeSet, f)
	}
	delete(includeSet, syncFieldName)
	delete(includeSet, "ID")

	result := make(map[string]int)
	for f := range includeSet {
		if _, ok := srcTypes[f]; !ok {
			continue
		}
		if _, ok := dstTypes[f]; !ok {
			continue
		}
		result[f] = srcTypes[f]
	}
	return result
}

// buildSyncDataMap 从源表记录构建同步数据，key 为源表 RecordID
func buildSyncDataMap(records []*larkbitable.AppTableRecord, syncFields map[string]int, syncFieldName string) map[string]map[string]any {
	result := make(map[string]map[string]any)
	for _, r := range records {
		if r.Fields == nil || r.RecordId == nil {
			continue
		}
		data := make(map[string]any, len(syncFields)+1)
		for fieldName := range syncFields {
			v, ok := r.Fields[fieldName]
			if !ok {
				continue
			}
			nv, err := ConvertInterface2TypeValue(v, syncFields[fieldName])
			if err != nil {
				data[fieldName] = v
			} else {
				data[fieldName] = nv
			}
		}
		data[syncFieldName] = *r.RecordId
		result[*r.RecordId] = data
	}
	return result
}

// batchUpdateRecords 分批更新记录
func batchUpdateRecords(client *FeishuAppTableClient, records map[string]map[string]any, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 500
	}
	type item struct {
		id   string
		data map[string]any
	}
	items := make([]item, 0, len(records))
	for id, data := range records {
		items = append(items, item{id, data})
	}
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batch := make(map[string]map[string]any, end-i)
		for _, it := range items[i:end] {
			batch[it.id] = it.data
		}
		if err := client.UpdateRecords(batch); err != nil {
			return err
		}
	}
	return nil
}

// batchInsertRecords 分批插入记录
func batchInsertRecords(client *FeishuAppTableClient, records []map[string]any, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 500
	}
	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		if err := client.InsertRecords(records[i:end]); err != nil {
			return err
		}
	}
	return nil
}

// batchDeleteRecords 分批删除记录
func batchDeleteRecords(client *FeishuAppTableClient, recordIDs []string, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 500
	}
	for i := 0; i < len(recordIDs); i += batchSize {
		end := i + batchSize
		if end > len(recordIDs) {
			end = len(recordIDs)
		}
		if err := client.DeleteRecords(recordIDs[i:end]); err != nil {
			return err
		}
	}
	return nil
}
