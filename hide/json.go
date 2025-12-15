package hide

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/pkg/errors"
)

// JSON сериализует тело
func JSON(key string, val []byte, limit int, textAttr bool) slog.Attr {
	stringKey := fmt.Sprintf("%s_text", key) // на случай если невозможно превратить JSON в Group
	if len(val) < limit {
		limit = len(val)
	}
	maskVal, err := MaskSensitiveJSONFields(val[:limit])

	if maskVal == "" {
		maskVal = string(val[:limit])
	}
	if err != nil || textAttr {
		return slog.String(stringKey, maskVal)
	}

	var mapVal map[string]any
	if err := json.Unmarshal([]byte(maskVal), &mapVal); err != nil {
		return slog.String(stringKey, maskVal)
	}

	args := make([]any, 0, len(mapVal))
	for key, val := range mapVal {
		args = append(args, key, val)
	}

	return slog.Group(key, args...)
}

const (
	unknown = iota
	object
	objectEnd
	fieldName
	fieldValue
	array
	arrayValue
	arrayEnd
)

// MaskSensitiveJSONFields маскирует приватные данные в JSON-строках.
func MaskSensitiveJSONFields(body []byte) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	strBuilder := new(strings.Builder)

	var (
		token         json.Token
		stateStack    intStack
		lastFieldName string
		err           error
	)

	state := unknown
	beforeNextValue := func() error {
		switch state {
		case fieldValue:
			if err := strBuilder.WriteByte(','); err != nil {
				return fmt.Errorf("logger: json mask encoder: %w", err)
			}

			state = fieldName
		case fieldName:
			if err := strBuilder.WriteByte(':'); err != nil {
				return fmt.Errorf("logger: json mask encoder: %w", err)
			}

			state = fieldValue
		case arrayValue:
			if err := strBuilder.WriteByte(','); err != nil {
				return fmt.Errorf("logger: json mask encoder: %w", err)
			}
		case array:
			state = arrayValue
		case object:
			state = fieldName
		}

		return nil
	}

	pushState := func() error {
		switch state {
		case object, fieldValue:
			stateStack = stateStack.push(fieldValue)
		case array, arrayValue:
			stateStack = stateStack.push(arrayValue)
		case unknown:
			stateStack = stateStack.push(unknown)
		default:
			return fmt.Errorf("logger: json mask encoder: invalid state (json is invalid?): %d", state)
		}

		return nil
	}

	setState := func(newState int) error {
		switch newState {
		case object:
			if err := beforeNextValue(); err != nil {
				return err
			}

			if err := strBuilder.WriteByte('{'); err != nil {
				return fmt.Errorf("logger: json mask encoder: %w", err)
			}

			if err := pushState(); err != nil {
				return err
			}

			state = newState
		case objectEnd:
			if err := strBuilder.WriteByte('}'); err != nil {
				return fmt.Errorf("logger: json mask encoder: %w", err)
			}

			stateStack, state = stateStack.pop()
		case array:
			if err := beforeNextValue(); err != nil {
				return err
			}

			if err := strBuilder.WriteByte('['); err != nil {
				return fmt.Errorf("logger: json mask encoder: %w", err)
			}

			if err := pushState(); err != nil {
				return err
			}

			state = newState
		case arrayEnd:
			if err := strBuilder.WriteByte(']'); err != nil {
				return fmt.Errorf("logger: json mask encoder: %w", err)
			}

			stateStack, state = stateStack.pop()
		}

		return nil
	}

	for {
		token, err = decoder.Token()
		if err != nil {
			break
		}

		if v, ok := token.(json.Delim); ok {
			switch v.String() {
			case "{":
				if err := setState(object); err != nil {
					return "", err
				}
			case "}":
				if err := setState(objectEnd); err != nil {
					return "", err
				}
			case "[":
				if err := setState(array); err != nil {
					return "", err
				}
			case "]":
				if err := setState(arrayEnd); err != nil {
					return "", err
				}
			}
		} else {
			if err := beforeNextValue(); err != nil {
				return "", err
			}
			var jsonBytes []byte
			switch dataType := token.(type) {
			case string:
				currentFieldName := ""
				if state == fieldName {
					lastFieldName = dataType
				} else if state == fieldValue {
					currentFieldName = lastFieldName
				}
				jsonBytes, err = json.Marshal(Hide(currentFieldName, dataType))
			default:
				jsonBytes, err = json.Marshal(dataType)
			}
			if err != nil {
				return "", fmt.Errorf("logger: json mask encoder: %w", err)
			}
			if _, err = strBuilder.Write(jsonBytes); err != nil {
				return "", fmt.Errorf("logger: json mask encoder: %w", err)
			}
		}
	}

	if errors.Is(err, io.EOF) {
		err = nil
	}
	return strBuilder.String(), err
}

type intStack []int

func (s intStack) push(v int) intStack {
	return append(s, v)
}

func (s intStack) pop() (intStack, int) {
	if len(s) == 0 {
		return s, 0
	}

	l := len(s)

	return s[:l-1], s[l-1]
}
