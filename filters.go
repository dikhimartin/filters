package fliters

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"github.com/iancoleman/strcase"
	"gorm.io/gorm"
)

// PageModel struct
type PageModel struct {
	Items                interface{} `json:"items"` // (Optional)
	Page                 int         `json:"page"`
	PrevPage             int         `json:"prev_page"`
	NextPage             int         `json:"next_page"`
	PostsPerPage         int         `json:"size"`
	CurrentParam         string      `json:"current_param"`
	First                bool        `json:"first"` // (Optional)
	Last                 bool        `json:"last"`
	HasPages             bool        `json:"hasPages"`
	Paginates            []int       `json:"paginates"`
	TotalPages           float64     `json:"totalPages,omitempty"` // (Optional) Total data shows
	TotalVisible         int         `json:"total,omitempty"`      // Total real data
	TotalRecordsFiltered int         `json:"visible,omitempty"`    // (Optional) Total records filter
	Additional           interface{} `json:"additional,omitempty"` // (Optional)
	Summary              interface{} `json:"summary,omitempty"`    // (Optional)
}

// BeetweenString func
func BeetweenString(value string, a string, b string) string {
	posFirst := strings.Index(value, a)
	if posFirst == -1 {
		return ""
	}
	posLast := strings.Index(value, b)
	if posLast == -1 {
		return ""
	}
	posFirstAdjusted := posFirst + len(a)
	if posFirstAdjusted >= posLast {
		return ""
	}
	return value[posFirstAdjusted:posLast]
}

// NormalizeParam func
func NormalizeParam(Param string) string {
	v := BeetweenString(Param, "page=", "&")

	value := strings.Replace(Param, "page="+v+"", "", -1)
	valInt, _ := strconv.Atoi(value)
	if valInt != 0 {
		value = ""
	}
	if v == "" {
		value = "&" + value
	}
	return value
}

// GeneratePagination func
func GeneratePagination(pageNumber, pageSize, TotalVisible int, CurrentParam []byte, Items interface{}) *PageModel {
	dataModel := &PageModel{}

	totalPages := math.Ceil(float64(TotalVisible) / float64(pageSize))
	if totalPages == math.Inf(0) || math.IsNaN(totalPages) {
		totalPages = 0
	}

	DecodedParam, _ := url.QueryUnescape(string(CurrentParam))
	Parameter := NormalizeParam(DecodedParam)

	dataModel.TotalPages = totalPages
	dataModel.TotalVisible = TotalVisible
	dataModel.TotalRecordsFiltered = reflect.ValueOf(Items).Len()
	dataModel.Items = Items
	dataModel.Page = pageNumber
	dataModel.PrevPage = pageNumber - 1
	dataModel.NextPage = pageNumber + 1
	dataModel.PostsPerPage = pageSize
	dataModel.CurrentParam = Parameter
	dataModel.Paginates = Paginates(int(totalPages))
	dataModel.HasPages = HasPages(int(totalPages))
	dataModel.First = pageNumber == 1
	dataModel.Last = (pageNumber * pageSize) >= dataModel.TotalVisible

	return dataModel
}

// Paginates func
func Paginates(totalPages int) []int {
	var paginates []int
	for i := 0; i < totalPages; i++ {
		numdata := i + 1
		paginates = append(paginates, numdata)
	}
	return paginates
}

// HasPages func
func HasPages(totalPages int) bool {
	var hasPages bool
	if totalPages > 1 {
		hasPages = true
	} else {
		hasPages = false
	}
	return hasPages
}

// FilterItem struct
type FilterItem struct {
	Field     string
	Operator  string
	Value     interface{}
	ValueType string
}

// QueryFilter struct
type QueryFilter struct {
	Item FilterItem
	Type string
}

// NormalizeFieldName func
func NormalizeFieldName(field string) string {
	slices := strings.Split(field, ",")
	if len(slices) == 1 {
		return field
	}
	newSlices := []string{}
	if len(slices) > 0 {
		newSlices = append(newSlices, strcase.ToCamel(slices[0]))
		for k, s := range slices {
			if k > 0 {
				newSlices = append(newSlices, s)
			}
		}
	}
	if len(newSlices) == 0 {
		return field
	}
	return strings.Join(newSlices, "__")
}

// SetFilterValue func
func SetFilterValue(item *FilterItem, a interface{}) {
	stringValue, isString := a.(string)
	boolValue, isBool := a.(bool)
	intValue, isNumber := a.(int64)
	int8Value, isNumber8 := a.(int8)
	floatValue, isFloat := a.(float64)
	arrayValue, isArray := a.([]interface{})
	if isString {
		item.Value = stringValue
		item.ValueType = "string"
	} else if isBool {
		item.Value = boolValue
		item.ValueType = "bool"
	} else if isNumber8 {
		item.Value = int8Value
		item.ValueType = "int8"
	} else if isNumber {
		item.Value = intValue
		item.ValueType = "int64"
	} else if isFloat {
		item.Value = floatValue
		item.ValueType = "float64"
	} else if isArray {
		item.Value = arrayValue
		item.ValueType = "array"
	}
}

// CreateFilter func
func CreateFilter(jsonParams string) []QueryFilter {
	var output interface{}
	err := json.Unmarshal([]byte(jsonParams), &output)
	if nil != err {
		fmt.Println(err)
		return []QueryFilter{}
	}

	filters := []QueryFilter{}

	iface, ok := output.([]interface{})
	if ok {
		var hasSingle = false
		singleFilter := QueryFilter{
			Item: FilterItem{},
			Type: "single",
		}
		for x, v := range iface {
			item, ok2 := v.([]interface{})
			if ok2 && !hasSingle {
				filter := QueryFilter{
					Type: "multiple",
					Item: FilterItem{},
				}
				for i, a := range item {
					if len(item) == 1 {
						filter.Item.Operator = a.(string)
						filter.Type = "operator"
						continue
					}
					if i == 0 {
						filter.Item.Field = a.(string)
					} else if i == 1 {
						if len(item) == 2 {
							filter.Item.Operator = "="
							SetFilterValue(&filter.Item, a)
						} else {
							filter.Item.Operator = a.(string)
						}
					} else if i == 2 {
						SetFilterValue(&filter.Item, a)
					}
				}
				filter.Item.Field = "\"" + NormalizeFieldName(filter.Item.Field) + "\""
				filter.Item.Operator = strings.ToUpper(filter.Item.Operator)
				filters = append(filters, filter)
			} else {
				hasSingle = true
				if x == 0 {
					fieldName, valid := v.(string)
					if valid {
						singleFilter.Item.Field = "\"" + NormalizeFieldName(fieldName) + "\""
					}
				} else if x == 1 {
					if len(iface) == 2 {
						singleFilter.Item.Operator = "="
						SetFilterValue(&singleFilter.Item, v)
					} else {
						opName, valid := v.(string)
						if valid {
							singleFilter.Item.Operator = strings.ToUpper(opName)
						}
					}
				} else if x == 2 {
					SetFilterValue(&singleFilter.Item, v)
				}
			}
		}
		if hasSingle {
			filters = append(filters, singleFilter)
		}
	}

	return filters
}

// CreateWhereCause func
func CreateWhereCause(filter QueryFilter, queryFilters *[]string, whereParams *[]interface{}) {
	if (filter.Item.Operator == "IS" || filter.Item.Operator == "IS NOT") && filter.Item.Value == nil {
		*queryFilters = append(*queryFilters, fmt.Sprintf("(%s %s NULL)",
			filter.Item.Field,
			filter.Item.Operator,
		))
		return
	}

	if filter.Item.Operator == "LIKE" || filter.Item.Operator == "NOT LIKE" {
		cause := fmt.Sprintf("%s %s ?",
			filter.Item.Field,
			filter.Item.Operator,
		)
		*queryFilters = append(*queryFilters, cause)
		value := ""
		switch filter.Item.ValueType {
		case "string":
			value = "%" + (filter.Item.Value.(string)) + "%"
		case "int64":
			value = "%" + fmt.Sprintf("%v", filter.Item.Value.(int64)) + "%"
		case "int8":
			value = "%" + fmt.Sprintf("%v", filter.Item.Value.(int8)) + "%"
		case "float64":
			value = "%" + fmt.Sprintf("%v", filter.Item.Value.(float64)) + "%"
		}

		if value != "" {
			value = strings.ReplaceAll(value, " ", "%")
			*whereParams = append(*whereParams, value)
		}

	} else if filter.Item.Operator == "IN" || filter.Item.Operator == "NOT IN" {
		cause := fmt.Sprintf("%s %s ?",
			filter.Item.Field,
			filter.Item.Operator,
		)
		*queryFilters = append(*queryFilters, cause)
		if filter.Item.ValueType == "array" {
			value, ok := filter.Item.Value.([]interface{})
			if ok {
				values := []interface{}{}
				for _, val := range value {
					v, o := val.(string)
					if o {
						values = append(values, strings.ToLower(v))
					} else {
						values = append(values, fmt.Sprintf("%v", v))
					}
				}
				*whereParams = append(*whereParams, values)
			}
		}
	} else if filter.Item.Operator == "BETWEEN" {
		cause := fmt.Sprintf("%s %s ? AND ?",
			filter.Item.Field,
			filter.Item.Operator,
		)
		*queryFilters = append(*queryFilters, cause)
		if filter.Item.ValueType == "array" {
			value, ok := filter.Item.Value.([]interface{})
			if ok {
				values := []interface{}{}
				for _, val := range value {
					v, o := val.(string)
					if o {
						values = append(values, strings.ToLower(v))
					} else {
						values = append(values, fmt.Sprintf("%v", v))
					}
				}

				if len(values) == 2 {
					*whereParams = append(*whereParams, values[0], values[1])
				}
			}
		}
	} else {
		cause := fmt.Sprintf("%s %s ?",
			filter.Item.Field,
			filter.Item.Operator,
		)
		*queryFilters = append(*queryFilters, cause)
		*whereParams = append(*whereParams, fmt.Sprintf("%v", filter.Item.Value))
	}
	return
}

// CreateCustomFilters func
// => Example
// => filters=["age", "not in", [20,21] ]
// -> pageFilters      := [["id","=","6"],["AND"],["status_transaction","=","waiting"],["AND"],["business_id","=","10"]]
// -> pageSearch       := "value"
// -> columnFilters    := ["trx_id","id"]
// -> pageSort         :=
// - Sort ascending example  : sort=column_name
// - Sort descending example : sort=-column_name
// - Sort multiple example   : sort=-fist_column,second_column
func CreateCustomFilters(pageFilters, pageSearch, columnFilter, pageSort string) (string, []interface{}, string, []interface{}, string) {
	queryFilters := []string{}
	whereFilters := []interface{}{}
	ResultFilters := ""

	querySearch := []string{}
	whereSearch := []interface{}{}
	ResultSearch := ""

	// search_by _column
	columnFilters := StringToJson(columnFilter)
	if nil != columnFilters {
		if CountLengthIface(columnFilters) > 0 {
			if "" != pageSearch {
				switch reflect.TypeOf(columnFilters).Kind() {
				case reflect.Slice:
					s := reflect.ValueOf(columnFilters)
					for i := 0; i < s.Len(); i++ {
						value := s.Index(i).Interface().(string)
						querySearch = append(querySearch, ""+value+" LIKE ?")
						whereSearch = append(whereSearch, "%"+pageSearch+"%")
					}
				}
				if len(querySearch) > 0 {
					ResultSearch = "" + strings.Join(querySearch, " OR ") + ""
				}
			}
		}
	}

	// filters_by_field
	if "" != pageFilters {
		filters := CreateFilter(pageFilters)
		if len(filters) == 1 && filters[0].Type == "single" {
			CreateWhereCause(filters[0], &queryFilters, &whereFilters)
		} else if len(filters) >= 1 {
			for b, filter := range filters {
				if filter.Type == "operator" {
					queryFilters = append(queryFilters, filter.Item.Operator)
					continue
				} else if filter.Type == "multiple" {
					if b > 0 && filters[b-1].Type != "operator" {
						queryFilters = append(queryFilters, " OR ")
					}
					CreateWhereCause(filter, &queryFilters, &whereFilters)
				}
			}
		}

		if len(queryFilters) > 0 {
			ResultFilters = "" + strings.Join(queryFilters, " ") + ""
		}
	}

	var orderBy = "id desc"
	if "" != pageSort {
		columnName := pageSort
		direction := "asc"
		if string(pageSort[0]) == "-" {
			columnName = string(pageSort[1:])
			direction = "desc"
		}
		columnName = NormalizeFieldName(columnName)
		orderBy = columnName + " " + direction
	}

	ResultFilters = strings.ReplaceAll(ResultFilters, "\"", "")

	return ResultFilters, whereFilters, ResultSearch, whereSearch, orderBy
}

// CountRecordsData func
func CountRecordsData(CustomFilters, pageFilters, pageSearch, columnFilter string, model *gorm.DB) int {
	queryFilter, whereFilters, querySearch, whereSearch, _ := CreateCustomFilters(pageFilters, pageSearch, columnFilter, "")
	customFilter, whereCustom, _, _, _ := CreateCustomFilters(CustomFilters, "", "", "")

	var count int64
	model.
		Where(customFilter, whereCustom...).
		Where(queryFilter, whereFilters...).
		Where(querySearch, whereSearch...).
		Count(&count)

	return int(count)
}

// StringToJson func
func StringToJson(value string) interface{} {
	var output interface{}
	err := json.Unmarshal([]byte(value), &output)
	if nil != err {
		fmt.Println(err)
	}
	return output
}

// CountLengthIface func
func CountLengthIface(data interface{}) int {
	var value int
	switch reflect.TypeOf(data).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(data)
		value = s.Len()
	}
	return value
}
