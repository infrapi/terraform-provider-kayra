package provider

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// body is the request body of v, a plan: every field but the read-only
// ones (Kayra's contracts refuse them in a request) and the path's, a
// field Terraform does not know yet (null, unknown) left out.
func body(fields []field, v tftypes.Value) (map[string]any, error) {
	attrs, err := objectAttrs(v)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for _, f := range fields {
		if f.ReadOnly || f.PathOnly {
			continue
		}
		a, ok := attrs[f.Attr]
		if !ok || !a.IsKnown() || a.IsNull() {
			continue
		}
		j, err := toJSON(f, a)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Attr, err)
		}
		out[f.name()] = j
	}
	return out, nil
}

func toJSON(f field, v tftypes.Value) (any, error) {
	switch f.Kind {
	case kString:
		var s string
		return s, v.As(&s)
	case kBool:
		var b bool
		return b, v.As(&b)
	case kInt:
		var n big.Float
		if err := v.As(&n); err != nil {
			return nil, err
		}
		i, _ := n.Int64()
		return i, nil
	case kStrings:
		var elems []tftypes.Value
		if err := v.As(&elems); err != nil {
			return nil, err
		}
		out := []string{}
		for _, e := range elems {
			var s string
			if err := e.As(&s); err != nil {
				return nil, err
			}
			out = append(out, s)
		}
		return out, nil
	case kStringMap:
		var elems map[string]tftypes.Value
		if err := v.As(&elems); err != nil {
			return nil, err
		}
		out := map[string]string{}
		for k, e := range elems {
			var s string
			if err := e.As(&s); err != nil {
				return nil, err
			}
			out[k] = s
		}
		return out, nil
	case kObject:
		return body(f.Fields, v)
	case kObjects:
		var elems []tftypes.Value
		if err := v.As(&elems); err != nil {
			return nil, err
		}
		out := []any{}
		for _, e := range elems {
			o, err := body(f.Fields, e)
			if err != nil {
				return nil, err
			}
			out = append(out, o)
		}
		return out, nil
	}
	return nil, fmt.Errorf("kind %d", f.Kind)
}

// objectAttrs are v's attributes; none when it is null or unknown.
func objectAttrs(v tftypes.Value) (map[string]tftypes.Value, error) {
	attrs := map[string]tftypes.Value{}
	if !v.IsKnown() || v.IsNull() {
		return attrs, nil
	}
	return attrs, v.As(&attrs)
}

// state is the object Kayra answered, m, as the attributes of fields, for
// a state: prior is what Terraform had, the plan or the state.
//
// Kayra leaves out a property at its zero value: an attribute prior has
// null stays null, one it has at that zero value keeps it, so the state is
// the configuration's when Kayra stores it. A write-only attribute, or a
// path's, keeps prior's; so does a sensitive read-only one Kayra returns
// once, a token's secret.
func state(fields []field, m map[string]any, prior tftypes.Value) (tftypes.Value, error) {
	priorAttrs, err := objectAttrs(prior)
	if err != nil {
		return tftypes.Value{}, err
	}
	vals := map[string]tftypes.Value{}
	for _, f := range fields {
		p, ok := priorAttrs[f.Attr]
		if !ok || !p.IsKnown() {
			p = tftypes.NewValue(tfType(f), nil)
		}
		raw, present := m[f.name()]
		switch {
		case f.WriteOnly || f.PathOnly:
			vals[f.Attr] = p
			continue
		case (!present || raw == nil) && f.ReadOnly && f.Sensitive && !p.IsNull():
			vals[f.Attr] = p
			continue
		}
		v, err := fromJSON(f, raw, p)
		if err != nil {
			return tftypes.Value{}, fmt.Errorf("%s: %w", f.Attr, err)
		}
		vals[f.Attr] = v
	}
	return tftypes.NewValue(objectType(fields), vals), nil
}

func fromJSON(f field, raw any, prior tftypes.Value) (tftypes.Value, error) {
	t := tfType(f)
	null := tftypes.NewValue(t, nil)
	if isZero(raw) {
		switch {
		case f.Kind == kBool && f.ReadOnly:
			return tftypes.NewValue(t, false), nil
		case prior.IsNull():
			return null, nil
		}
	}
	switch f.Kind {
	case kString:
		s, _ := raw.(string)
		if raw != nil {
			if _, ok := raw.(string); !ok {
				return null, fmt.Errorf("%v: not a string", raw)
			}
		}
		return tftypes.NewValue(t, s), nil
	case kBool:
		b, _ := raw.(bool)
		return tftypes.NewValue(t, b), nil
	case kInt:
		var n float64
		switch x := raw.(type) {
		case float64:
			n = x
		case json.Number:
			n, _ = x.Float64()
		}
		return tftypes.NewValue(t, new(big.Float).SetFloat64(n)), nil
	case kStrings:
		list, _ := raw.([]any)
		elems := []tftypes.Value{}
		for _, e := range list {
			s, ok := e.(string)
			if !ok {
				return null, fmt.Errorf("%v: not a string", e)
			}
			elems = append(elems, tftypes.NewValue(tftypes.String, s))
		}
		return tftypes.NewValue(t, elems), nil
	case kStringMap:
		obj, _ := raw.(map[string]any)
		elems := map[string]tftypes.Value{}
		for k, e := range obj {
			s, ok := e.(string)
			if !ok {
				return null, fmt.Errorf("%s: %v: not a string", k, e)
			}
			elems[k] = tftypes.NewValue(tftypes.String, s)
		}
		return tftypes.NewValue(t, elems), nil
	case kObject:
		obj, _ := raw.(map[string]any)
		return state(f.Fields, obj, prior)
	case kObjects:
		list, _ := raw.([]any)
		var priorElems []tftypes.Value
		if prior.IsKnown() && !prior.IsNull() {
			if err := prior.As(&priorElems); err != nil {
				return null, err
			}
		}
		elems := []tftypes.Value{}
		for i, e := range list {
			obj, ok := e.(map[string]any)
			if !ok {
				return null, fmt.Errorf("%v: not an object", e)
			}
			p := tftypes.NewValue(objectType(f.Fields), nil)
			if i < len(priorElems) {
				p = priorElems[i]
			}
			v, err := state(f.Fields, obj, p)
			if err != nil {
				return null, err
			}
			elems = append(elems, v)
		}
		return tftypes.NewValue(t, elems), nil
	}
	return null, fmt.Errorf("kind %d", f.Kind)
}

// isZero tells a value Kayra would leave out: absent, "", false, 0, an
// empty list or an object of zero values only.
func isZero(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case bool:
		return !x
	case float64:
		return x == 0
	case []any:
		return len(x) == 0
	case map[string]any:
		for _, e := range x {
			if !isZero(e) {
				return false
			}
		}
		return true
	}
	return false
}

// keysID is d's import id of v: its keys' values joined by /.
func keysID(keys []string, attrs map[string]tftypes.Value) (string, error) {
	id := ""
	for i, k := range keys {
		var s string
		a, ok := attrs[k]
		if !ok || !a.IsKnown() || a.IsNull() {
			return "", fmt.Errorf("%s is not known", k)
		}
		if err := a.As(&s); err != nil {
			return "", err
		}
		if i > 0 {
			id += "/"
		}
		id += s
	}
	return id, nil
}
