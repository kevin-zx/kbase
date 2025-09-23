package kfeishu

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
)

type TypeValue int

const (
	TypeValueText         TypeValue = iota + 1 // 1 多行文本
	TypeValueNumber                            // 2 数字
	TypeValueSingleSelect                      // 3 单选
	TypeValueMultiSelect                       // 4 多选
	TypeValueDateTime                          // 5 日期
	TypeValueCheckbox                          // 7 复选框
	TypeValueUser                              // 11 人员
	TypeValuePhone                             // 13 电话号码
	TypeValueUrl                               // 15 超链接
	TypeValueAttachment                        // 17 附件
	TypeValueSingleLink                        // 18 单向关联
	TypeValueLookup                            // 19 查找引用
	TypeValueFormula                           // 20 公式
	TypeValueDuplexLink                        // 21 双向关联
	TypeValueLocation                          // 22 地理位置
	TypeValueGroupChat                         // 23 群组
	TypeValueCreatedTime                       // 1001 创建时间
	TypeValueModifiedTime                      // 1002 最后更新时间
	TypeValueCreatedUser                       // 1003 创建人
	TypeValueModifiedUser                      // 1004 修改人
	TypeValueAutoNumber                        // 1005 自动编号
)

type SimpleField struct {
	FieldName         string `json:"field_name"`
	UiType            string `json:"ui_type"`
	Type              int    `json:"type"`
	DateFormatter     string `json:"date_formatter"`
	Formatter         string `json:"formatter"`
	FormulaExpression string `json:"formula_expression"`
	IsPrimary         bool   `json:"is_primary"`
	AutoIncrement     bool   `json:"auto_increment"`
}

func ConvertInterface2TypeValue(data any, typeValue int) (any, error) {
	v, ok := data.(map[string]any)
	if ok {
		if v["data"] != nil {
			vv, ok := v["data"].(map[string]any)
			if ok {
				v = vv
			}
		}
		tv, ok := v["type"].(float64)
		if ok {
			return ConvertInterface2TypeValue(v["value"], int(tv))
		}
	}

	switch typeValue {
	case 1, 3, 13, 1005:
		// 将interface{}转换为string
		if data == nil {
			return "", nil
		}
		return ConvertData2String(data), nil
	case 2:
		// 将interface{}转换为float64
		return ConvertData2Number(data)
	case 15:
		return ConvertData2Link(data)
	case 4, 18:
		// 将interface{}转换为[]string
		return convertData2Array(data)
	case 7:
		// 将interface{}转换为bool
		return ConvertData2Bool(data)
	case 5, 1001, 1002:
		return covnertDate2UnixTime(data)
	default:
		return nil, fmt.Errorf("unsupported type: %d", typeValue)
	}
}

func ConvertData2Link(data any) (map[string]string, error) {
	if data == nil {
		return nil, nil
	}
	// {
	// 	"link": "https://www.feishu.cn",
	// 	"text": "飞书"
	// }
	switch v := data.(type) {
	case map[string]any:
		if link, ok := v["link"].(string); ok {
			if text, ok := v["text"].(string); ok {
				return map[string]string{
					"link": link,
					"text": text,
				}, nil
			}
			return map[string]string{
				"link": link,
				"text": link,
			}, nil
		}
	case string:
		return map[string]string{
			"link": v,
			"text": v,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported link type: %T", data)
	}

	return nil, nil
}

// convert data 2 bool
func ConvertData2Bool(data interface{}) (bool, error) {
	switch v := data.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	case int64:
		return v != 0, nil
	case int:
		return v != 0, nil
	default:
		return false, fmt.Errorf("unsupported bool type: %T", data)
	}
}

func ConvertData2Number(data any) (float64, error) {
	if data == nil {
		return 0.0, nil
	}

	switch v := data.(type) {
	case int64:
		return float64(v), nil
	case string:
		if v == "" {
			return 0, nil
		}
		return strconv.ParseFloat(v, 64)
	case []uint8:
		return strconv.ParseFloat(string(v), 64)
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case []any:
		if len(v) == 0 {
			return 0, nil
		}
		return ConvertData2Number(v[0])
	case map[string]any:
		vsi, ok := v["value"]
		if !ok {
			vsi, ok = v["data"]
		}
		if ok {
			vs, ok := vsi.([]any)
			if ok {
				return ConvertData2Number(vs[0])
			}
			return ConvertData2Number(vsi)
		}
		return 0, fmt.Errorf("unsupported number type: %T", data)
	default:
		return 0, fmt.Errorf("unsupported number type: %T", data)
	}
}

// convert data 2 int
func ConvertData2Int(data interface{}) (int, error) {
	if data == nil {
		return 0, nil
	}
	r, err := ConvertData2Number(data)
	if err != nil {
		return 0, err
	}
	return int(r), nil
}

// convert data 2 int64
func ConvertData2Int64(data interface{}) (int64, error) {
	if data == nil {
		return 0, nil
	}
	r, err := ConvertData2Number(data)
	if err != nil {
		return 0, err
	}
	return int64(r), nil
}

// convert data 2 records array
func ConvertData2Records(data any) []string {
	d, ok := data.(map[string]any)
	var rcs []string
	if ok {
		if v, ok := d["link_record_ids"]; ok {
			dd, ok := v.([]any)

			if ok {
				for _, d := range dd {
					if d == nil {
						continue
					}
					rcs = append(rcs, fmt.Sprintf("%v", d))
				}
			}

		}
	}

	return rcs

}

type File struct {
	// file token
	FileToken string `json:"file_token"`
	// name
	Name string `json:"name"`
	// size
	Size int64 `json:"size"`
	// type
	Type string `json:"type"`
	// url
	URL string `json:"url"`
	// tmp_url
	TmpURL string `json:"tmp_url"`
}

// convert data 2 file
func ConvertData2File(data any) ([]*File, error) {
	if data == nil {
		return nil, nil
	}
	d, ok := data.([]any)
	if !ok {
		m, ok := data.(map[string]any)
		if ok {
			t, ok := m["type"].(float64)
			if !ok || t != 17 {
				return nil, fmt.Errorf("unsupported file type: %T", data)
			}
			return ConvertData2File(m["value"])
		}
		return nil, fmt.Errorf("unsupported file type: %T", data)
	}
	files := make([]*File, 0, len(d))
	for _, v := range d {

		if v == nil {
			continue
		}
		m, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("unsupported file type: %T", v)
		}
		file := &File{}
		if fileToken, ok := m["file_token"].(string); ok {
			file.FileToken = fileToken
		}
		if name, ok := m["name"].(string); ok {
			file.Name = name
		}
		if size, ok := m["size"].(float64); ok {
			file.Size = int64(size)
		}
		if typeValue, ok := m["type"].(string); ok {

			file.Type = typeValue
		}
		if url, ok := m["url"].(string); ok {
			file.URL = url
		}
		if tmpURL, ok := m["tmp_url"].(string); ok {
			file.TmpURL = tmpURL
		}
		files = append(files, file)
	}
	return files, nil
}

// convert data 2 string
func ConvertData2String(data any) string {
	dd, ok := data.([]any)

	if ok {
		if len(dd) == 0 {
			return ""
		}
		ss := make([]string, 0, len(dd))
		for _, d := range dd {
			if d == nil {
				continue
			}
			dm, ok := d.(map[string]any)
			if !ok {
				ss = append(ss, fmt.Sprintf("%v", d))
				continue
			}
			if v, ok := dm["text"]; ok {
				ss = append(ss, fmt.Sprintf("%v", v))
			} else {
				ss = append(ss, fmt.Sprintf("%v", d))
			}
		}
		return strings.Join(ss, "")
	}

	dm, ok := data.(map[string]any)
	if ok {
		if v, ok := dm["text"]; ok {
			return fmt.Sprintf("%v", v)
		}
		if v, ok := dm["value"]; ok {
			return ConvertData2String(v)
		}
	}

	return fmt.Sprintf("%v", data)
}

func covnertDate2UnixTime(data any) (int64, error) {
	switch v := data.(type) {
	case time.Time:
		return v.Unix() * 1000, nil
	case *time.Time:
		return v.Unix() * 1000, nil
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case string:
		if strings.Contains(v, "-") {
			t, err := time.Parse("2006-01-02 15:04:05", v)
			if err != nil {
				return 0, err
			}
			return t.Unix() * 1000, nil
		}
		if v == "" {
			return 0, nil
		}
		return strconv.ParseInt(v, 10, 64)
	case []uint8:
		return strconv.ParseInt(string(v), 10, 64)

	case []any:
		if len(v) == 0 {
			return 0, nil
		}
		return covnertDate2UnixTime(v[0])
	default:
		return 0, fmt.Errorf("unsupported date type: %T", v)
	}
}

func convertData2Array(i any) ([]string, error) {
	if i == nil {
		return []string{}, nil
	}
	switch v := i.(type) {
	case []string:
		return v, nil
	case string:
		return convertString2Array(v), nil
	case []uint8:
		return convertString2Array(string(v)), nil
	case []any:
		arr := make([]string, 0, len(v))
		for _, item := range v {
			if item == nil {
				continue
			}
			arr = append(arr, ConvertData2String(item))
		}
		return arr, nil
	default:
		return nil, fmt.Errorf("unsupported string type: %T", i)
	}

}

func convertString2Array(s string) []string {
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	return strings.Split(s, ",")
}

func (f *SimpleField) Build() *larkbitable.AppTableField {
	b := larkbitable.NewAppTableFieldBuilder()
	b = b.FieldName(f.FieldName).Type(f.Type)
	if f.UiType != "" {
		b = b.UiType(f.UiType)
	}

	if f.DateFormatter != "" || f.Formatter != "" || f.FormulaExpression != "" {
		pb := larkbitable.NewAppTableFieldPropertyBuilder()
		if f.DateFormatter != "" {
			pb = pb.DateFormatter(f.DateFormatter)
		}
		if f.Formatter != "" {
			pb = pb.Formatter(f.Formatter)
		}
		if f.FormulaExpression != "" {
			pb = pb.FormulaExpression(f.FormulaExpression)
		}
		if f.AutoIncrement {
			afpab := larkbitable.NewAppFieldPropertyAutoSerialBuilder()
			opb := larkbitable.NewAppFieldPropertyAutoSerialOptionsBuilder()
			opb.Type("auto_increment_number")
			afpab.Options([]*larkbitable.AppFieldPropertyAutoSerialOptions{opb.Build()})
			pb = pb.AutoSerial(afpab.Build())
		}
		b = b.Property(pb.Build())
	}
	b = b.IsPrimary(f.IsPrimary)
	return b.Build()
}

// AppTableField
func ConvertFeishuAppTableField2SimpleField(t *larkbitable.AppTableField) (*SimpleField, error) {
	if t == nil {
		return nil, nil
	}
	if t.Type == nil {
		return nil, fmt.Errorf("type is nil")
	}
	sf := &SimpleField{
		FieldName: printfAddressString(t.FieldName),
		UiType:    printfAddressString(t.UiType),
		Type:      *t.Type,
		IsPrimary: *t.IsPrimary,
	}
	if t.Property == nil {
		sf.DateFormatter = ""
		sf.Formatter = ""
		sf.FormulaExpression = ""
	}
	if t.Property != nil {
		sf.DateFormatter = printfAddressString(t.Property.DateFormatter)
		sf.Formatter = printfAddressString(t.Property.Formatter)
		sf.FormulaExpression = printfAddressString(t.Property.FormulaExpression)
		if t.Property.AutoSerial != nil {
			sf.AutoIncrement = *t.Property.AutoSerial.Type == "auto_increment_number"
		}
	}
	return sf, nil
}

// ConvertFeishuAppTableListField2SimpleField
func ConvertFeishuAppTableListField2SimpleField(t *larkbitable.AppTableFieldForList) (*SimpleField, error) {
	if t == nil {
		return nil, nil
	}
	if t.Type == nil {
		return nil, fmt.Errorf("type is nil")
	}
	autoIncrement := false
	if t.Property != nil && t.Property.AutoSerial != nil && t.Property.AutoSerial.Type != nil {
		autoIncrement = *t.Property.AutoSerial.Type == "auto_increment_number"
	}
	sf := &SimpleField{
		FieldName:     printfAddressString(t.FieldName),
		UiType:        printfAddressString(t.UiType),
		Type:          *t.Type,
		IsPrimary:     *t.IsPrimary,
		AutoIncrement: autoIncrement,
	}
	if t.Property == nil {
		sf.DateFormatter = ""
		sf.Formatter = ""
		sf.FormulaExpression = ""
	}
	if t.Property != nil {
		sf.DateFormatter = printfAddressString(t.Property.DateFormatter)
		sf.Formatter = printfAddressString(t.Property.Formatter)
		sf.FormulaExpression = printfAddressString(t.Property.FormulaExpression)
		if t.Property.AutoSerial != nil {
			sf.AutoIncrement = *t.Property.AutoSerial.Type == "auto_increment_number"
		}
	}
	return sf, nil
}

// 格式化引用字符串
func printfAddressString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// 1：多行文本（默认值）、条码  string
// 2：数字（默认值）、进度、货币、评分 number
// 3：单选 string
// 4：多选 array
// 5：日期 number unix时间戳+'000'
// 7：复选框 boolean
// 11：人员 -
// 13：电话号码 string
// 15：超链接 -
// 17：附件 -
// 18：单向关联 array
// 19：查找引用 -
// 20：公式 -
// 21：双向关联 -
// 22：地理位置 -
// 23：群组 -
// 1001：创建时间 number
// 1002：最后更新时间 number
// 1003：创建人 -
// 1004：修改人 -
// 1005：自动编号 string

// "Text"：多行文本
// "Barcode"：条码
// "Number"：数字
// "Progress"：进度
// "Currency"：货币
// "Rating"：评分
// "SingleSelect"：单选
// "MultiSelect"：多选
// "DateTime"：日期
// "Checkbox"：复选框
// "User"：人员
// "GroupChat"：群组
// "Phone"：电话号码
// "Url"：超链接
// "Attachment"：附件
// "SingleLink"：单向关联
// "Formula"：公式
// "Lookup": 查找引用
// "DuplexLink"：双向关联
// "Location"：地理位置
// "CreatedTime"：创建时间
// "ModifiedTime"：最后更新时间
// "CreatedUser"：创建人
// "ModifiedUser"：修改人
// "AutoNumber"：自动编号

func ConvertFeishuData2Object(data map[string]any, obj any) error {
	// 获取对象的 feishu tag
	v := reflect.ValueOf(obj)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		tag := field.Tag.Get("feishu")
		if tag == "" {
			continue
		}

		if v.Field(i).CanSet() {
			if v.Field(i).Kind() == reflect.String {
				if v.Field(i).CanSet() {
					if v.Field(i).CanSet() {
						v.Field(i).SetString(ConvertData2String(data[tag]))
					}
				}
			} else if v.Field(i).Kind() == reflect.Int {
				if v.Field(i).CanSet() {
					iv, err := ConvertData2Int(data[tag])
					if err != nil {
						return err
					}
					v.Field(i).SetInt(int64(iv))
				}
			} else if v.Field(i).Kind() == reflect.Int64 {
				if v.Field(i).CanSet() {

					iv, err := ConvertData2Int64(data[tag])
					if err != nil {
						return err
					}
					v.Field(i).SetInt(iv)
				}
			} else if v.Field(i).Kind() == reflect.Float64 {
				if v.Field(i).CanSet() {
					iv, err := ConvertData2Number(data[tag])
					if err != nil {
						return err
					}
					v.Field(i).SetFloat(iv)
				}
			} else if v.Field(i).Kind() == reflect.Bool {
				if v.Field(i).CanSet() {
					iv, err := ConvertData2Bool(data[tag])
					if err != nil {
						return err
					}
					v.Field(i).SetBool(iv)
				}
			} else if v.Field(i).Kind() == reflect.Slice {
				if v.Field(i).CanSet() {
					v.Field(i).Set(reflect.ValueOf(data[tag]))
				}
			}
		}
	}
	return nil

}
