package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func dynamicJSONObject(value types.Dynamic) (*json.RawMessage, error) {
	if value.IsNull() {
		return nil, nil
	}
	if value.IsUnknown() {
		return nil, nil
	}
	if value.IsUnderlyingValueUnknown() {
		return nil, errors.New("value must be known before it can be sent to Coval")
	}

	underlying := value.UnderlyingValue()
	switch underlying.(type) {
	case types.Map, types.Object:
	default:
		return nil, fmt.Errorf("value must be an object, got %T", underlying)
	}

	goValue, err := terraformValueToJSON(underlying)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(goValue)
	if err != nil {
		return nil, fmt.Errorf("encode value as JSON: %w", err)
	}
	raw := json.RawMessage(encoded)
	return &raw, nil
}

func dynamicJSONObjectHasKey(value types.Dynamic, key string) (bool, error) {
	if value.IsNull() || value.IsUnknown() || value.IsUnderlyingValueUnknown() {
		return false, nil
	}

	switch object := value.UnderlyingValue().(type) {
	case types.Map:
		_, ok := object.Elements()[key]
		return ok, nil
	case types.Object:
		_, ok := object.Attributes()[key]
		return ok, nil
	default:
		return false, fmt.Errorf("value must be an object, got %T", value.UnderlyingValue())
	}
}

func jsonObjectWithoutKey(raw json.RawMessage, key string) (json.RawMessage, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return raw, nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, fmt.Errorf("decode Coval JSON object: %w", err)
	}
	delete(object, key)
	encoded, err := json.Marshal(object)
	if err != nil {
		return nil, fmt.Errorf("encode Coval JSON object: %w", err)
	}
	return json.RawMessage(encoded), nil
}

func dynamicJSONArray(value types.Dynamic) (*json.RawMessage, error) {
	if value.IsNull() {
		return nil, nil
	}
	if value.IsUnknown() {
		return nil, nil
	}
	if value.IsUnderlyingValueUnknown() {
		return nil, errors.New("value must be known before it can be sent to Coval")
	}

	underlying := value.UnderlyingValue()
	switch underlying.(type) {
	case types.List, types.Tuple:
	default:
		return nil, fmt.Errorf("value must be an array, got %T", underlying)
	}

	goValue, err := terraformValueToJSON(underlying)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(goValue)
	if err != nil {
		return nil, fmt.Errorf("encode value as JSON: %w", err)
	}
	raw := json.RawMessage(encoded)
	return &raw, nil
}

func dynamicJSONArrayLength(value types.Dynamic) (int, error) {
	if value.IsNull() || value.IsUnknown() || value.IsUnderlyingValueUnknown() {
		return 0, nil
	}

	switch array := value.UnderlyingValue().(type) {
	case types.List:
		return len(array.Elements()), nil
	case types.Tuple:
		return len(array.Elements()), nil
	default:
		return 0, fmt.Errorf("value must be an array, got %T", value.UnderlyingValue())
	}
}

func terraformValueToJSON(value attr.Value) (any, error) {
	if value == nil || value.IsNull() {
		return nil, nil
	}
	if value.IsUnknown() {
		return nil, errors.New("JSON values cannot contain unknown elements")
	}

	switch typed := value.(type) {
	case types.Dynamic:
		if typed.IsUnderlyingValueNull() {
			return nil, nil
		}
		return terraformValueToJSON(typed.UnderlyingValue())
	case types.String:
		return typed.ValueString(), nil
	case types.Bool:
		return typed.ValueBool(), nil
	case types.Number:
		return json.Number(typed.ValueBigFloat().Text('g', -1)), nil
	case types.Int64:
		return typed.ValueInt64(), nil
	case types.Float64:
		return typed.ValueFloat64(), nil
	case types.Map:
		result := make(map[string]any, len(typed.Elements()))
		for key, element := range typed.Elements() {
			converted, err := terraformValueToJSON(element)
			if err != nil {
				return nil, fmt.Errorf("convert object key %q: %w", key, err)
			}
			result[key] = converted
		}
		return result, nil
	case types.Object:
		result := make(map[string]any, len(typed.Attributes()))
		for key, element := range typed.Attributes() {
			converted, err := terraformValueToJSON(element)
			if err != nil {
				return nil, fmt.Errorf("convert object key %q: %w", key, err)
			}
			result[key] = converted
		}
		return result, nil
	case types.List:
		return terraformCollectionToJSON(typed.Elements())
	case types.Set:
		return terraformCollectionToJSON(typed.Elements())
	case types.Tuple:
		return terraformCollectionToJSON(typed.Elements())
	default:
		return nil, fmt.Errorf("unsupported Terraform JSON value type %T", value)
	}
}

func terraformCollectionToJSON(elements []attr.Value) ([]any, error) {
	result := make([]any, len(elements))
	for index, element := range elements {
		converted, err := terraformValueToJSON(element)
		if err != nil {
			return nil, fmt.Errorf("convert array element %d: %w", index, err)
		}
		result[index] = converted
	}
	return result, nil
}

func dynamicFromJSONObject(raw json.RawMessage) (types.Dynamic, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		raw = json.RawMessage(`{}`)
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return types.DynamicNull(), fmt.Errorf("decode Coval JSON value: %w", err)
	}
	if _, ok := decoded.(map[string]any); !ok {
		return types.DynamicNull(), fmt.Errorf("coval returned a non-object JSON value %T", decoded)
	}

	value, err := terraformValueFromJSON(decoded)
	if err != nil {
		return types.DynamicNull(), err
	}
	return types.DynamicValue(value), nil
}

func dynamicFromNullableJSONObject(raw json.RawMessage) (types.Dynamic, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return types.DynamicNull(), nil
	}
	return dynamicFromJSONObject(raw)
}

func dynamicFromJSONObjectPreserving(raw json.RawMessage, prior types.Dynamic) (types.Dynamic, error) {
	if !prior.IsNull() && !prior.IsUnknown() && !prior.IsUnderlyingValueUnknown() {
		priorRaw, err := dynamicJSONObject(prior)
		if err == nil && priorRaw != nil && jsonValuesEqual(*priorRaw, raw) {
			return prior, nil
		}
	}
	return dynamicFromJSONObject(raw)
}

func dynamicFromJSONArray(raw json.RawMessage) (types.Dynamic, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return types.DynamicNull(), nil
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return types.DynamicNull(), fmt.Errorf("decode Coval JSON value: %w", err)
	}
	if _, ok := decoded.([]any); !ok {
		return types.DynamicNull(), fmt.Errorf("coval returned a non-array JSON value %T", decoded)
	}

	value, err := terraformValueFromJSON(decoded)
	if err != nil {
		return types.DynamicNull(), err
	}
	return types.DynamicValue(value), nil
}

func dynamicFromJSONArrayPreserving(raw json.RawMessage, prior types.Dynamic) (types.Dynamic, error) {
	if !prior.IsNull() && !prior.IsUnknown() && !prior.IsUnderlyingValueUnknown() {
		priorRaw, err := dynamicJSONArray(prior)
		if err == nil && priorRaw != nil {
			if jsonValuesEqual(*priorRaw, raw) {
				return prior, nil
			}
			if (len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null"))) && jsonValuesEqual(*priorRaw, json.RawMessage(`[]`)) {
				return prior, nil
			}
		}
	}
	return dynamicFromJSONArray(raw)
}

func jsonValuesEqual(left json.RawMessage, right json.RawMessage) bool {
	decode := func(raw json.RawMessage) (any, error) {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var value any
		err := decoder.Decode(&value)
		return value, err
	}
	leftValue, leftErr := decode(left)
	rightValue, rightErr := decode(right)
	return leftErr == nil && rightErr == nil && reflect.DeepEqual(leftValue, rightValue)
}

func terraformValueFromJSON(value any) (attr.Value, error) {
	switch typed := value.(type) {
	case nil:
		return types.DynamicNull(), nil
	case string:
		return types.StringValue(typed), nil
	case bool:
		return types.BoolValue(typed), nil
	case json.Number:
		number, _, err := big.ParseFloat(typed.String(), 10, 512, big.ToNearestEven)
		if err != nil {
			return nil, fmt.Errorf("decode JSON number %q: %w", typed, err)
		}
		return types.NumberValue(number), nil
	case float64:
		return types.NumberValue(big.NewFloat(typed)), nil
	case map[string]any:
		attributeTypes := make(map[string]attr.Type, len(typed))
		attributes := make(map[string]attr.Value, len(typed))
		for key, item := range typed {
			converted, err := terraformValueFromJSON(item)
			if err != nil {
				return nil, fmt.Errorf("convert JSON object key %q: %w", key, err)
			}
			attributeTypes[key] = converted.Type(context.Background())
			attributes[key] = converted
		}
		object, diagnostics := types.ObjectValue(attributeTypes, attributes)
		if diagnostics.HasError() {
			return nil, fmt.Errorf("convert JSON object to Terraform: %v", diagnostics)
		}
		return object, nil
	case []any:
		elementTypes := make([]attr.Type, len(typed))
		elements := make([]attr.Value, len(typed))
		for index, item := range typed {
			converted, err := terraformValueFromJSON(item)
			if err != nil {
				return nil, fmt.Errorf("convert JSON array element %d: %w", index, err)
			}
			elementTypes[index] = converted.Type(context.Background())
			elements[index] = converted
		}
		tuple, diagnostics := types.TupleValue(elementTypes, elements)
		if diagnostics.HasError() {
			return nil, fmt.Errorf("convert JSON array to Terraform: %v", diagnostics)
		}
		return tuple, nil
	default:
		return nil, fmt.Errorf("unsupported JSON value type %T", value)
	}
}
