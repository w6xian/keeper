package console

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// Group 创建一个内联分组，后续输出会缩进
func Group(label string) {
	std.Group(label)
}

func (c *Console) Group(label string) {
	if c.disabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.groupLevel++
	indent := strings.Repeat("  ", c.groupLevel-1)
	c.writef(c.output, colorPurple, "%s▼ %s", indent, label)
}

// GroupCollapsed 创建一个折叠的分组（在终端中与 Group 行为相同）
func GroupCollapsed(label string) {
	std.GroupCollapsed(label)
}

func (c *Console) GroupCollapsed(label string) {
	if c.disabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.groupLevel++
	indent := strings.Repeat("  ", c.groupLevel-1)
	c.writef(c.output, colorPurple, "%s▶ %s (折叠)", indent, label)
}

// GroupEnd 结束当前分组
func GroupEnd() {
	std.GroupEnd()
}

func (c *Console) GroupEnd() {
	if c.disabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.groupLevel > 0 {
		c.groupLevel--
	}
}

// Dir 显示对象的交互式列表
func Dir(obj interface{}) {
	std.Dir(obj)
}

func (c *Console) Dir(obj interface{}) {
	if c.disabled {
		return
	}

	c.writef(c.output, colorCyan, "对象详细信息:")

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i)
			c.writef(c.output, colorGray, "  %s: %v", field.Name, value.Interface())
		}
	} else if v.Kind() == reflect.Map {
		for _, key := range v.MapKeys() {
			value := v.MapIndex(key)
			c.writef(c.output, colorGray, "  %v: %v", key.Interface(), value.Interface())
		}
	} else {
		data, _ := json.MarshalIndent(obj, "", "  ")
		c.write(c.output, colorGray, string(data))
	}
}

// DirXML 显示对象的 XML/HTML 表示（Go 中简化为 JSON 格式）
func DirXML(obj interface{}) {
	std.DirXML(obj)
}

func (c *Console) DirXML(obj interface{}) {
	if c.disabled {
		return
	}

	c.writef(c.output, colorCyan, "对象 XML/HTML 表示:")

	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		c.Error("无法序列化对象:", err)
		return
	}

	c.write(c.output, colorGray, string(data))
}

// Table 以表格形式显示数组或对象
func Table(data interface{}) {
	std.Table(data)
}

func (c *Console) Table(data interface{}) {
	if c.disabled {
		return
	}

	v := reflect.ValueOf(data)

	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		if v.Len() == 0 {
			c.Info("空数组")
			return
		}

		// 判断是否是结构体数组
		if v.Index(0).Kind() == reflect.Struct {
			t := v.Index(0).Type()
			fields := []string{}
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				fields = append(fields, field.Name)
			}

			// 计算每列最大宽度
			colWidths := make([]int, len(fields))
			for i, field := range fields {
				colWidths[i] = len(field)
			}

			rows := make([][]string, v.Len())
			for i := 0; i < v.Len(); i++ {
				row := []string{}
				for j := 0; j < t.NumField(); j++ {
					val := fmt.Sprintf("%v", v.Index(i).Field(j).Interface())
					row = append(row, val)
					if len(val) > colWidths[j] {
						colWidths[j] = len(val)
					}
				}
				rows[i] = row
			}

			// 输出表头
			header := ""
			for j, field := range fields {
				header += fmt.Sprintf("%-*s  ", colWidths[j]+2, field)
			}
			c.writeWithColor(c.output, colorCyan, header)

			// 输出分隔线
			separator := ""
			for _, w := range colWidths {
				separator += fmt.Sprintf("%-*s  ", w+2, strings.Repeat("-", w))
			}
			c.write(c.output, colorGray, separator)

			// 输出数据行
			for _, row := range rows {
				line := ""
				for j, val := range row {
					line += fmt.Sprintf("%-*s  ", colWidths[j]+2, val)
				}
				c.write(c.output, colorReset, line)
			}
		} else {
			// 简单数组
			c.writef(c.output, colorCyan, "索引    值")
			c.write(c.output, colorGray, "────────  ──────────")
			for i := 0; i < v.Len(); i++ {
				c.writef(c.output, colorReset, "%-8d  %v", i, v.Index(i).Interface())
			}
		}
	} else if v.Kind() == reflect.Map {
		// Map 类型
		c.writef(c.output, colorCyan, "键         值")
		c.write(c.output, colorGray, "─────────  ──────────")
		for _, key := range v.MapKeys() {
			value := v.MapIndex(key)
			c.writef(c.output, colorReset, "%-9v  %v", key.Interface(), value.Interface())
		}
	} else {
		c.writef(c.output, colorCyan, "属性        值")
		c.write(c.output, colorGray, "───────────  ──────────")
		data, _ := json.MarshalIndent(data, "", "  ")
		c.write(c.output, colorReset, string(data))
	}
}
