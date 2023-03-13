package helpers

import (
	"fmt"
	"sort"
	"strings"
)

func FlattenMap(jmap map[string]any) map[string]any {
	flattened := make(map[string]any)
	flatten(nil, jmap, flattened)
	return flattened
}

func flatten(key *string, obj map[string]any, ref map[string]any) {
	for k, v := range obj {
		var fullKey string
		if key == nil {
			fullKey = k
		} else {
			fullKey = fmt.Sprintf("%s.%s", *key, k)
		}
		set(&fullKey, v, ref)
	}
}
func set(key *string, obj any, ref map[string]any) {
	switch t := obj.(type) {
	case map[string]any:
		{
			flatten(key, t, ref)
		}
	case []any:
		{
			for index, item := range t {
				key := fmt.Sprintf("%s.{%d}", *key, index)
				set(&key, item, ref)
			}
		}
	default:
		{
			ref[*key] = obj
		}
	}
}

func UnFlatten(jo map[string]any) map[string]any {
	keys, mapKeys := Sort(jo)
	return unFlatten(jo, keys, mapKeys)
}

func unFlatten(jo map[string]any, keys []*[]string, keyMap map[*[]string]string) map[string]any {
	mapper := make(map[string]any)
	ref := &mapper
	for _, key := range keys {
		k := *key
		for i := 0; i < len(k); i++ {
			if i < len(k)-1 {
				r, ok := (*ref)[k[i]].(map[string]any)
				if !ok {
					r = make(map[string]any)
					(*ref)[k[i]] = r
				}
				ref = &r
			} else {
				(*ref)[k[i]] = jo[keyMap[key]]
			}
		}
		ref = &mapper
	}
	return mapper
}

func Sort(mapper map[string]interface{}) ([]*[]string, map[*[]string]string) {
	keys := make([]string, 0, len(mapper))
	for k := range mapper {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	output := make([]*[]string, 0, len(keys))
	keyMap := make(map[*[]string]string, len(keys))
	for _, key := range keys {
		sgmts := strings.Split(key, ".")
		output = append(output, &sgmts)
		keyMap[&sgmts] = key
	}
	return output, keyMap
}
