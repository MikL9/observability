package hide

import (
	"net/url"
	"strings"
	"sync/atomic"
	"unicode"
)

var defaultConverter atomic.Pointer[Converter]

func SetDefaultConverter(c *Converter) {
	defaultConverter.Store(c)
}

type Rule interface {
	Hide(val string) string
	Fields() []string
}

type rule struct {
	transform func(string) string
	pattern   []string
}

func (r *rule) Hide(val string) string {
	return r.transform(val)
}

func (r *rule) Fields() []string {
	return r.pattern
}

type Converter struct {
	fields map[string]Rule
}

func (c *Converter) Hide(key, val string) string {
	if rule, ok := c.fields[key]; ok {
		return rule.Hide(val)
	}
	return val
}

func NewConverter(rules ...Rule) *Converter {
	fields := make(map[string]Rule)
	for _, rule := range rules {
		for _, field := range rule.Fields() {
			if _, ok := fields[field]; ok {
				panic("logic error, multiple rules for field")
			}
			fields[field] = rule
		}
	}
	return &Converter{
		fields: fields,
	}
}

func maskName(name string) string {
	nameRune := []rune(name)
	nameLen := len([]rune(name))
	if nameLen > 2 {
		out := make([]rune, 0, nameLen)
		pos := 0
		for _, l := range nameRune {
			if !unicode.IsLetter(l) {
				out = append(out, l)
				pos = 0
				continue
			}
			if pos < 2 {
				out = append(out, l)
			} else {
				out = append(out, '*')
			}
			pos++
		}
		return string(out)
	}
	return fullExclude(name)
}

func maskCardAndPhone(phone string) string {
	phoneRune := []rune(phone)
	phoneLen := len(phoneRune)
	out := make([]rune, 0, phoneLen)
	for i, l := range phoneRune {
		isDigit := unicode.IsDigit(l)
		if !isDigit || i >= phoneLen-4 {
			out = append(out, l)
		} else {
			out = append(out, '*')
		}
	}
	return string(out)
}

func maskEmail(email string) string {
	tmp := strings.Split(email, "@")
	if len(tmp) == 1 {
		return fullExclude(email)
	}
	addr := tmp[0]
	domain := tmp[1]
	addrRune := []rune(addr)
	addrLen := len(addrRune)
	if addrLen < 4 {
		return strings.Repeat("*", addrLen) + "@" + domain
	}
	out := make([]rune, 0, len(addrRune))
	for i, l := range addrRune {
		if i < 3 {
			out = append(out, l)
		} else {
			out = append(out, '*')
		}
	}
	return string(out) + "@" + domain
}

func maskURL(s string) string {
	converter := defaultConverter.Load()
	if converter == nil {
		return s
	}
	if isParamsOnly := !strings.Contains(s, "?") && strings.Contains(s, "="); isParamsOnly {
		s = "?" + s
	}

	u, err := url.Parse(s)
	if err != nil {
		return fullExclude(s)
	}
	oldValues := u.Query()
	for key := range oldValues {
		oldValues.Set(key, converter.Hide(key, oldValues.Get(key)))
	}
	escapeQuery, _ := url.QueryUnescape(oldValues.Encode())
	u.RawQuery = escapeQuery
	return u.Redacted()
}

func fullExclude(s string) string {
	return strings.Repeat("*", len([]rune(s)))
}
