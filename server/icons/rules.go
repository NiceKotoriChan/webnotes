package icons

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MaxRulesBytes 规则文件的体积上限：几 KiB 的东西，一万倍余量足够，防止一次请求把文件写坏。
const MaxRulesBytes = 1 << 20

// Rule 一条匹配规则：match 是大小写不敏感的正则，icon 是不带 mdi: 前缀的图标名。
type Rule struct {
	Match string `json:"match"`
	Icon  string `json:"icon"`
}

// RulesDoc 规则文件的内容，默认规则与自定义规则同构。
type RulesDoc struct {
	Fallback string `json:"fallback"`
	Rules    []Rule `json:"rules"`
}

// Rules 管理图标规则文件：默认规则随目录走、只读，自定义规则可被前端反复改写。
type Rules struct {
	dir        string
	defName    string
	customName string
}

func NewRules(dir, defName, customName string) *Rules {
	return &Rules{dir: dir, defName: defName, customName: customName}
}

// Ensure 保证自定义规则存在：缺失时用默认规则灌一份。
// 这样「自定义规则默认就等于默认规则」，前端拿到的永远是一份可以直接改的完整规则。
func (r *Rules) Ensure() error {
	custom, def := r.customPath(), r.defaultPath()
	if _, err := os.Stat(custom); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	data, err := os.ReadFile(def)
	if err != nil {
		return fmt.Errorf("读取默认规则 %s: %w", r.defName, err)
	}
	if _, err := ParseRules(data); err != nil {
		return fmt.Errorf("默认规则 %s 本身不合法: %w", r.defName, err)
	}
	return r.write(custom, data)
}

// Custom 当前的自定义规则。
func (r *Rules) Custom() (RulesDoc, error) {
	data, err := os.ReadFile(r.customPath())
	if err != nil {
		return RulesDoc{}, err
	}
	doc, err := ParseRules(data)
	if err != nil {
		return RulesDoc{}, fmt.Errorf("%s 不合法: %w", r.customName, err)
	}
	return doc, nil
}

// SaveCustom 覆盖自定义规则：先校验、再原子替换，写坏了不认账。
func (r *Rules) SaveCustom(body []byte) (RulesDoc, error) {
	doc, err := ParseRules(body)
	if err != nil {
		return RulesDoc{}, err
	}
	// 落盘用规范化后的 JSON，不把前端那边的缩进/字段顺序原样带进文件
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return RulesDoc{}, err
	}
	if err := r.write(r.customPath(), append(out, '\n')); err != nil {
		return RulesDoc{}, err
	}
	return doc, nil
}

// ResetCustom 回退到默认设置：用默认规则覆盖自定义规则。
func (r *Rules) ResetCustom() (RulesDoc, error) {
	data, err := os.ReadFile(r.defaultPath())
	if err != nil {
		return RulesDoc{}, err
	}
	return r.SaveCustom(data)
}

func (r *Rules) defaultPath() string { return filepath.Join(r.dir, r.defName) }
func (r *Rules) customPath() string  { return filepath.Join(r.dir, r.customName) }

// write 原子落盘：先写同目录的临时文件再 rename。
func (r *Rules) write(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// ParseRules 校验规则文件：必须是一个对象，带 rules 数组，每条都有非空的 match 与 icon。
// 正则本身**不在这里编译** —— 规则是给前端用 JS 正则执行的，Go 的方言更窄，
// 拿标准库去卡会把合法的前端规则误判成非法。这里只保证结构对得上。
func ParseRules(data []byte) (RulesDoc, error) {
	if len(data) > MaxRulesBytes {
		return RulesDoc{}, fmt.Errorf("规则超过 %d 字节", MaxRulesBytes)
	}
	var doc RulesDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return RulesDoc{}, fmt.Errorf("不是合法 JSON: %w", err)
	}
	if doc.Rules == nil {
		return RulesDoc{}, errors.New("缺少 rules 数组")
	}
	for i, rule := range doc.Rules {
		if strings.TrimSpace(rule.Match) == "" {
			return RulesDoc{}, fmt.Errorf("第 %d 条规则缺少 match", i+1)
		}
		if strings.TrimSpace(rule.Icon) == "" {
			return RulesDoc{}, fmt.Errorf("第 %d 条规则缺少 icon", i+1)
		}
	}
	return doc, nil
}
