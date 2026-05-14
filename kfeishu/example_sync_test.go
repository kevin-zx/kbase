package kfeishu_test

import (
	"fmt"
	"testing"

	"github.com/kevin-zx/kbase/kfeishu"
	"github.com/kevin-zx/kbase/kfeishu/token"
)

// —— 前置准备 ——
// 所有同步操作都需要创建 FeishuAppTableClient。
// 参数说明：
//   ts       - token.TokenService，管理飞书用户授权令牌
//   appID    - 飞书应用的 App ID
//   appSecret- 飞书应用的 App Secret
//   appToken - 目标多维表格的 app_token（从飞书多维表格 URL 中获取）
//   tableID  - 目标表的 table_id（从飞书多维表格 URL 中获取）

func newClientForTest(t *testing.T, ts token.TokenService, appToken, tableID string) *kfeishu.FeishuAppTableClient {
	t.Helper()
	return kfeishu.NewFeishuAppClient(
		ts,
		"cli_xxxxxxxxxxxx",
		"xxxxxxxxxxxx",
		appToken,
		tableID,
	)
}

// =============================================================================
// SyncRecords —— 将外部数据批量同步到飞书多维表
//
// 场景：你有一个外部数据源（数据库查询、API 返回、CSV 解析等），需要批量导入到飞书多维表。
// 以 idField 为匹配字段，表中已有该 ID 的记录则更新字段值，没有则插入新记录。
// 重复 ID 的记录自动去重，保留最后一条。
//
// 关键参数：
//   idField - 用于匹配已有记录的字段名（必须存在于目标表中）
//   records - 要同步的记录列表，每条记录为 map[string]any
//   opts    - 可选配置：WithBatchSize 调整批量大小等
//
// 注意：外部数据字段值会按目标表的字段类型自动转换。
//       比如数字字段传 float64，文本字段传 string，日期字段传毫秒时间戳，
//       人员字段传 *FeishuUser，链接字段传 map[string]string{"link":"...", "text":"..."}。
//       无法转换的类型会保留原值透传。
// =============================================================================
func TestSyncRecords(t *testing.T) {
	t.Skip("需要飞书凭证才能运行，此处仅展示用法。使用时去掉 t.Skip() 并填入真实凭证")

	// 1. 创建目标表客户端
	client := newClientForTest(t, nil, "bascnxxxxxxxxxxxx", "tblxxxxxxxxxxxx")

	// 2. 准备要同步的数据（模拟从数据库查询到的商品数据）
	records := []map[string]any{
		{
			"SKU":    "SKU-001",
			"商品名":   "飞书多维表同步指南",
			"价格":    99.00,                // 数字字段，传 float64
			"库存":    100,                   // 数字字段，传 int 亦可自动转换
			"是否上架":  true,                 // 复选框字段
			"上架日期":  1715692800000,        // 日期字段，传毫秒时间戳
			"分类":    "工具书",               // 单选/文本字段
			"商品链接":  map[string]string{     // 超链接字段
				"link": "https://example.com/sku001",
				"text": "查看详情",
			},
		},
		{
			"SKU":   "SKU-002",
			"商品名":  "Go 语言高级编程",
			"价格":   128.00,
			"库存":   50,
			"是否上架": true,
			"上架日期": 1715692800000,
			"分类":   "技术书",
		},
		{
			// SKU-001 重复 —— SyncRecords 会自动去重，保留后出现的这条
			"SKU":   "SKU-001",
			"商品名":  "飞书多维表同步指南（修订版）",
			"价格":   89.00,
			"库存":   120,
			"是否上架": true,
			"上架日期": 1715692800000,
			"分类":   "工具书",
		},
	}

	// 3. 执行同步
	//    以 "SKU" 字段匹配已有记录：表中已有 SKU-001 → 更新；没有 SKU-002 → 插入
	err := kfeishu.SyncRecords(client, "SKU", records,
		kfeishu.WithBatchSize(200), // 每批最多 200 条（飞书多维表单批上限 500）
	)
	if err != nil {
		t.Fatalf("同步失败: %v", err)
	}
	t.Log("同步完成 —— SKU-001 已更新，SKU-002 已插入")
}

// =============================================================================
// SyncTableForward —— 主表同步到子表（正向同步）
//
// 场景：两个飞书多维表存在关联关系，需要将主表数据同步到子表。
// 子表中需要有一个字段（默认名 "SyncID"）存储主表的 RecordID，用于关联。
//
// 正向同步的行为：
//   - 主表中有、子表中无 → 在子表中插入新记录，同时写入关联字段
//   - 主表中有、子表中也有 → 更新子表已有记录的同步字段
//   - 主表中无、子表中有 → 由 WithDeleteUnmatched 控制是否删除子表多余记录
//
// 子表结构要求：
//   子表必须有一个字段（如 "SyncID" 或自定义名）用于存储主表的 RecordID
// =============================================================================
func TestSyncTableForward(t *testing.T) {
	t.Skip("需要飞书凭证才能运行，此处仅展示用法")

	// 主表：需求池（包含完整的需求信息）
	srcClient := newClientForTest(t, nil, "bascn_xxxxxx", "tbl_main")

	// 子表：迭代看板（包含关联字段 + 部分字段）
	dstClient := newClientForTest(t, nil, "bascn_xxxxxx", "tbl_sub")

	err := kfeishu.SyncTableForward(srcClient, dstClient,
		// 关联字段名：子表中存储主表 RecordID 的字段
		kfeishu.WithSyncFieldName("主表关联"),

		// 只同步这三个字段，其余字段保持不变
		kfeishu.WithIncludeFields([]string{"需求标题", "负责人", "优先级"}),

		// 主表已删除的记录，子表也一并删除
		kfeishu.WithDeleteUnmatched(true),

		// 批量操作大小
		kfeishu.WithBatchSize(300),
	)
	if err != nil {
		t.Fatalf("正向同步失败: %v", err)
	}
	t.Log("正向同步完成")
}

// =============================================================================
// SyncTableForward —— 排除系统字段不同步
//
// 有些字段（如创建时间、创建人、最后更新时间、修改人）通常不应该被外部同步覆盖。
// 用 WithExcludeFields 排除它们。
// 注意：WithIncludeFields 和 WithExcludeFields 同时用时，exclude 优先生效。
// =============================================================================
func TestSyncTableForward_excludeFields(t *testing.T) {
	t.Skip("需要飞书凭证才能运行，此处仅展示用法")

	srcClient := newClientForTest(t, nil, "bascn_xxxxxx", "tbl_src")
	dstClient := newClientForTest(t, nil, "bascn_xxxxxx", "tbl_dst")

	err := kfeishu.SyncTableForward(srcClient, dstClient,
		kfeishu.WithSyncFieldName("SyncID"),
		// 不同步以下系统自动字段
		kfeishu.WithExcludeFields([]string{"创建时间", "创建人", "最后更新时间", "修改人"}),
	)
	if err != nil {
		t.Fatalf("同步失败: %v", err)
	}
	t.Log("同步完成（已排除系统字段）")
}

// =============================================================================
// SyncTableReverse —— 子表回流到主表（反向同步）
//
// 场景：子表中某些字段（如处理状态、备注等）发生变更后，同步回主表对应字段。
// 通过 syncFieldName 存储主表 RecordID 来匹配主表记录。
//
// 典型用法：
//   - 「工程师」在子表更新了 bug 的「处理状态」后，回流到主表的「当前状态」
//   - 「标注人员」在子表填写了「标注结果」「置信度」后，回流到主表
//
// 性能提示：
//   对于数据量大的子表，建议配合 WithSortField + WithLimit 只回流最近更新的记录。
//   不指定时默认拉取子表全部记录再匹配。
// =============================================================================
func TestSyncTableReverse_recent(t *testing.T) {
	t.Skip("需要飞书凭证才能运行，此处仅展示用法")

	mainClient := newClientForTest(t, nil, "bascn_xxxxxx", "tbl_main")
	subClient := newClientForTest(t, nil, "bascn_xxxxxx", "tbl_sub")

	// 回流子表的「处理状态」和「处理备注」到主表对应字段
	err := kfeishu.SyncTableReverse(mainClient, subClient,
		[]string{"处理状态", "处理备注"}, // 需要回流到主表的字段列表

		// 子表中存储主表 RecordID 的字段名
		kfeishu.WithSyncFieldName("主表关联"),

		// 按「最后更新时间」倒序，只取最近 100 条
		// 避免每次回流都全量拉取子表数据
		kfeishu.WithSortField("最后更新时间", true),
		kfeishu.WithLimit(100),
	)
	if err != nil {
		t.Fatalf("回流失败: %v", err)
	}
	t.Log("回流完成 —— 最近 100 条子表记录的「处理状态」和「处理备注」已同步回主表")
}

// =============================================================================
// SyncTableReverse —— 全量回流（数据量小的场景）
//
// 不指定 WithSortField 和 WithLimit 时，会拉取子表全部记录进行匹配和回流。
// 适用于子表数据量较小（几百到几千条）的场景。
// =============================================================================
func TestSyncTableReverse_all(t *testing.T) {
	t.Skip("需要飞书凭证才能运行，此处仅展示用法")

	mainClient := newClientForTest(t, nil, "bascn_xxxxxx", "tbl_main")
	subClient := newClientForTest(t, nil, "bascn_xxxxxx", "tbl_sub")

	// 默认拉取子表全部记录
	err := kfeishu.SyncTableReverse(mainClient, subClient,
		[]string{"状态", "备注"},
		kfeishu.WithSyncFieldName("关联主表ID"),
	)
	if err != nil {
		t.Fatalf("全量回流失败: %v", err)
	}
	t.Log("全量回流完成")
}

// =============================================================================
// SearchRecords —— 高级搜索（带排序和字段筛选）
//
// SearchRecords 是 sync.go 为 FeishuAppTableClient 新增的方法。
//
// 相比 ListAllRecords，SearchRecords 支持：
//   - 排序（SortArg），可指定多个排序字段
//   - 字段筛选（fieldNames），只返回需要的字段减少数据量
//   - 筛选条件（filter），按条件过滤记录
//   - 分页控制（maxPages），0 表示获取全部页
//
// 注意：filter 类型为 *larkbitable.FilterInfo，传 nil 表示不过滤。
// =============================================================================
func TestSearchRecords(t *testing.T) {
	t.Skip("需要飞书凭证才能运行，此处仅展示用法")

	client := newClientForTest(t, nil, "bascn_xxxxxx", "tbl_main")

	// 按「创建时间」倒序，只返回「标题」「状态」两个字段，只取第一页
	records, err := client.SearchRecords(
		nil, // 无筛选条件
		200, // 每页 200 条
		[]kfeishu.SortArg{
			{FieldName: "创建时间", Desc: true}, // 创建时间倒序（最新的在前）
		},
		[]string{"标题", "状态"}, // 只返回这两个字段，传 nil 返回全部
		1, // 只取第 1 页
	)
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}

	t.Logf("搜索结果: 共 %d 条记录", len(records))
	for _, r := range records {
		title := ""
		status := ""
		if r.Fields != nil {
			title = fmt.Sprint(r.Fields["标题"])
			status = fmt.Sprint(r.Fields["状态"])
		}
		t.Logf("  [%s] %s", status, title)
	}

	// 如果要获取全部页，传 maxPages=0：
	// allRecords, _ := client.SearchRecords(nil, 500, sorts, nil, 0)
}

// =============================================================================
// 完整工作流示例：数据库表 → 飞书多维表定时同步
//
// 这是最常见的同步场景：定时将数据库的数据全量或增量同步到飞书多维表。
// 示例演示了完整的流程：
//   1. 从数据库查询数据
//   2. 按飞书字段格式转换
//   3. 调用 SyncRecords 完成同步
// =============================================================================
func TestFullWorkflow_databaseToFeishu(t *testing.T) {
	t.Skip("需要飞书凭证 + 数据库连接才能运行，此处仅展示流程")

	// 假设有一个 MySQL 表 products：
	//   id | name | price | stock | category | is_online | created_at
	//
	// 对应的飞书多维表字段：
	//   SKU | 商品名 | 价格 | 库存 | 分类 | 是否上架 | 上架日期

	// 1. 创建目标飞书表客户端
	client := newClientForTest(t, nil, "bascn_xxxxxx", "tbl_products")

	// 2. 从数据库查询数据（示意）
	type product struct {
		SKU      string
		Name     string
		Price    float64
		Stock    int
		Category string
		IsOnline bool
		CreateAt int64 // unix 毫秒时间戳
	}

	// 模拟数据库查询结果
	dbResult := []product{
		{SKU: "P001", Name: "商品A", Price: 99.0, Stock: 100, Category: "电子", IsOnline: true, CreateAt: 1715692800000},
		{SKU: "P002", Name: "商品B", Price: 199.0, Stock: 50, Category: "家居", IsOnline: false, CreateAt: 1715779200000},
	}

	// 3. 转换为飞书记录格式（map[string]any）
	records := make([]map[string]any, 0, len(dbResult))
	for _, p := range dbResult {
		records = append(records, map[string]any{
			"SKU":   p.SKU,
			"商品名":  p.Name,
			"价格":   p.Price,
			"库存":   p.Stock,
			"分类":   p.Category,
			"是否上架": p.IsOnline,
			"上架日期": p.CreateAt,
		})
	}

	// 4. 同步
	err := kfeishu.SyncRecords(client, "SKU", records,
		kfeishu.WithBatchSize(500),
	)
	if err != nil {
		t.Fatalf("同步失败: %v", err)
	}
	t.Logf("同步完成: 共 %d 条记录", len(records))
}
