package util

import (
	"encoding/json"
	"os"
	"sync"
)

type Param struct {
	length int64
	start  int64
	draw   int64
}

type MapLock struct {
	Data map[string]interface{}
	Lock sync.RWMutex
}

func (m *MapLock) Set(k string, v interface{}) {
	m.Lock.Lock()
	if len(m.Data) < 1 {
		m.Data = make(map[string]interface{})
	}
	m.Data[k] = v
	defer m.Lock.Unlock()
}

func returnMap(data interface{}) string {
	v, err := json.Marshal(data)
	if err == nil {
		return string(v)
	}
	return "{}"
}

// 为表格提供的数据
// return
func ResponseMap(listResult interface{}, total interface{}, draw interface{}) map[string]interface{} {
	maps := MapLock{}
	maps.Set("data", listResult)
	maps.Set("recordsTotal", total)
	maps.Set("recordsFiltered", total)
	maps.Set("draw", draw)
	return maps.Data
}

// 2019-01-15
//  获取表的行数
func GetTableRows(table string) {

}

// 响应错误信息
func ResponseMapError(err string) map[string]interface{} {
	maps := MapLock{}
	maps.Set("data", err)
	maps.Set("recordsTotal", 0)
	maps.Set("recordsFiltered", 0)
	maps.Set("draw", 1)
	return maps.Data
}

// API响应信息
func ApiResponse(status bool, info interface{}) map[string]interface{} {
	maps := MapLock{}
	maps.Set("data", info)
	maps.Set("status", status)
	maps.Set("date", GetDate())
	hostname, _ := os.Hostname()
	maps.Set("server", hostname)
	maps.Set("code", 0)
	if !status {
		maps.Set("code", -1)
	}
	return maps.Data
}

// API为表格提供的数据
// return
func NewResponseMap(listResult interface{}, total int, pageNo int64, pageSize int64) map[string]interface{} {
	maps := MapLock{}
	maps.Set("status", 200)
	maps.Set("timestamp", TimeToStamp(string(GetDate())))
	maps.Set("result", map[string]interface{}{
		"data":       listResult,
		"pageSize":   pageSize,
		"pageNo":     pageNo,
		"totalCount": total,
	})
	return maps.Data
}

// API响应信息
func RestApiResponse(code int, info interface{}) map[string]interface{} {
	maps := MapLock{}
	maps.Set("status", 200)
	maps.Set("timestamp", TimeToStamp(string(GetDate())))
	hostname, _ := os.Hostname()
	maps.Set("server", hostname)
	if code != 200 {
		maps.Set("status", 500)
		maps.Set("message", info)
	} else {
		maps.Set("result", info)
	}

	maps.Set("code", code)
	return maps.Data
}

// API响应信息
func NewApiResponse(status bool, info interface{}) map[string]interface{} {
	maps := MapLock{}
	maps.Set("result", info)
	maps.Set("status", status)
	maps.Set("date", TimeToStamp(string(GetDate())))
	hostname, _ := os.Hostname()
	maps.Set("server", hostname)
	maps.Set("code", 0)
	if !status {
		maps.Set("code", -1)
	}
	return maps.Data
}
